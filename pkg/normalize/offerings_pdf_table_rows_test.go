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

func TestInferOfferingsFromPDFTableRows_동호수없는전세임대공급호수형후보를만든다(t *testing.T) {
	artifacts := []extraction.ExtractedArtifact{
		{
			Type:       extraction.ArtifactTypePDFTableRow,
			SourceRow:  1,
			SourceSpan: "pdf://dndn.pdf#table=jeonse_lease_support&row=1",
			Content: map[string]any{
				"source":                 "pdf_table_jeonse_lease_support",
				"housing_name":           "전세임대형 든든주택",
				"address":                "서울특별시 전 지역",
				"supply_method":          "전세임대",
				"supply_count":           "500",
				"application_unit_label": "전세임대형 든든주택 서울특별시 전 지역 500호",
				"jeonse_deposit_text":    "보증금한도액 300,000,000원",
				"contract_deposit_text":  "입주자부담금 40,000,000원",
				"balance_payment_text":   "최대지원금액 160,000,000원",
				"monthly_rent_text":      "지원액 기준 연 1.2~2.2%",
			},
			Confidence: 0.78,
		},
	}

	offerings := InferOfferingsFromPDFTableRows(artifacts)

	if len(offerings) != 1 {
		t.Fatalf("InferOfferingsFromPDFTableRows() len = %d, want 1", len(offerings))
	}
	got := offerings[0]
	if got.UnitNo != "" {
		t.Fatalf("UnitNo = %q, want empty", got.UnitNo)
	}
	if got.SupplyCount == nil || *got.SupplyCount != 500 {
		t.Fatalf("SupplyCount = %v, want 500", got.SupplyCount)
	}
	if got.ApplicationUnitLabel != "전세임대형 든든주택 서울특별시 전 지역 500호" {
		t.Fatalf("ApplicationUnitLabel = %q", got.ApplicationUnitLabel)
	}
	if got.JeonseDepositKRW == nil || *got.JeonseDepositKRW != 300000000 {
		t.Fatalf("JeonseDepositKRW = %v, want 300000000", got.JeonseDepositKRW)
	}
	if got.ContractDepositKRW == nil || *got.ContractDepositKRW != 40000000 {
		t.Fatalf("ContractDepositKRW = %v, want 40000000", got.ContractDepositKRW)
	}
	if got.BalancePaymentKRW == nil || *got.BalancePaymentKRW != 160000000 {
		t.Fatalf("BalancePaymentKRW = %v, want 160000000", got.BalancePaymentKRW)
	}
}

func TestInferOfferingsFromPDFTableRows_성별호점단위는UnitNo없이공급호수로변환한다(t *testing.T) {
	artifacts := []extraction.ExtractedArtifact{
		{
			Type:       extraction.ArtifactTypePDFTableRow,
			SourceRow:  1,
			SourceSpan: "object://hic-originals/sh/304555/1-notice.pdf#table=chungshin_theater_dure_gender_supply&row=1",
			Content: map[string]any{
				"source":                 "pdf_table_chungshin_theater_dure_gender_supply",
				"housing_name":           "충신동 연극인 두레주택",
				"application_unit_label": "충신동 연극인 두레주택 4호점 남성",
				"address":                "서울특별시 종로구 충신동 1-136",
				"gender_requirement":     "남성",
				"supply_count":           "3",
				"deposit_text":           "1,090,000원",
				"monthly_rent_text":      "147,200~162,500원",
			},
			Confidence: 0.78,
		},
	}

	offerings := InferOfferingsFromPDFTableRows(artifacts)

	if len(offerings) != 1 {
		t.Fatalf("InferOfferingsFromPDFTableRows() len = %d, want 1", len(offerings))
	}
	got := offerings[0]
	if got.UnitNo != "" {
		t.Fatalf("UnitNo = %q, want empty", got.UnitNo)
	}
	if got.SupplyCount == nil || *got.SupplyCount != 3 {
		t.Fatalf("SupplyCount = %v, want 3", got.SupplyCount)
	}
	if got.ApplicationUnitLabel != "충신동 연극인 두레주택 4호점 남성" {
		t.Fatalf("ApplicationUnitLabel = %q", got.ApplicationUnitLabel)
	}
	if got.GenderRequirement != "남성" {
		t.Fatalf("GenderRequirement = %q", got.GenderRequirement)
	}
	if got.DepositKRW == nil || *got.DepositKRW != 1090000 {
		t.Fatalf("DepositKRW = %v, want 1090000", got.DepositKRW)
	}
	if got.MonthlyRentKRW == nil || *got.MonthlyRentKRW != 147200 {
		t.Fatalf("MonthlyRentKRW = %v, want 147200", got.MonthlyRentKRW)
	}
}
