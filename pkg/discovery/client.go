package discovery

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type HTTPDocument struct {
	URL         string
	Method      string
	RequestBody string
	ContentType string
	Body        []byte
	FetchedAt   time.Time
}

type Fetcher interface {
	FetchBoardList(ctx context.Context, board Board, page int) (HTTPDocument, error)
	FetchBoardDetail(ctx context.Context, board Board, seq string) (HTTPDocument, error)
}

type HTTPFetcher struct {
	httpClient *http.Client
	userAgent  string
	delay      time.Duration
	last       time.Time
}

func NewHTTPFetcher() *HTTPFetcher {
	return &HTTPFetcher{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		userAgent:  "hic-discovery/0.1 (+public housing recruitment discovery)",
		delay:      500 * time.Millisecond,
	}
}

func (f *HTTPFetcher) FetchBoardList(ctx context.Context, board Board, page int) (HTTPDocument, error) {
	u, err := boardURL(board, board.ListPath)
	if err != nil {
		return HTTPDocument{}, err
	}
	values := u.Query()
	if board.Agency == "SH" && board.BoardKind == "rental" {
		values.Set("multi_itm_seq", "2")
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	u.RawQuery = values.Encode()
	return f.do(ctx, http.MethodGet, board, u.String(), "", "")
}

func (f *HTTPFetcher) FetchBoardDetail(ctx context.Context, board Board, seq string) (HTTPDocument, error) {
	u, err := boardURL(board, board.ViewPath)
	if err != nil {
		return HTTPDocument{}, err
	}
	values := url.Values{}
	if board.Agency == "SH" && board.BoardKind == "rental" {
		values.Set("page", "1")
		values.Set("multi_itm_seq", "2")
		values.Set("multi_itm_seqsStr", "")
		values.Set("seq", seq)
		values.Set("srchTp", "0")
		values.Set("srchWord", "")
	}
	return f.do(ctx, http.MethodPost, board, u.String(), values.Encode(), "application/x-www-form-urlencoded")
}

func (f *HTTPFetcher) do(ctx context.Context, method string, board Board, rawURL string, body string, contentType string) (HTTPDocument, error) {
	if err := f.wait(ctx); err != nil {
		return HTTPDocument{}, err
	}

	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, reader)
	if err != nil {
		return HTTPDocument{}, err
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/json;q=0.9,*/*;q=0.8")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if method == http.MethodPost {
		req.Header.Set("Referer", board.BaseURL+"/app/index.do")
	}

	resp, err := f.httpClient.Do(req)
	f.last = time.Now()
	if err != nil {
		return HTTPDocument{}, fmt.Errorf("%s %s: %w", method, rawURL, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return HTTPDocument{}, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return HTTPDocument{}, fmt.Errorf("%s %s: unexpected status %d", method, rawURL, resp.StatusCode)
	}

	return HTTPDocument{
		URL:         rawURL,
		Method:      method,
		RequestBody: body,
		ContentType: normalizeContentType(resp.Header.Get("Content-Type")),
		Body:        data,
		FetchedAt:   time.Now(),
	}, nil
}

func (f *HTTPFetcher) wait(ctx context.Context) error {
	if f.last.IsZero() || f.delay <= 0 {
		return nil
	}
	until := f.last.Add(f.delay)
	if time.Now().After(until) {
		return nil
	}
	timer := time.NewTimer(time.Until(until))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func boardURL(board Board, path string) (*url.URL, error) {
	base, err := url.Parse(board.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse board base URL: %w", err)
	}
	u, err := base.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parse board path: %w", err)
	}
	return u, nil
}

func normalizeContentType(value string) string {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return strings.TrimSpace(value)
	}
	return mediaType
}
