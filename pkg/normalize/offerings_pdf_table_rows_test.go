package normalize

import (
	"testing"

	"hic/pkg/extraction"
)

func TestInferOfferingsFromPDFTableRows_표행을공급항목후보로변환한다(t *testing.T) {
	artifacts := []extraction.ExtractedArtifact{
		{
			Type:       extraction.ArtifactTypePDFTableRow,
			SourcePage: 2,
			SourceSpan: "pdf://sample.pdf#table=unit_conditions&row=1",
			Content: map[string]any{
				"district":          "강동구",
				"address":           "서울시 강동구 예시로 1",
				"housing_name":      "위국헌신청년주택",
				"unit_no":           "401",
				"exclusive_area_m2": "29.99",
				"deposit_text":      "1,000,000",
				"monthly_rent_text": "298,800",
			},
		},
	}

	offerings := InferOfferingsFromPDFTableRows(artifacts)

	if len(offerings) != 1 {
		t.Fatalf("InferOfferingsFromPDFTableRows() len = %d, want 1", len(offerings))
	}
	got := offerings[0]
	if got.UnitNo != "401" {
		t.Fatalf("UnitNo = %q", got.UnitNo)
	}
	if got.District != "강동구" {
		t.Fatalf("District = %q", got.District)
	}
	if got.DepositKRW == nil || *got.DepositKRW != 1000000 {
		t.Fatalf("DepositKRW = %#v, want 1000000", got.DepositKRW)
	}
	if got.MonthlyRentKRW == nil || *got.MonthlyRentKRW != 298800 {
		t.Fatalf("MonthlyRentKRW = %#v, want 298800", got.MonthlyRentKRW)
	}
	if got.SourcePage != 2 {
		t.Fatalf("SourcePage = %d, want 2", got.SourcePage)
	}
}
