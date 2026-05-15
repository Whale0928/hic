package normalize

import (
	"fmt"

	"hic/pkg/extraction"
)

func InferOfferingsFromPDFTableRows(artifacts []extraction.ExtractedArtifact) []OfferingCandidate {
	offerings := make([]OfferingCandidate, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.Type != extraction.ArtifactTypePDFTableRow {
			continue
		}
		unitNo := contentString(artifact.Content, "unit_no")
		if unitNo == "" {
			continue
		}
		depositText := contentString(artifact.Content, "deposit_text")
		rentText := contentString(artifact.Content, "monthly_rent_text")
		address := contentString(artifact.Content, "address")
		district := contentString(artifact.Content, "district")
		if district == "" {
			district = districtFromAddress(address)
		}

		offering := OfferingCandidate{
			OfferingType:    OfferingTypeUnit,
			SupplyCategory:  contentString(artifact.Content, "supply_category"),
			ListNo:          contentString(artifact.Content, "list_no"),
			District:        district,
			Address:         address,
			HousingName:     contentString(artifact.Content, "housing_name"),
			ComplexName:     contentString(artifact.Content, "housing_name"),
			UnitNo:          unitNo,
			ExclusiveAreaM2: parseFloatPtr(contentString(artifact.Content, "exclusive_area_m2")),
			DepositText:     depositText,
			DepositKRW:      parseKRWPtr(depositText),
			MonthlyRentText: rentText,
			MonthlyRentKRW:  parseKRWPtr(rentText),
			SupplyCount:     intPtr(1),
			SourceRow:       artifact.SourceRow,
			SourcePage:      artifact.SourcePage,
			SourceSpan:      artifact.SourceSpan,
			RawRow:          cloneContent(artifact.Content),
			Confidence:      artifact.Confidence,
		}
		offerings = append(offerings, offering)
	}
	return offerings
}

func contentString(content map[string]any, key string) string {
	if content == nil {
		return ""
	}
	value, ok := content[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func cloneContent(content map[string]any) map[string]any {
	out := make(map[string]any, len(content))
	for key, value := range content {
		out[key] = value
	}
	return out
}
