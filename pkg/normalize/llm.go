package normalize

import (
	"strconv"
	"strings"

	"hic/pkg/llm"
)

func OfferingsFromLLMRepairOutput(output llm.RepairOutput) []OfferingCandidate {
	sourceSpanCounts := make(map[string]int)
	for _, item := range output.Offerings {
		sourceSpan := strings.TrimSpace(item.SourceSpan)
		if sourceSpan == "" {
			sourceSpan = strings.TrimSpace(output.SourceSpan)
		}
		if sourceSpan != "" {
			sourceSpanCounts[sourceSpan]++
		}
	}
	sourceSpanIndexes := make(map[string]int)
	offerings := make([]OfferingCandidate, 0, len(output.Offerings))
	for _, item := range output.Offerings {
		sourceSpan := strings.TrimSpace(item.SourceSpan)
		if sourceSpan == "" {
			sourceSpan = strings.TrimSpace(output.SourceSpan)
		}
		if sourceSpanCounts[sourceSpan] > 1 {
			sourceSpanIndexes[sourceSpan]++
			sourceSpan = appendLLMItemSourceSpan(sourceSpan, sourceSpanIndexes[sourceSpan])
		}
		if strings.TrimSpace(item.ApplicationUnitLabel) == "" || sourceSpan == "" {
			continue
		}
		sourcePage := 0
		if item.SourcePage != nil {
			sourcePage = *item.SourcePage
		}
		unitNo := ""
		if item.UnitNo != nil {
			unitNo = strings.TrimSpace(*item.UnitNo)
		}
		offerings = append(offerings, OfferingCandidate{
			ApplicationUnitLabel: strings.TrimSpace(item.ApplicationUnitLabel),
			HousingName:          strings.TrimSpace(item.HousingName),
			UnitNo:               unitNo,
			ExclusiveAreaM2:      cloneFloat64Ptr(item.ExclusiveAreaM2),
			SupplyCount:          cloneIntPtr(item.SupplyCount),
			DepositKRW:           cloneInt64Ptr(item.DepositKRW),
			MonthlyRentKRW:       cloneInt64Ptr(item.MonthlyRentKRW),
			JeonseDepositKRW:     cloneInt64Ptr(item.JeonseDepositKRW),
			DormitoryFeeKRW:      cloneInt64Ptr(item.DormitoryFeeKRW),
			GenderRequirement:    strings.TrimSpace(item.GenderRequirement),
			SourcePage:           sourcePage,
			SourceSpan:           sourceSpan,
			RawRow: map[string]any{
				"source":       "llm_repair",
				"raw_evidence": item.RawEvidence,
			},
			Confidence: item.Confidence,
		})
	}
	return offerings
}

func appendLLMItemSourceSpan(sourceSpan string, index int) string {
	separator := "#"
	if strings.Contains(sourceSpan, "#") {
		separator = "&"
	}
	return sourceSpan + separator + "llm_item=" + strconv.Itoa(index)
}

func cloneFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	out := *value
	return &out
}
