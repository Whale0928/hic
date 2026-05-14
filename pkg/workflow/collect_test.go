package workflow

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"hic/pkg/discovery"
	"hic/pkg/extraction"
)

type fakeAttachmentFetcher struct {
	body        []byte
	contentType string
}

func (f fakeAttachmentFetcher) FetchAttachment(ctx context.Context, board discovery.Board, attachment discovery.AttachmentMeta) (AttachmentDocument, error) {
	return AttachmentDocument{
		ContentType: f.contentType,
		Body:        io.NopCloser(bytes.NewReader(f.body)),
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
