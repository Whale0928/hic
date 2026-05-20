package workflow

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"hic/pkg/discovery"
	"hic/pkg/extraction"
	"hic/pkg/global"
)

type AttachmentDocument struct {
	ContentType string
	Body        io.ReadCloser
}

type AttachmentFetcher interface {
	FetchAttachment(ctx context.Context, board discovery.Board, attachment discovery.AttachmentMeta) (AttachmentDocument, error)
}

type AttachmentPreviewFetcher interface {
	FetchAttachmentPreview(ctx context.Context, board discovery.Board, attachment discovery.AttachmentMeta) (AttachmentDocument, error)
}

type Collector struct {
	fetcher AttachmentFetcher
	store   global.ObjectStore
}

func NewCollector(fetcher AttachmentFetcher, store global.ObjectStore) Collector {
	return Collector{fetcher: fetcher, store: store}
}

type AttachmentObject struct {
	Seq                 string
	FileSeq             string
	Filename            string
	Kind                extraction.AttachmentKind
	StoredObject        global.StoredObject
	PreviewStoredObject *global.StoredObject
}

type PreserveReport struct {
	Agency     string
	BoardKind  string
	Seq        string
	Downloaded int
	Previewed  int
	Objects    []AttachmentObject
}

func (r PreserveReport) String() string {
	return fmt.Sprintf("agency=%s board=%s seq=%s downloaded=%d previews=%d objects=%d", r.Agency, r.BoardKind, r.Seq, r.Downloaded, r.Previewed, len(r.Objects))
}

func (c Collector) PreserveCandidateAttachments(ctx context.Context, board discovery.Board, candidate discovery.Candidate) (PreserveReport, error) {
	report := PreserveReport{
		Agency:    candidate.Agency,
		BoardKind: candidate.BoardKind,
		Seq:       candidate.Seq,
	}

	for _, attachment := range candidate.Attachments {
		doc, err := c.fetcher.FetchAttachment(ctx, board, attachment)
		if err != nil {
			return report, err
		}
		if doc.Body == nil {
			return report, fmt.Errorf("attachment body is required: seq=%s file_seq=%s", attachment.Seq, attachment.FileSeq)
		}

		filename := attachment.DisplayFilename()
		kind := extraction.ClassifyAttachment(filename)
		stored, err := c.store.Put(ctx, global.Object{
			Key:          ObjectKeyForAttachment(candidate, attachment),
			Reader:       doc.Body,
			ContentType:  doc.ContentType,
			OriginalName: filename,
			Metadata: map[string]string{
				"agency":          candidate.Agency,
				"board_kind":      candidate.BoardKind,
				"seq":             candidate.Seq,
				"file_seq":        attachment.FileSeq,
				"attachment_kind": string(kind),
			},
		})
		closeErr := doc.Body.Close()
		if err != nil {
			return report, err
		}
		if closeErr != nil {
			return report, fmt.Errorf("close attachment body: %w", closeErr)
		}

		attachmentObject := AttachmentObject{
			Seq:          attachment.Seq,
			FileSeq:      attachment.FileSeq,
			Filename:     filename,
			Kind:         kind,
			StoredObject: stored,
		}
		if preview, ok, err := c.preserveAttachmentPreview(ctx, board, candidate, attachment); err != nil {
			return report, err
		} else if ok {
			report.Previewed++
			attachmentObject.PreviewStoredObject = &preview
		}

		report.Downloaded++
		report.Objects = append(report.Objects, attachmentObject)
	}

	return report, nil
}

func (c Collector) preserveAttachmentPreview(ctx context.Context, board discovery.Board, candidate discovery.Candidate, attachment discovery.AttachmentMeta) (global.StoredObject, bool, error) {
	if strings.TrimSpace(attachment.PreviewPath) == "" {
		return global.StoredObject{}, false, nil
	}
	fetcher, ok := c.fetcher.(AttachmentPreviewFetcher)
	if !ok {
		return global.StoredObject{}, false, nil
	}
	doc, err := fetcher.FetchAttachmentPreview(ctx, board, attachment)
	if err != nil {
		return global.StoredObject{}, false, err
	}
	if doc.Body == nil {
		return global.StoredObject{}, false, fmt.Errorf("attachment preview body is required: seq=%s file_seq=%s", attachment.Seq, attachment.FileSeq)
	}
	stored, err := c.store.Put(ctx, global.Object{
		Key:          ObjectKeyForAttachmentPreview(candidate, attachment),
		Reader:       doc.Body,
		ContentType:  firstNonEmptyString(doc.ContentType, "text/html"),
		OriginalName: sanitizeObjectFilename(attachment.DisplayFilename()) + ".preview.html",
		Metadata: map[string]string{
			"agency":          candidate.Agency,
			"board_kind":      candidate.BoardKind,
			"seq":             candidate.Seq,
			"file_seq":        attachment.FileSeq,
			"artifact_type":   string(extraction.ArtifactTypeHTMLPreview),
			"source_preview":  attachment.PreviewPath,
			"attachment_kind": string(extraction.ClassifyAttachment(attachment.DisplayFilename())),
		},
	})
	closeErr := doc.Body.Close()
	if err != nil {
		return global.StoredObject{}, false, err
	}
	if closeErr != nil {
		return global.StoredObject{}, false, fmt.Errorf("close attachment preview body: %w", closeErr)
	}
	return stored, true, nil
}

var unsafeObjectKeyPattern = regexp.MustCompile(`[^0-9A-Za-z가-힣._() \-\[\]]+`)

func ObjectKeyForAttachment(candidate discovery.Candidate, attachment discovery.AttachmentMeta) string {
	agency := strings.ToLower(strings.TrimSpace(candidate.Agency))
	if agency == "" {
		agency = "unknown"
	}
	seq := strings.TrimSpace(candidate.Seq)
	if seq == "" {
		seq = strings.TrimSpace(attachment.Seq)
	}
	if seq == "" {
		seq = "unknown"
	}
	fileSeq := strings.TrimSpace(attachment.FileSeq)
	if fileSeq == "" {
		fileSeq = "0"
	}
	filename := sanitizeObjectFilename(attachment.DisplayFilename())
	return filepath.ToSlash(filepath.Join("hic-originals", agency, seq, fileSeq+"-"+filename))
}

func ObjectKeyForAttachmentPreview(candidate discovery.Candidate, attachment discovery.AttachmentMeta) string {
	agency := strings.ToLower(strings.TrimSpace(candidate.Agency))
	if agency == "" {
		agency = "unknown"
	}
	seq := strings.TrimSpace(candidate.Seq)
	if seq == "" {
		seq = strings.TrimSpace(attachment.Seq)
	}
	if seq == "" {
		seq = "unknown"
	}
	fileSeq := strings.TrimSpace(attachment.FileSeq)
	if fileSeq == "" {
		fileSeq = "0"
	}
	return filepath.ToSlash(filepath.Join("hic-artifacts", agency, seq, fileSeq+"-preview.html"))
}

func sanitizeObjectFilename(filename string) string {
	filename = strings.TrimSpace(filepath.Base(filename))
	if filename == "." || filename == string(filepath.Separator) || filename == "" {
		filename = "attachment.bin"
	}
	filename = unsafeObjectKeyPattern.ReplaceAllString(filename, "_")
	if len([]rune(filename)) > 180 {
		filename = string([]rune(filename)[:180])
	}
	return filename
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
