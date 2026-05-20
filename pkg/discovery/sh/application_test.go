package sh

import (
	"strings"
	"testing"
	"time"
)

func TestParseApplicationList_든든주택청약중행을파싱한다(t *testing.T) {
	html := `<table><tbody><tr>
	<td>1</td>
	<td><a onclick="userSsnCheck('202620092','','12', '32', 'N', '', '2026년 전세임대형 든든주택 입주자 모집 공고(2026.4.29.)')">2026년 전세임대형 든든주택 입주자 모집 공고(2026.4.29.)</a></td>
	<td>500</td>
	<td>2026-04-29</td>
	<td><a class="btn btnGreen">청약중</a></td>
	</tr></tbody></table>`

	rows, err := ParseApplicationList(strings.NewReader(html), ApplicationEndpoint{SupplyType: "12"})

	if err != nil {
		t.Fatalf("ParseApplicationList() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.RecruitNoticeCode != "202620092" || row.SupplyType != "12" || row.RecruitType != "32" {
		t.Fatalf("codes = %+v", row)
	}
	if row.Title != "2026년 전세임대형 든든주택 입주자 모집 공고(2026.4.29.)" {
		t.Fatalf("Title = %q", row.Title)
	}
	if row.Status != StatusOpen {
		t.Fatalf("Status = %q, want %q", row.Status, StatusOpen)
	}
	if row.SupplyCount == nil || *row.SupplyCount != 500 {
		t.Fatalf("SupplyCount = %+v, want 500", row.SupplyCount)
	}
	wantPosted := time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC)
	if !row.PostedAt.Equal(wantPosted) {
		t.Fatalf("PostedAt = %v, want %v", row.PostedAt, wantPosted)
	}
}

func TestParseApplicationList_접수예정행을상태로구분한다(t *testing.T) {
	html := `<table><tbody><tr>
	<td>1</td>
	<td><a onclick="userSsnCheck('202620111','','06', '20', 'N', '', '희망하우징 모집공고')">희망하우징 모집공고</a></td>
	<td>18</td>
	<td>2026-05-11</td>
	<td><a class="btn btnTextG">접수예정</a></td>
	</tr></tbody></table>`

	rows, err := ParseApplicationList(strings.NewReader(html), ApplicationEndpoint{SupplyType: "06"})

	if err != nil {
		t.Fatalf("ParseApplicationList() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].Status != StatusPending {
		t.Fatalf("Status = %q, want %q", rows[0].Status, StatusPending)
	}
}

func TestDefaultApplicationEndpoints_전세임대12를포함한다(t *testing.T) {
	endpoints := DefaultApplicationEndpoints()

	ep, ok := endpoints.BySupplyType("12")

	if !ok {
		t.Fatal("splyTy=12 endpoint not found")
	}
	if ep.BaseURL != "https://www.i-sh.co.kr" || !strings.Contains(ep.Path, "appUser_list.do") {
		t.Fatalf("endpoint = %+v", ep)
	}
}
