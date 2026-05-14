package normalize

import (
	"testing"

	"hic/pkg/extraction"
)

func TestInferHousingUnitsFromPDFText_신청주택표를주택후보로변환한다(t *testing.T) {
	artifact := extraction.ExtractedArtifact{
		Type:       extraction.ArtifactTypePDFText,
		SourceSpan: "pdf://sample.pdf",
		RawText: `
신청
주택
주소
서울특별시
금천구
시흥대로
88
길
18
공급호실
방
개수
면적
(
㎡
)
임대조건
(
원
)
계
전용
공용
보증금
임대료
502
호
0
60.64
47.09
13.55
42,000,000
495,300
입주가능일
2026.08.20.
`,
	}

	units := InferHousingUnitsFromPDFText(artifact)

	if len(units) != 1 {
		t.Fatalf("InferHousingUnitsFromPDFText() len = %d, want 1", len(units))
	}
	got := units[0]
	if got.Address != "서울특별시 금천구 시흥대로88길 18" {
		t.Fatalf("Address = %q", got.Address)
	}
	if got.UnitNo != "502호" {
		t.Fatalf("UnitNo = %q", got.UnitNo)
	}
	if got.ExclusiveAreaM2 == nil || *got.ExclusiveAreaM2 != 47.09 {
		t.Fatalf("ExclusiveAreaM2 = %#v, want 47.09", got.ExclusiveAreaM2)
	}
	if got.DepositKRW == nil || *got.DepositKRW != 42000000 {
		t.Fatalf("DepositKRW = %#v, want 42000000", got.DepositKRW)
	}
	if got.MonthlyRentKRW == nil || *got.MonthlyRentKRW != 495300 {
		t.Fatalf("MonthlyRentKRW = %#v, want 495300", got.MonthlyRentKRW)
	}
	if got.SourceSpan != "pdf://sample.pdf" {
		t.Fatalf("SourceSpan = %q", got.SourceSpan)
	}
}

func TestInferHousingUnitsFromPDFText_공급주택표는이전주소필드를무시한다(t *testing.T) {
	artifact := extraction.ExtractedArtifact{
		Type:       extraction.ArtifactTypePDFText,
		SourceSpan: "pdf://supply.pdf",
		RawText: `
개인정보 제공 동의서
주소
신청자
-
Ⅱ
.
공급주택
:
서울특별시
금천구
시흥대로
88
길
18,
소담빌라
502
호
공급대상
방개수
면적
(
㎡
)
임대조건
(
원
)
모집호수
계
전용
공용
보증금
임대료
502
호
1
60.64
47.09
13.55
42,000,000
495,300
1
호
※
입주가능일
`,
	}

	units := InferHousingUnitsFromPDFText(artifact)

	if len(units) != 1 {
		t.Fatalf("InferHousingUnitsFromPDFText() len = %d, want 1", len(units))
	}
	got := units[0]
	if got.Address != "서울특별시 금천구 시흥대로88길 18, 소담빌라" {
		t.Fatalf("Address = %q", got.Address)
	}
	if got.UnitNo != "502호" {
		t.Fatalf("UnitNo = %q", got.UnitNo)
	}
	if got.ExclusiveAreaM2 == nil || *got.ExclusiveAreaM2 != 47.09 {
		t.Fatalf("ExclusiveAreaM2 = %#v, want 47.09", got.ExclusiveAreaM2)
	}
	if got.DepositKRW == nil || *got.DepositKRW != 42000000 {
		t.Fatalf("DepositKRW = %#v, want 42000000", got.DepositKRW)
	}
	if got.MonthlyRentKRW == nil || *got.MonthlyRentKRW != 495300 {
		t.Fatalf("MonthlyRentKRW = %#v, want 495300", got.MonthlyRentKRW)
	}
}

func TestInferHousingUnitsFromPDFText_PDFText가아니면무시한다(t *testing.T) {
	units := InferHousingUnitsFromPDFText(extraction.ExtractedArtifact{
		Type:    extraction.ArtifactTypeXLSXRow,
		RawText: "신청 주택 주소 서울특별시 금천구 시흥대로 88 길 18 공급호실 502 호",
	})

	if len(units) != 0 {
		t.Fatalf("InferHousingUnitsFromPDFText() len = %d, want 0", len(units))
	}
}
