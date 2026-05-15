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

func TestInferOfferingsFromPDFTableRows_천원단위전세금액을원단위로변환한다(t *testing.T) {
	artifacts := []extraction.ExtractedArtifact{
		{
			Type:       extraction.ArtifactTypePDFTableRow,
			SourceRow:  1,
			SourceSpan: "pdf://longterm.pdf#table=long_term_jeonse&row=1",
			Content: map[string]any{
				"housing_name":          "세곡2지구",
				"exclusive_area_m2":     "59",
				"application_category":  "일반",
				"supply_count":          "34",
				"jeonse_deposit_text":   "514,020",
				"contract_deposit_text": "51,402",
				"balance_payment_text":  "462,618",
				"money_unit":            "천원",
			},
			Confidence: 0.78,
		},
	}

	offerings := InferOfferingsFromPDFTableRows(artifacts)

	if len(offerings) != 1 {
		t.Fatalf("InferOfferingsFromPDFTableRows() len = %d, want 1", len(offerings))
	}
	got := offerings[0]
	if got.JeonseDepositKRW == nil || *got.JeonseDepositKRW != 514020000 {
		t.Fatalf("JeonseDepositKRW = %v", got.JeonseDepositKRW)
	}
	if got.ContractDepositKRW == nil || *got.ContractDepositKRW != 51402000 {
		t.Fatalf("ContractDepositKRW = %v", got.ContractDepositKRW)
	}
	if got.BalancePaymentKRW == nil || *got.BalancePaymentKRW != 462618000 {
		t.Fatalf("BalancePaymentKRW = %v", got.BalancePaymentKRW)
	}
}
