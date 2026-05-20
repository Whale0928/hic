package discovery

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeFetcher struct {
	listHTML    string
	detailHTMLs map[string]string
}

func (f fakeFetcher) FetchBoardList(ctx context.Context, board Board, page int) (HTTPDocument, error) {
	return HTTPDocument{Body: []byte(f.listHTML)}, nil
}

func (f fakeFetcher) FetchBoardDetail(ctx context.Context, board Board, seq string) (HTTPDocument, error) {
	return HTTPDocument{Body: []byte(f.detailHTMLs[seq])}, nil
}

type pagedFakeFetcher struct {
	listHTMLByPage map[int]string
	detailHTMLs    map[string]string
}

func (f pagedFakeFetcher) FetchBoardList(ctx context.Context, board Board, page int) (HTTPDocument, error) {
	return HTTPDocument{Body: []byte(f.listHTMLByPage[page])}, nil
}

func (f pagedFakeFetcher) FetchBoardDetail(ctx context.Context, board Board, seq string) (HTTPDocument, error) {
	return HTTPDocument{Body: []byte(f.detailHTMLs[seq])}, nil
}

func TestDiscoverer_Discover_모집공고만후보로남긴다(t *testing.T) {
	listHTML := `
	<table id="listTb"><tbody>
	<tr><td>1</td><td><a href="javascript:getDetailView('100');">입주자 모집공고</a></td><td>공급부</td><td>2026-05-13</td><td>1</td></tr>
	<tr><td>2</td><td><a href="javascript:getDetailView('200');">당첨자 발표 안내</a></td><td>공급부</td><td>2026-05-13</td><td>1</td></tr>
	</tbody></table>`
	fetcher := fakeFetcher{
		listHTML: listHTML,
		detailHTMLs: map[string]string{
			"100": `<table><tr><th>제목</th><td>입주자 모집공고</td></tr><tr><td class="cont">공급대상 있음</td></tr></table>`,
			"200": `<table><tr><th>제목</th><td>당첨자 발표 안내</td></tr><tr><td class="cont">결과 안내</td></tr></table>`,
		},
	}

	report, err := NewDiscoverer(fetcher).Discover(context.Background(), Board{Agency: "SH", BoardKind: "rental"}, Options{Pages: 1})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if report.Pages != 1 || report.ListRows != 2 || report.Details != 2 {
		t.Fatalf("report counters = %+v", report)
	}
	if len(report.Candidates) != 1 || report.Candidates[0].Seq != "100" {
		t.Fatalf("candidates = %+v", report.Candidates)
	}
	if len(report.Rejected) != 1 || report.Rejected[0].Seq != "200" {
		t.Fatalf("rejected = %+v", report.Rejected)
	}
	if !strings.Contains(report.String(), "candidates=1") {
		t.Fatalf("report string = %q", report.String())
	}
}

func TestDiscoverer_Discover_Seq지정시목록없이상세만확인한다(t *testing.T) {
	fetcher := fakeFetcher{
		listHTML: "unused",
		detailHTMLs: map[string]string{
			"296353": `<table><tr><th>제목</th><td>청년 매입임대주택 입주자 모집공고</td></tr><tr><td class="cont">공급대상 있음</td></tr></table>`,
		},
	}

	report, err := NewDiscoverer(fetcher).Discover(context.Background(), Board{Agency: "SH", BoardKind: "rental"}, Options{Seqs: []string{"296353"}})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if report.Pages != 0 || report.ListRows != 0 || report.Details != 1 {
		t.Fatalf("report counters = %+v", report)
	}
	if len(report.Candidates) != 1 || report.Candidates[0].Seq != "296353" {
		t.Fatalf("candidates = %+v", report.Candidates)
	}
}

func TestDiscoverer_Discover_Seq와페이지를함께지정하면목록제목으로분류한다(t *testing.T) {
	listHTML := `
	<table id="listTb"><tbody>
	<tr><td>1</td><td><a href="javascript:getDetailView('296598');">2025년 하반기 청년 매입임대주택 우선공급 입주자 모집공고</a></td><td>공급부</td><td>2025-12-19</td><td>1</td></tr>
	<tr><td>2</td><td><a href="javascript:getDetailView('111111');">당첨자 발표</a></td><td>공급부</td><td>2025-12-19</td><td>1</td></tr>
	</tbody></table>`
	fetcher := fakeFetcher{
		listHTML: listHTML,
		detailHTMLs: map[string]string{
			"296598": `<script>var initParam={}; initParam.downList=[{"brdId":"GS0401","seq":"296598","fileSeq":"2","filename":"주택목록.xlsx","size":"10"}];</script><table><tr><td class="cont">상세 내용</td></tr></table>`,
		},
	}

	report, err := NewDiscoverer(fetcher).Discover(context.Background(), Board{Agency: "SH", BoardKind: "rental"}, Options{Pages: 2, Seqs: []string{"296598"}})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if report.ListRows != 2 || report.Details != 1 {
		t.Fatalf("report counters = %+v", report)
	}
	if len(report.Candidates) != 1 || report.Candidates[0].Seq != "296598" {
		t.Fatalf("candidates = %+v", report.Candidates)
	}
	if report.Candidates[0].Title == "" || len(report.Candidates[0].Attachments) != 1 {
		t.Fatalf("candidate detail = %+v", report.Candidates[0])
	}
}

func TestDiscoverer_Discover_이미알려진Seq는상세조회하지않는다(t *testing.T) {
	listHTML := `
	<table id="listTb"><tbody>
	<tr><td>1</td><td><a href="javascript:getDetailView('100');">입주자 모집공고</a></td><td>공급부</td><td>2026-05-13</td><td>1</td></tr>
	<tr><td>2</td><td><a href="javascript:getDetailView('200');">입주자 모집공고</a></td><td>공급부</td><td>2026-05-13</td><td>1</td></tr>
	</tbody></table>`
	fetcher := fakeFetcher{
		listHTML: listHTML,
		detailHTMLs: map[string]string{
			"200": `<table><tr><th>제목</th><td>입주자 모집공고</td></tr><tr><td class="cont">공급대상 있음</td></tr></table>`,
		},
	}

	report, err := NewDiscoverer(fetcher).Discover(context.Background(), Board{Agency: "SH", BoardKind: "rental"}, Options{
		Pages:     1,
		KnownSeqs: map[string]bool{"100": true},
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if report.Details != 1 || report.SkippedKnown != 1 {
		t.Fatalf("report counters = %+v", report)
	}
	if len(report.Candidates) != 1 || report.Candidates[0].Seq != "200" {
		t.Fatalf("candidates = %+v", report.Candidates)
	}
}

func TestDiscoverer_Discover_컷오프보다오래된행을만나면페이지탐색을멈춘다(t *testing.T) {
	listHTML := `
	<table id="listTb"><tbody>
	<tr><td>1</td><td><a href="javascript:getDetailView('100');">입주자 모집공고</a></td><td>공급부</td><td>2026-05-13</td><td>1</td></tr>
	<tr><td>2</td><td><a href="javascript:getDetailView('200');">입주자 모집공고</a></td><td>공급부</td><td>2026-04-01</td><td>1</td></tr>
	</tbody></table>`
	fetcher := fakeFetcher{
		listHTML: listHTML,
		detailHTMLs: map[string]string{
			"100": `<table><tr><th>제목</th><td>입주자 모집공고</td></tr><tr><td class="cont">공급대상 있음</td></tr></table>`,
		},
	}
	cutoff := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)

	report, err := NewDiscoverer(fetcher).Discover(context.Background(), Board{Agency: "SH", BoardKind: "rental"}, Options{Pages: 5, CutoffDate: cutoff})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if report.Pages != 1 || report.Details != 1 || report.SkippedOld != 1 || !report.StoppedByCutoff {
		t.Fatalf("report counters = %+v", report)
	}
	if len(report.Candidates) != 1 || report.Candidates[0].Seq != "100" {
		t.Fatalf("candidates = %+v", report.Candidates)
	}
}

func TestDiscoverer_Discover_목표공고가남아있으면컷오프에서멈추지않는다(t *testing.T) {
	fetcher := pagedFakeFetcher{
		listHTMLByPage: map[int]string{
			1: `<table id="listTb"><tbody>
				<tr><td>1</td><td><a href="javascript:getDetailView('100');">오래된 모집공고</a></td><td>공급부</td><td>2026-04-01</td><td>1</td></tr>
				</tbody></table>`,
			2: `<table id="listTb"><tbody>
				<tr><td>1</td><td><a href="javascript:getDetailView('303584');">2026년 전세임대형 든든주택 입주자 모집공고(2026.04.29.)</a></td><td>공급부</td><td>2026-04-29</td><td>1</td></tr>
				</tbody></table>`,
		},
		detailHTMLs: map[string]string{
			"303584": `<table><tr><th>제목</th><td>2026년 전세임대형 든든주택 입주자 모집공고(2026.04.29.)</td></tr><tr><td class="cont">공급대상 있음</td></tr></table>`,
		},
	}
	cutoff := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	report, err := NewDiscoverer(fetcher).Discover(context.Background(), Board{Agency: "SH", BoardKind: "rental"}, Options{
		Pages:        2,
		CutoffDate:   cutoff,
		TargetTitles: []string{"2026년 전세임대형 든든주택 입주자 모집공고"},
	})

	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if report.Pages != 2 || len(report.Candidates) != 1 || report.Candidates[0].Seq != "303584" {
		t.Fatalf("report = %+v", report)
	}
	if report.StoppedByCutoff {
		t.Fatalf("StoppedByCutoff = true, want false while target remains")
	}
}
