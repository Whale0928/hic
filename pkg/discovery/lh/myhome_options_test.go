package lh

import "testing"

func TestMyHomePagesToFetch_AllPages는TotalCount와NumRows로계산한다(t *testing.T) {
	pages := MyHomePagesToFetch(425, 200, true, 1)

	if pages != 3 {
		t.Fatalf("pages = %d, want 3", pages)
	}
}

func TestMyHomePagesToFetch_AllPages가아니면요청Pages를유지한다(t *testing.T) {
	pages := MyHomePagesToFetch(425, 200, false, 2)

	if pages != 2 {
		t.Fatalf("pages = %d, want 2", pages)
	}
}

func TestFilterMyHomeItemsByAgency_기관명을필터링한다(t *testing.T) {
	items := []MyHomeNoticeItem{
		{NoticeID: "1", Agency: "LH"},
		{NoticeID: "2", Agency: "부산도시공사"},
		{NoticeID: "3", Agency: "LH"},
	}

	got := FilterMyHomeItemsByAgency(items, "LH")

	if len(got) != 2 || got[0].NoticeID != "1" || got[1].NoticeID != "3" {
		t.Fatalf("filtered = %+v", got)
	}
}
