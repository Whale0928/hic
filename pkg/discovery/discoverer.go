package discovery

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"
)

type Options struct {
	Pages      int
	Seqs       []string
	CutoffDate time.Time
	KnownSeqs  map[string]bool
}

type Candidate struct {
	Agency      string
	BoardKind   string
	Seq         string
	Title       string
	PostedAt    time.Time
	Attachments []AttachmentMeta
}

type RejectedPost struct {
	Agency    string
	BoardKind string
	Seq       string
	Title     string
	Reason    NoticeCategory
}

type Report struct {
	Agency          string
	BoardKind       string
	Pages           int
	ListRows        int
	Details         int
	SkippedOld      int
	SkippedKnown    int
	StoppedByCutoff bool
	Candidates      []Candidate
	Rejected        []RejectedPost
}

func (r Report) String() string {
	return fmt.Sprintf(
		"agency=%s board=%s pages=%d list_rows=%d details=%d candidates=%d rejected=%d skipped_old=%d skipped_known=%d stopped_by_cutoff=%t",
		r.Agency,
		r.BoardKind,
		r.Pages,
		r.ListRows,
		r.Details,
		len(r.Candidates),
		len(r.Rejected),
		r.SkippedOld,
		r.SkippedKnown,
		r.StoppedByCutoff,
	)
}

type Discoverer struct {
	fetcher Fetcher
}

func NewDiscoverer(fetcher Fetcher) Discoverer {
	return Discoverer{fetcher: fetcher}
}

func (d Discoverer) Discover(ctx context.Context, board Board, opts Options) (Report, error) {
	if opts.Pages <= 0 {
		opts.Pages = 1
	}
	report := Report{Agency: board.Agency, BoardKind: board.BoardKind}
	if len(opts.Seqs) > 0 && opts.Pages > 1 {
		return d.discoverSelectedRows(ctx, board, opts, report)
	}
	if len(opts.Seqs) > 0 {
		for _, seq := range opts.Seqs {
			if strings.TrimSpace(seq) == "" {
				continue
			}
			if err := d.discoverDetail(ctx, board, BoardRow{
				Agency:    board.Agency,
				BoardKind: board.BoardKind,
				Seq:       strings.TrimSpace(seq),
			}, &report); err != nil {
				return report, err
			}
		}
		return report, nil
	}

	for page := 1; page <= opts.Pages; page++ {
		doc, err := d.fetcher.FetchBoardList(ctx, board, page)
		if err != nil {
			return report, err
		}
		report.Pages++

		rows, err := ParseBoardList(bytes.NewReader(doc.Body), board)
		if err != nil {
			return report, err
		}
		report.ListRows += len(rows)

		seenOldRow := false
		for _, row := range rows {
			if isOlderThanCutoff(row.PostedAt, opts.CutoffDate) {
				report.SkippedOld++
				seenOldRow = true
				continue
			}
			if opts.KnownSeqs[row.Seq] {
				report.SkippedKnown++
				continue
			}
			if err := d.discoverDetail(ctx, board, row, &report); err != nil {
				return report, err
			}
		}
		if seenOldRow {
			report.StoppedByCutoff = true
			return report, nil
		}
	}

	return report, nil
}

func isOlderThanCutoff(postedAt time.Time, cutoff time.Time) bool {
	return !postedAt.IsZero() && !cutoff.IsZero() && postedAt.Before(cutoff)
}

func (d Discoverer) discoverSelectedRows(ctx context.Context, board Board, opts Options, report Report) (Report, error) {
	wanted := make(map[string]bool, len(opts.Seqs))
	for _, seq := range opts.Seqs {
		seq = strings.TrimSpace(seq)
		if seq != "" {
			wanted[seq] = false
		}
	}
	for page := 1; page <= opts.Pages; page++ {
		doc, err := d.fetcher.FetchBoardList(ctx, board, page)
		if err != nil {
			return report, err
		}
		report.Pages++

		rows, err := ParseBoardList(bytes.NewReader(doc.Body), board)
		if err != nil {
			return report, err
		}
		report.ListRows += len(rows)

		for _, row := range rows {
			if _, ok := wanted[row.Seq]; !ok {
				continue
			}
			if wanted[row.Seq] {
				continue
			}
			if err := d.discoverDetail(ctx, board, row, &report); err != nil {
				return report, err
			}
			wanted[row.Seq] = true
		}
		if allSeqsFound(wanted) {
			return report, nil
		}
	}

	for seq, found := range wanted {
		if found {
			continue
		}
		if err := d.discoverDetail(ctx, board, BoardRow{
			Agency:    board.Agency,
			BoardKind: board.BoardKind,
			Seq:       seq,
		}, &report); err != nil {
			return report, err
		}
	}
	return report, nil
}

func allSeqsFound(seqs map[string]bool) bool {
	for _, found := range seqs {
		if !found {
			return false
		}
	}
	return true
}

func (d Discoverer) discoverDetail(ctx context.Context, board Board, row BoardRow, report *Report) error {
	detailDoc, err := d.fetcher.FetchBoardDetail(ctx, board, row.Seq)
	if err != nil {
		return err
	}
	report.Details++

	detail, err := ParseBoardDetail(bytes.NewReader(detailDoc.Body))
	if err != nil {
		return err
	}
	title := firstNonEmpty(detail.Title, row.Title)
	category := ClassifyNotice(title, detail.BodyText)
	if category != NoticeCategoryRecruitment {
		report.Rejected = append(report.Rejected, RejectedPost{
			Agency:    row.Agency,
			BoardKind: row.BoardKind,
			Seq:       row.Seq,
			Title:     title,
			Reason:    category,
		})
		return nil
	}

	report.Candidates = append(report.Candidates, Candidate{
		Agency:      row.Agency,
		BoardKind:   row.BoardKind,
		Seq:         row.Seq,
		Title:       strings.TrimSpace(title),
		PostedAt:    row.PostedAt,
		Attachments: detail.Attachments,
	})
	return nil
}
