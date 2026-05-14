package workflow

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"hic/pkg/discovery"
)

type SHAttachmentFetcher struct {
	httpClient *http.Client
}

func NewSHAttachmentFetcher() *SHAttachmentFetcher {
	return &SHAttachmentFetcher{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *SHAttachmentFetcher) FetchAttachment(ctx context.Context, board discovery.Board, attachment discovery.AttachmentMeta) (AttachmentDocument, error) {
	base, err := url.Parse(board.BaseURL)
	if err != nil {
		return AttachmentDocument{}, fmt.Errorf("parse board base URL: %w", err)
	}
	u, err := base.Parse("/app/com/file/innoFD.do")
	if err != nil {
		return AttachmentDocument{}, fmt.Errorf("parse attachment path: %w", err)
	}
	values := u.Query()
	values.Set("brdId", attachment.BRDID)
	values.Set("seq", attachment.Seq)
	values.Set("fileTp", firstNonEmpty(attachment.FileType, "A"))
	values.Set("fileSeq", attachment.FileSeq)
	u.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return AttachmentDocument{}, err
	}
	req.Header.Set("User-Agent", "hic-workflow/0.1 (+public housing attachment preservation)")
	req.Header.Set("Accept", "*/*")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return AttachmentDocument{}, fmt.Errorf("download attachment: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return AttachmentDocument{}, fmt.Errorf("download attachment: unexpected status %d", resp.StatusCode)
	}

	return AttachmentDocument{
		ContentType: resp.Header.Get("Content-Type"),
		Body:        resp.Body,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
