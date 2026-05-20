package normalize

import "testing"

func TestMyHomePDFRentSummaryFromText_집주인임대주택표를요약한다(t *testing.T) {
	text := `
임대조건
주택군 소재지 호수 방 전용 공용 합계 임대보증금 (원) 월임대료 (원) 계 계약금 잔금
201
1
22.82
7.76
30.58
7,200,000
720,000
6,480,000
432,000
302
1
22.96
7.81
30.77
7,200,000
720,000
6,480,000
432,000
- 2 -
`

	summary, ok := MyHomePDFRentSummaryFromText(text)
	if !ok {
		t.Fatalf("MyHomePDFRentSummaryFromText() ok = false")
	}
	if summary.DepositKRW == nil || *summary.DepositKRW != 7200000 {
		t.Fatalf("DepositKRW = %v", summary.DepositKRW)
	}
	if summary.MonthlyRentKRW == nil || *summary.MonthlyRentKRW != 432000 {
		t.Fatalf("MonthlyRentKRW = %v", summary.MonthlyRentKRW)
	}
	if summary.DepositText != "7,200,000원" || summary.MonthlyRentText != "432,000원" {
		t.Fatalf("summary text = %+v", summary)
	}
}

func TestMyHomePDFRentSummaryFromText_여기가주택표는범위문구로요약한다(t *testing.T) {
	text := `
동 호 모집유형 면적 층수 방수 임대조건 (원)
전용면적 공용면적 합계 보증금 임대료 (월)
201
장애인
59.9560
12.7850
72.7410
2
층
2
7,930,000
562,220
102
장애인
69.5500
12.9890
82.5390
1
층
2
9,031,000
616,690
- 4 -
`

	summary, ok := MyHomePDFRentSummaryFromText(text)
	if !ok {
		t.Fatalf("MyHomePDFRentSummaryFromText() ok = false")
	}
	if summary.DepositKRW != nil || summary.MonthlyRentKRW != nil {
		t.Fatalf("range summary should not collapse to one numeric amount: %+v", summary)
	}
	if summary.DepositText != "7,930,000~9,031,000원" {
		t.Fatalf("DepositText = %q", summary.DepositText)
	}
	if summary.MonthlyRentText != "562,220~616,690원" {
		t.Fatalf("MonthlyRentText = %q", summary.MonthlyRentText)
	}
}
