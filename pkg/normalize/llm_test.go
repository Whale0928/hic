package normalize

import (
	"testing"

	"hic/pkg/llm"
)

func TestOfferingsFromLLMRepairOutput_공급항목후보로변환한다(t *testing.T) {
	unitNo := "502호"
	area := 47.09
	count := 1
	deposit := int64(42000000)
	rent := int64(495300)
	page := 3
	output := llm.RepairOutput{
		SourceSpan: "object://hic-originals/sh/304271/8-notice.pdf#page=3",
		Offerings: []llm.Offering{{
			ApplicationUnitLabel: "위국헌신청년주택 502호",
			HousingName:          "위국헌신청년주택",
			UnitNo:               &unitNo,
			ExclusiveAreaM2:      &area,
			SupplyCount:          &count,
			DepositKRW:           &deposit,
			MonthlyRentKRW:       &rent,
			GenderRequirement:    "남성",
			SourcePage:           &page,
			SourceSpan:           "object://hic-originals/sh/304271/8-notice.pdf#page=3&row=1",
			Confidence:           0.83,
			RawEvidence:          "502호 47.09 42,000,000 495,300",
		}},
	}

	offerings := OfferingsFromLLMRepairOutput(output)

	if len(offerings) != 1 {
		t.Fatalf("len(offerings) = %d, want 1", len(offerings))
	}
	offering := offerings[0]
	if offering.ApplicationUnitLabel != "위국헌신청년주택 502호" ||
		offering.HousingName != "위국헌신청년주택" ||
		offering.UnitNo != "502호" ||
		offering.SourcePage != 3 ||
		offering.SourceSpan != "object://hic-originals/sh/304271/8-notice.pdf#page=3&row=1" {
		t.Fatalf("offering = %+v", offering)
	}
	if offering.DepositKRW == nil || *offering.DepositKRW != 42000000 {
		t.Fatalf("DepositKRW = %v", offering.DepositKRW)
	}
	if offering.MonthlyRentKRW == nil || *offering.MonthlyRentKRW != 495300 {
		t.Fatalf("MonthlyRentKRW = %v", offering.MonthlyRentKRW)
	}
	if offering.RawRow["source"] != "llm_repair" || offering.RawRow["raw_evidence"] != "502호 47.09 42,000,000 495,300" {
		t.Fatalf("RawRow = %+v", offering.RawRow)
	}
}

func TestOfferingsFromLLMRepairOutput_호실이없어도공급호수단위는유지한다(t *testing.T) {
	count := 15
	output := llm.RepairOutput{
		Offerings: []llm.Offering{{
			ApplicationUnitLabel: "청담르엘 49 일반",
			HousingName:          "청담르엘",
			SupplyCount:          &count,
			SourceSpan:           "object://hic-originals/sh/304271/13-pamphlet.pdf#page=5&row=2",
			Confidence:           0.8,
		}},
	}

	offerings := OfferingsFromLLMRepairOutput(output)

	if len(offerings) != 1 {
		t.Fatalf("len(offerings) = %d, want 1", len(offerings))
	}
	if offerings[0].UnitNo != "" {
		t.Fatalf("UnitNo = %q, want empty for nullable DB column", offerings[0].UnitNo)
	}
	if offerings[0].SupplyCount == nil || *offerings[0].SupplyCount != 15 {
		t.Fatalf("SupplyCount = %v", offerings[0].SupplyCount)
	}
}

func TestOfferingsFromLLMRepairOutput_중복SourceSpan은항목별Suffix를붙인다(t *testing.T) {
	span := "object://hic-originals/sh/304555/notice.pdf"
	output := llm.RepairOutput{
		Offerings: []llm.Offering{
			{ApplicationUnitLabel: "4호점 101", HousingName: "충신동 연극인 두레주택", UnitNo: stringPtr("101"), SourceSpan: span, Confidence: 0.9},
			{ApplicationUnitLabel: "4호점 102", HousingName: "충신동 연극인 두레주택", UnitNo: stringPtr("102"), SourceSpan: span, Confidence: 0.9},
		},
	}

	offerings := OfferingsFromLLMRepairOutput(output)

	if len(offerings) != 2 {
		t.Fatalf("len(offerings) = %d, want 2", len(offerings))
	}
	if offerings[0].SourceSpan != span+"#llm_item=1" || offerings[1].SourceSpan != span+"#llm_item=2" {
		t.Fatalf("source spans = %q, %q", offerings[0].SourceSpan, offerings[1].SourceSpan)
	}
}

func stringPtr(value string) *string {
	return &value
}
