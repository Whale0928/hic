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

type Collector struct {
	fetcher AttachmentFetcher
	store   global.ObjectStore
}

func NewCollector(fetcher AttachmentFetcher, store global.ObjectStore) Collector {
	return Collector{fetcher: fetcher, store: store}
}

type AttachmentObject struct {
	Seq          string
	FileSeq      string
	Filename     string
	Kind         extraction.AttachmentKind
	StoredObject global.StoredObject
}

type PreserveReport struct {
	Agency     string
	BoardKind  string
	Seq        string
	Downloaded int
	Objects    []AttachmentObject
}

func (r PreserveReport) String() string {
	return fmt.Sprintf("agency=%s board=%s seq=%s downloaded=%d objects=%d", r.Agency, r.BoardKind, r.Seq, r.Downloaded, len(r.Objects))
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

		report.Downloaded++
		report.Objects = append(report.Objects, AttachmentObject{
			Seq:          attachment.Seq,
			FileSeq:      attachment.FileSeq,
			Filename:     filename,
			Kind:         kind,
			StoredObject: stored,
		})
	}

	return report, nil
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
