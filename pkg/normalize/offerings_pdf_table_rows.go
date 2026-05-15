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
		depositText := contentString(artifact.Content, "deposit_text")
		jeonseDepositText := contentString(artifact.Content, "jeonse_deposit_text")
		moneyUnit := contentString(artifact.Content, "money_unit")
		rentText := contentString(artifact.Content, "monthly_rent_text")
		address := contentString(artifact.Content, "address")
		district := contentString(artifact.Content, "district")
		if district == "" {
			district = districtFromAddress(address)
		}
		area := parseFloatPtr(contentString(artifact.Content, "exclusive_area_m2"))
		housingName := contentString(artifact.Content, "housing_name")
		applicationCategory := contentString(artifact.Content, "application_category")
		supplyCount := parseIntPtr(contentString(artifact.Content, "supply_count"))
		if supplyCount == nil && unitNo != "" {
			supplyCount = intPtr(1)
		}

		offering := OfferingCandidate{
			ApplicationUnitLabel: firstNonEmptyString(contentString(artifact.Content, "application_unit_label"), buildApplicationUnitLabel(housingName, unitNo, area, applicationCategory, contentString(artifact.Content, "gender_requirement"))),
			SupplyMethod:         contentString(artifact.Content, "supply_method"),
			ApplicationCategory:  applicationCategory,
			SupplyCategory:       contentString(artifact.Content, "supply_category"),
			ListNo:               contentString(artifact.Content, "list_no"),
			District:             district,
			Address:              address,
			HousingName:          housingName,
			ComplexName:          housingName,
			UnitNo:               unitNo,
			ExclusiveAreaM2:      area,
			DepositText:          depositText,
			DepositKRW:           parseKRWPtr(depositText),
			JeonseDepositText:    jeonseDepositText,
			JeonseDepositKRW:     parseMoneyPtr(jeonseDepositText, moneyUnit),
			ContractDepositKRW:   parseMoneyPtr(contentString(artifact.Content, "contract_deposit_text"), moneyUnit),
			BalancePaymentKRW:    parseMoneyPtr(contentString(artifact.Content, "balance_payment_text"), moneyUnit),
			MonthlyRentText:      rentText,
			MonthlyRentKRW:       parseKRWPtr(rentText),
			SupplyCount:          supplyCount,
			ReservedCount:        parseIntPtr(contentString(artifact.Content, "reserved_count")),
			GenderRequirement:    contentString(artifact.Content, "gender_requirement"),
			OccupancyType:        contentString(artifact.Content, "occupancy_type"),
			CapacityPersons:      parseIntPtr(contentString(artifact.Content, "capacity_persons")),
			DormitoryFeeKRW:      parseKRWPtr(contentString(artifact.Content, "dormitory_fee_text")),
			HeatingMethod:        contentString(artifact.Content, "heating_method"),
			MoveInStartText:      contentString(artifact.Content, "move_in_start_text"),
			SourceRow:            artifact.SourceRow,
			SourcePage:           artifact.SourcePage,
			SourceSpan:           artifact.SourceSpan,
			RawRow:               cloneContent(artifact.Content),
			Confidence:           artifact.Confidence,
		}
		if offering.ApplicationUnitLabel == "" && offering.UnitNo == "" && offering.SupplyCount == nil {
			continue
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
