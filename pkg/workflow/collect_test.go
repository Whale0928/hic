package workflow

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hic/pkg/discovery"
	"hic/pkg/extraction"
	"hic/pkg/global"
)

type fakeAttachmentFetcher struct {
	body        []byte
	contentType string
}

func TestSHAttachmentFetcher_FetchAttachmentPreview_미리보기HTML을가져온다(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>입주자 모집공고</body></html>"))
	}))
	defer server.Close()

	fetcher := &SHAttachmentFetcher{httpClient: server.Client()}
	doc, err := fetcher.FetchAttachmentPreview(context.Background(), discovery.Board{BaseURL: server.URL}, discovery.AttachmentMeta{
		PreviewPath: "/app/com/util/htmlConverter.do?brd_id=GS0401&seq=304295&data_tp=A&file_seq=1",
	})
	if err != nil {
		t.Fatalf("FetchAttachmentPreview() error = %v", err)
	}
	defer doc.Body.Close()
	body, err := io.ReadAll(doc.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(body), "입주자 모집공고") {
		t.Fatalf("body = %q", string(body))
	}
	if gotPath != "/app/com/util/htmlConverter.do?brd_id=GS0401&seq=304295&data_tp=A&file_seq=1" {
		t.Fatalf("gotPath = %q", gotPath)
	}
}

func (f fakeAttachmentFetcher) FetchAttachment(ctx context.Context, board discovery.Board, attachment discovery.AttachmentMeta) (AttachmentDocument, error) {
	return AttachmentDocument{
		ContentType: f.contentType,
		Body:        io.NopCloser(bytes.NewReader(f.body)),
	}, nil
}

type fakeAttachmentPreviewFetcher struct {
	fakeAttachmentFetcher
	previewBody        []byte
	previewContentType string
}

func (f fakeAttachmentPreviewFetcher) FetchAttachmentPreview(ctx context.Context, board discovery.Board, attachment discovery.AttachmentMeta) (AttachmentDocument, error) {
	return AttachmentDocument{
		ContentType: f.previewContentType,
		Body:        io.NopCloser(bytes.NewReader(f.previewBody)),
	}, nil
}

func TestCollector_PreserveCandidateAttachments_ObjectStore에원본을보존한다(t *testing.T) {
	store := extraction.NewLocalObjectStore(t.TempDir())
	collector := NewCollector(fakeAttachmentFetcher{
		body:        []byte("pdf body"),
		contentType: "application/pdf",
	}, store)
	board := discovery.Board{Agency: "SH", BoardKind: "rental", BaseURL: "https://www.i-sh.co.kr"}
	candidate := discovery.Candidate{
		Agency:    "SH",
		BoardKind: "rental",
		Seq:       "304295",
		Title:     "테스트 모집공고",
		Attachments: []discovery.AttachmentMeta{
			{BRDID: "GS0401", Seq: "304295", FileSeq: "1", Filename: "[공고문] 테스트.pdf", FileType: "A"},
		},
	}

	report, err := collector.PreserveCandidateAttachments(context.Background(), board, candidate)
	if err != nil {
		t.Fatalf("PreserveCandidateAttachments() error = %v", err)
	}

	if report.Downloaded != 1 || len(report.Objects) != 1 {
		t.Fatalf("report = %+v", report)
	}
	object := report.Objects[0]
	if object.Kind != extraction.AttachmentKindNoticePDF {
		t.Fatalf("Kind = %q, want %q", object.Kind, extraction.AttachmentKindNoticePDF)
	}
	if object.StoredObject.SizeBytes != int64(len("pdf body")) || object.StoredObject.SHA256 == "" {
		t.Fatalf("StoredObject = %+v", object.StoredObject)
	}
	if !strings.HasPrefix(object.StoredObject.Key, "hic-originals/sh/304295/1-") {
		t.Fatalf("object key = %q", object.StoredObject.Key)
	}

	rc, err := store.Get(context.Background(), object.StoredObject.Key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(got) != "pdf body" {
		t.Fatalf("stored body = %q", string(got))
	}
}

func TestCollector_PreserveCandidateAttachments_HTMLPreview를보존한다(t *testing.T) {
	store := extraction.NewLocalObjectStore(t.TempDir())
	collector := NewCollector(fakeAttachmentPreviewFetcher{
		fakeAttachmentFetcher: fakeAttachmentFetcher{
			body:        []byte("pdf body"),
			contentType: "application/pdf",
		},
		previewBody:        []byte("<html><body>입주자 모집공고</body></html>"),
		previewContentType: "text/html",
	}, store)
	board := discovery.Board{Agency: "SH", BoardKind: "rental", BaseURL: "https://www.i-sh.co.kr"}
	candidate := discovery.Candidate{
		Agency:    "SH",
		BoardKind: "rental",
		Seq:       "304295",
		Title:     "테스트 모집공고",
		Attachments: []discovery.AttachmentMeta{
			{BRDID: "GS0401", Seq: "304295", FileSeq: "1", Filename: "[공고문] 테스트.pdf", FileType: "A", PreviewPath: "/app/com/util/htmlConverter.do?brd_id=GS0401&seq=304295&data_tp=A&file_seq=1"},
		},
	}

	report, err := collector.PreserveCandidateAttachments(context.Background(), board, candidate)
	if err != nil {
		t.Fatalf("PreserveCandidateAttachments() error = %v", err)
	}

	if report.Previewed != 1 {
		t.Fatalf("Previewed = %d, want 1", report.Previewed)
	}
	if len(report.Objects) != 1 || report.Objects[0].PreviewStoredObject == nil {
		t.Fatalf("preview object missing: %+v", report.Objects)
	}
	preview := report.Objects[0].PreviewStoredObject
	if preview.ContentType != "text/html" || !strings.HasPrefix(preview.Key, "hic-artifacts/sh/304295/1-preview") {
		t.Fatalf("preview object = %+v", preview)
	}
	rc, err := store.Get(context.Background(), preview.Key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(got), "입주자 모집공고") {
		t.Fatalf("preview body = %q", string(got))
	}
}

func TestObjectKeyForAttachment_파일명을안전하게정규화한다(t *testing.T) {
	key := ObjectKeyForAttachment(discovery.Candidate{Agency: "SH", Seq: "304295"}, discovery.AttachmentMeta{
		FileSeq:  "1",
		Filename: "../위험?.pdf",
	})

	if strings.Contains(key, "..") || strings.Contains(key, "?") {
		t.Fatalf("unsafe key = %q", key)
	}
	if key != "hic-originals/sh/304295/1-위험_.pdf" {
		t.Fatalf("key = %q", key)
	}
}

func TestExtractPreservedAttachment_HWP모집공고는UnsupportedArtifact를남긴다(t *testing.T) {
	store := extraction.NewLocalObjectStore(t.TempDir())
	stored, err := store.Put(context.Background(), global.Object{
		Key:          "hic-originals/sh/304295/1-notice.hwp",
		Reader:       strings.NewReader("hwp body"),
		ContentType:  "application/x-hwp",
		OriginalName: "모집공고.hwp",
	})
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	artifacts, err := ExtractPreservedAttachment(store, PreservedAttachmentRef{
		ObjectKey: stored.Key,
		Kind:      extraction.AttachmentKindNoticeHWP,
	})
	if err != nil {
		t.Fatalf("extractPreservedAttachment() error = %v", err)
	}

	if len(artifacts) != 1 {
		t.Fatalf("len(artifacts) = %d, want 1", len(artifacts))
	}
	if artifacts[0].Type != extraction.ArtifactTypeHWPUnsupported || artifacts[0].Status != extraction.ArtifactStatusUnsupported {
		t.Fatalf("artifact = %+v", artifacts[0])
	}
	if artifacts[0].SourceSpan != "object://hic-originals/sh/304295/1-notice.hwp" {
		t.Fatalf("SourceSpan = %q", artifacts[0].SourceSpan)
	}
}

func TestExtractPreservedAttachment_HWPX모집공고는텍스트Artifact를남긴다(t *testing.T) {
	root := t.TempDir()
	store := extraction.NewLocalObjectStore(root)
	key := "hic-originals/sh/304295/1-notice.hwpx"
	path, err := store.PathForKey(key)
	if err != nil {
		t.Fatalf("PathForKey() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	writeWorkflowTestHWPX(t, path, "입주자 모집공고")

	artifacts, err := ExtractPreservedAttachment(store, PreservedAttachmentRef{
		ObjectKey: key,
		Filename:  "모집공고.hwpx",
		Kind:      extraction.AttachmentKindNoticeHWP,
	})
	if err != nil {
		t.Fatalf("ExtractPreservedAttachment() error = %v", err)
	}

	if len(artifacts) != 1 || artifacts[0].Type != extraction.ArtifactTypeHWPXText {
		t.Fatalf("artifacts = %+v", artifacts)
	}
	if artifacts[0].SourceSpan != "object://hic-originals/sh/304295/1-notice.hwpx" {
		t.Fatalf("SourceSpan = %q", artifacts[0].SourceSpan)
	}
}

func TestExtractPreservedPreview_HTMLPreviewArtifact를생성한다(t *testing.T) {
	store := extraction.NewLocalObjectStore(t.TempDir())
	stored, err := store.Put(context.Background(), global.Object{
		Key:          "hic-artifacts/sh/304295/1-preview.html",
		Reader:       strings.NewReader("<html><body>입주자 모집공고</body></html>"),
		ContentType:  "text/html",
		OriginalName: "preview.html",
	})
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	artifacts, err := ExtractPreservedPreview(store, stored.Key)
	if err != nil {
		t.Fatalf("ExtractPreservedPreview() error = %v", err)
	}

	if len(artifacts) != 1 || artifacts[0].Type != extraction.ArtifactTypeHTMLPreview {
		t.Fatalf("artifacts = %+v", artifacts)
	}
	if artifacts[0].SourceSpan != "object://hic-artifacts/sh/304295/1-preview.html" {
		t.Fatalf("SourceSpan = %q", artifacts[0].SourceSpan)
	}
}

func writeWorkflowTestHWPX(t *testing.T, path string, text string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("Contents/section0.xml")
	if err != nil {
		t.Fatalf("zip Create() error = %v", err)
	}
	_, err = w.Write([]byte(`<hp:sec xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph"><hp:p><hp:run><hp:t>` + text + `</hp:t></hp:run></hp:p></hp:sec>`))
	if err != nil {
		t.Fatalf("zip Write() error = %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip Close() error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
