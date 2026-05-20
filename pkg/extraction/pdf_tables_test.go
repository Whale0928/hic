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

func TestExtractPDFTableRowsFromText_희망하우징공급대상을행으로추출한다(t *testing.T) {
	rows := ExtractPDFTableRowsFromText(`
■ 공급대상 : 희망하우징(공공기숙사) 잔여세대 18호
주택명 주택유형 전용면적 성별 공급호수
합계 남성6 여성12
내발산 공공기숙사 원룸형 1인실(장애인/일반) 23.3㎡ 여성1
정릉 희망하우징 원룸형 1인실 14.2㎡ 남성1 여성5
연남 공공원룸텔 원룸형 1인실 13.4㎡ 남성5 여성6
주택명 관련정보
내발산 공공기숙사 · 주소 : 서울특별시 강서구 수명로1길 131 (내발산동 740)
정릉 희망하우징 · 주소 : 서울특별시 성북구 정릉로 199 (정릉동 1036)
연남 공공원룸텔 · 주소 : 서울특별시 마포구 성미산로 17길 79 (연남동 487-35)
주택명 임대보증금 월 임대료 기숙사비
내발산 공공기숙사 - - 120,000원
정릉 희망하우징 1,090,000원 90,900원 -
연남 공공원룸텔 1,090,000원 145,300원 -
`, "pdf://hope.pdf")

	if len(rows) != 5 {
		t.Fatalf("ExtractPDFTableRowsFromText() len = %d, want 5", len(rows))
	}
	assertPDFTableCell(t, rows[0], "housing_name", "내발산 공공기숙사")
	assertPDFTableCell(t, rows[0], "gender_requirement", "여성")
	assertPDFTableCell(t, rows[0], "supply_count", "1")
	assertPDFTableCell(t, rows[0], "dormitory_fee_text", "120,000원")
	assertPDFTableCell(t, rows[1], "housing_name", "정릉 희망하우징")
	assertPDFTableCell(t, rows[1], "gender_requirement", "남성")
	assertPDFTableCell(t, rows[2], "gender_requirement", "여성")
	assertPDFTableCell(t, rows[4], "monthly_rent_text", "145,300원")
}

func TestExtractPDFTableRowsFromText_전세임대형든든주택공급개요를행으로추출한다(t *testing.T) {
	rows := ExtractPDFTableRowsFromText(`
2026년 전세임대형 든든주택 입주자 모집공고
■ 서울주택도시개발공사에서 신생아 가구, 다자녀 가구, 신혼부부, 예비신혼부부 등을 대상으로 전세임대형 든든주택 입주자를 모집합니다.
■ 사업지역 : 서울특별시 전 지역
■ 공급호수 : 500호
■ 대상주택 : 서울특별시 내 아래의 조건을 충족하는 주택
■ 지원금액
구분 지원기준금액 최대지원금액 입주자부담금 보증금한도액
호당 2억원 지원기준금액의 80% 이내(최대 1억 6,000만원) 지원기준금액 및 전세보증금의 20% 지원기준금액의 150% 이내(최대 3억원)
■ 월 임대료 : 지원받은 금액에 따라 연 금리 1.2~2.2% 적용
`, "pdf://dndn.pdf")

	if len(rows) != 1 {
		t.Fatalf("ExtractPDFTableRowsFromText() len = %d, want 1", len(rows))
	}
	assertPDFTableCell(t, rows[0], "source", "pdf_table_jeonse_lease_support")
	assertPDFTableCell(t, rows[0], "housing_name", "전세임대형 든든주택")
	assertPDFTableCell(t, rows[0], "address", "서울특별시 전 지역")
	assertPDFTableCell(t, rows[0], "supply_method", "전세임대")
	assertPDFTableCell(t, rows[0], "supply_count", "500")
	assertPDFTableCell(t, rows[0], "jeonse_deposit_text", "보증금한도액 300,000,000원")
	assertPDFTableCell(t, rows[0], "contract_deposit_text", "입주자부담금 40,000,000원")
	assertPDFTableCell(t, rows[0], "balance_payment_text", "최대지원금액 160,000,000원")
}

func TestExtractPDFTableRowsFromText_전세임대형OCR깨진퍼센트는금액으로보지않는다(t *testing.T) {
	rows := ExtractPDFTableRowsFromText(`
2026년 전세임대형 든든주택 입주자 모집공고
■ 사업지역 : 서울특별시 전 지역
■ 공급호수 : 500호
■ 지원금액
구분지원기준금액1)�대지원금액2)입주자부담금3)보증금한도액4)
호당2�원 지원기준금액의 �0� 이�(�대 1� 6,000만원) 지원기준금액 �� � 전세보증금의 20� 지원기준금액의 150� 이�(�대 3�원)
■ 월 임대료
`, "pdf://dndn-ocr.pdf")

	if len(rows) != 1 {
		t.Fatalf("ExtractPDFTableRowsFromText() len = %d, want 1", len(rows))
	}
	assertPDFTableCell(t, rows[0], "jeonse_deposit_text", "보증금한도액 300,000,000원")
	assertPDFTableCell(t, rows[0], "contract_deposit_text", "입주자부담금 40,000,000원")
	assertPDFTableCell(t, rows[0], "balance_payment_text", "최대지원금액 160,000,000원")
}

func TestExtractPDFTableRowsFromText_장기전세공급현황을행으로추출한다(t *testing.T) {
	rows := ExtractPDFTableRowsFromText(`
자치구 단지명 전용 면적 (㎡) 유형 모집 호수 (예비) 전세금액(천원) 세대 당 계약면적(㎡) 난방 방식
강남구
세곡2지구 - 래미안포레, 강남한양수자인
59 일반 34 514,020 51,402 462,618 59.97 25.10 36.64 121.71 개별난방
주거약자 7 514,020 51,402 462,618 59.97 25.10 36.64 121.71 개별난방
84 일반 11 655,980 65,598 590,382 84.85 33.45 51.84 170.14 개별난방
`, "pdf://longterm.pdf")

	if len(rows) != 3 {
		t.Fatalf("ExtractPDFTableRowsFromText() len = %d, want 3", len(rows))
	}
	assertPDFTableCell(t, rows[0], "district", "강남구")
	assertPDFTableCell(t, rows[0], "housing_name", "세곡2지구 - 래미안포레 강남한양수자인")
	assertPDFTableCell(t, rows[0], "exclusive_area_m2", "59")
	assertPDFTableCell(t, rows[0], "application_category", "일반")
	assertPDFTableCell(t, rows[0], "supply_count", "34")
	assertPDFTableCell(t, rows[0], "jeonse_deposit_text", "514,020")
	assertPDFTableCell(t, rows[0], "heating_method", "개별난방")
	assertPDFTableCell(t, rows[1], "application_category", "주거약자")
	assertPDFTableCell(t, rows[2], "exclusive_area_m2", "84")
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
