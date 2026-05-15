package extraction

import "testing"

func TestExtractPDFTableRowsFromText_위국헌신청년주택호실별규모를행으로추출한다(t *testing.T) {
	rows := ExtractPDFTableRowsFromText(`
공급 대상 주택 목록 구 분 소재지 주소 공급 호수 전용 면적 ( ㎡ ) 주택유형명 비고
1 서울특별시 강동구 천호동 191-25 13 호 29.99 오피스텔 도시형생활주택
2 서울특별시 강북구 수유동 392-132 2 호 27.58 27.73 도시형생활주택
ㅇ 호실별 규모 및 임대조건
공급대상 세대별 면적 ( ㎡ ) 임대조건
구분 연번 호수 전용면적 공급면적 ( 전용 + 공용 ) 기준 임대보증금 ( 원 ) 기준 임대료 ( 원 )
강동구 1 401 29.99 43.37 1,000,000 298,800 2,000,000 497,300
2 402 29.99 43.37 1,000,000 298,800 2,000,000 497,300
강북구 1 403 27.73 36.52 1,000,000 208,200 2,000,000 346,500
`, "pdf://sample.pdf")

	if len(rows) != 3 {
		t.Fatalf("ExtractPDFTableRowsFromText() len = %d, want 3", len(rows))
	}
	assertPDFTableCell(t, rows[0], "district", "강동구")
	assertPDFTableCell(t, rows[0], "address", "서울특별시 강동구 천호동 191-25")
	assertPDFTableCell(t, rows[0], "unit_no", "401")
	assertPDFTableCell(t, rows[0], "exclusive_area_m2", "29.99")
	assertPDFTableCell(t, rows[0], "deposit_text", "1,000,000")
	assertPDFTableCell(t, rows[0], "monthly_rent_text", "298,800")
	assertPDFTableCell(t, rows[1], "district", "강동구")
	assertPDFTableCell(t, rows[2], "district", "강북구")
	assertPDFTableCell(t, rows[2], "address", "서울특별시 강북구 수유동 392-132")
}

func TestExtractPDFTableRowsFromText_사회주택공실정보를행으로추출한다(t *testing.T) {
	rows := ExtractPDFTableRowsFromText(`
□ 공급주택 ① 유니버설디자인하우스 _ 수락 ( 서울시 노원구 동일로 243 길 103)
공실 정보 주택 호수 전용면적 ( ㎡ ) 공급면적 ( ㎡ ) 방 (room) 보증금 ( 원 ) 임대료 ( 원 ) 입주일
수락 207 호 32 ㎡ 51 ㎡ 1 룸 114,790,000 292,500 협의
수락 405 호 43 ㎡ 67 ㎡ 1.5 룸 160,000,000 445,000 2026 년 6 월 19 일
`, "pdf://social.pdf")

	if len(rows) != 2 {
		t.Fatalf("ExtractPDFTableRowsFromText() len = %d, want 2", len(rows))
	}
	assertPDFTableCell(t, rows[0], "housing_name", "수락")
	assertPDFTableCell(t, rows[0], "address", "서울시 노원구 동일로243길 103")
	assertPDFTableCell(t, rows[0], "unit_no", "207호")
	assertPDFTableCell(t, rows[0], "exclusive_area_m2", "32")
	assertPDFTableCell(t, rows[0], "deposit_text", "114,790,000")
	assertPDFTableCell(t, rows[0], "monthly_rent_text", "292,500")
	assertPDFTableCell(t, rows[1], "unit_no", "405호")
}

func TestExtractPDFTableRowsFromText_두레주택보증금선택형표를행으로추출한다(t *testing.T) {
	rows := ExtractPDFTableRowsFromText(`
공급대상 주택 위치 위 치 서울시 동대문구 망우로 18 다길 31-5
임대보증금 및 월임대료 공급대상 ( 호점 ) 위치 호 전용면적 ( ㎡ ) 월임대료
보증금 5,450,000 원 보증금 16,360,000 원 보증금 27,270,000 원
휘경마을 두레주택 동대문구 망우로 18 다길 31-5 201 10.92 268,300 원 213,800 원 159,200 원 202 10.92
`, "pdf://dure.pdf")

	if len(rows) != 2 {
		t.Fatalf("ExtractPDFTableRowsFromText() len = %d, want 2", len(rows))
	}
	assertPDFTableCell(t, rows[0], "housing_name", "휘경마을 두레주택")
	assertPDFTableCell(t, rows[0], "address", "서울시 동대문구 망우로18다길 31-5")
	assertPDFTableCell(t, rows[0], "unit_no", "201")
	assertPDFTableCell(t, rows[0], "exclusive_area_m2", "10.92")
	assertPDFTableCell(t, rows[0], "deposit_text", "5,450,000 원")
	assertPDFTableCell(t, rows[0], "monthly_rent_text", "268,300 원")
	assertPDFTableCell(t, rows[1], "unit_no", "202")
}

func assertPDFTableCell(t *testing.T, artifact ExtractedArtifact, key string, want string) {
	t.Helper()
	if artifact.Type != ArtifactTypePDFTableRow {
		t.Fatalf("Type = %q, want %q", artifact.Type, ArtifactTypePDFTableRow)
	}
	got, _ := artifact.Content[key].(string)
	if got != want {
		t.Fatalf("Content[%q] = %q, want %q", key, got, want)
	}
}
