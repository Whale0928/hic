package normalize

import (
	"regexp"
	"strings"

	"hic/pkg/extraction"
)

var roadBranchSpacingPattern = regexp.MustCompile(`(대로|로) ([0-9]+길)`)

func InferHousingUnitsFromPDFText(artifact extraction.ExtractedArtifact) []HousingUnitCandidate {
	if artifact.Type != extraction.ArtifactTypePDFText {
		return nil
	}

	tokens := pdfTokens(artifact.RawText)
	if len(tokens) == 0 {
		return nil
	}
	unit := inferSupplyHousingUnit(artifact, tokens)
	if unit.UnitNo == "" {
		unit = inferApplicationFormUnit(artifact, tokens)
	}
	if unit.UnitNo == "" {
		return nil
	}
	return []HousingUnitCandidate{unit}
}

func inferSupplyHousingUnit(artifact extraction.ExtractedArtifact, tokens []string) HousingUnitCandidate {
	supplyIndex := indexToken(tokens, "공급주택")
	targetIndex := indexToken(tokens, "공급대상")
	valuesStart := indexToken(tokens, "임대료")
	moveInIndex := indexToken(tokens, "입주가능일")
	if supplyIndex < 0 || targetIndex < 0 || valuesStart < 0 || moveInIndex < 0 {
		return HousingUnitCandidate{}
	}
	if !(supplyIndex < targetIndex && targetIndex < valuesStart && valuesStart < moveInIndex) {
		return HousingUnitCandidate{}
	}

	values := tokens[valuesStart+1 : moveInIndex]
	if len(values) < 8 {
		return HousingUnitCandidate{}
	}
	unitNo := unitNoFromValues(values)
	addressTokens := trimSupplyAddressTokens(tokens[supplyIndex+1:targetIndex], unitNo)
	return buildPDFUnit(artifact, formatKoreanAddress(addressTokens), unitNo, values, "pdf_supply_housing_table")
}

func inferApplicationFormUnit(artifact extraction.ExtractedArtifact, tokens []string) HousingUnitCandidate {
	roomIndex := indexToken(tokens, "공급호실")
	addressStart := nearestTokenBefore(tokens, "주소", roomIndex)
	if addressStart < 0 || roomIndex < 0 || addressStart >= roomIndex {
		return HousingUnitCandidate{}
	}

	valuesStart := indexToken(tokens, "임대료")
	moveInIndex := indexToken(tokens, "입주가능일")
	if valuesStart < 0 || moveInIndex < 0 || valuesStart >= moveInIndex {
		return HousingUnitCandidate{}
	}
	values := tokens[valuesStart+1 : moveInIndex]
	if len(values) < 8 {
		return HousingUnitCandidate{}
	}

	address := formatKoreanAddress(tokens[addressStart+1 : roomIndex])
	return buildPDFUnit(artifact, address, unitNoFromValues(values), values, "pdf_application_form_table")
}

func buildPDFUnit(artifact extraction.ExtractedArtifact, address string, unitNo string, values []string, source string) HousingUnitCandidate {
	valueOffset := 0
	if len(values) > 1 && values[1] == "호" {
		valueOffset = 2
	}
	if len(values) < valueOffset+6 {
		return HousingUnitCandidate{}
	}
	depositText := values[valueOffset+4]
	rentText := values[valueOffset+5]
	return HousingUnitCandidate{
		Address:         address,
		District:        districtFromAddress(address),
		UnitNo:          unitNo,
		SupplyCount:     intPtr(1),
		ExclusiveAreaM2: parseFloatPtr(values[valueOffset+2]),
		DepositText:     depositText,
		DepositKRW:      parseKRWPtr(depositText),
		MonthlyRentText: rentText,
		MonthlyRentKRW:  parseKRWPtr(rentText),
		SourceSpan:      artifact.SourceSpan,
		RawRow: map[string]any{
			"source": source,
			"tokens": values,
		},
		Confidence: 0.64,
	}
}

func pdfTokens(text string) []string {
	fields := strings.Fields(text)
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		field = strings.Trim(field, "[](){}:：")
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func indexToken(tokens []string, want string) int {
	for i, token := range tokens {
		if token == want {
			return i
		}
	}
	return -1
}

func nearestTokenBefore(tokens []string, want string, before int) int {
	if before < 0 {
		return -1
	}
	for i := before - 1; i >= 0; i-- {
		if tokens[i] == want {
			return i
		}
	}
	return -1
}

func unitNoFromValues(values []string) string {
	if len(values) == 0 {
		return ""
	}
	unitNo := values[0]
	if len(values) > 1 && values[1] == "호" {
		unitNo += "호"
	}
	return unitNo
}

func trimSupplyAddressTokens(tokens []string, unitNo string) []string {
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token != "" {
			out = append(out, token)
		}
	}
	if len(out) > 0 && out[0] == "" {
		out = out[1:]
	}
	if len(out) >= 2 && out[len(out)-1] == "호" && out[len(out)-2]+"호" == unitNo {
		out = out[:len(out)-2]
	}
	return out
}

func formatKoreanAddress(tokens []string) string {
	var b strings.Builder
	for i, token := range tokens {
		if token == "" {
			continue
		}
		if i > 0 && needsAddressSpace(tokens[i-1], token) {
			b.WriteByte(' ')
		}
		b.WriteString(token)
	}
	return roadBranchSpacingPattern.ReplaceAllString(strings.TrimSpace(b.String()), "$1$2")
}

func needsAddressSpace(prev string, curr string) bool {
	if prev == "" || curr == "" {
		return false
	}
	if curr == "길" || curr == "로" || curr == "가" || curr == "동" {
		return false
	}
	if numericPattern.MatchString(prev) && (curr == "길" || curr == "로") {
		return false
	}
	return strings.HasSuffix(prev, "시") ||
		strings.HasSuffix(prev, "도") ||
		strings.HasSuffix(prev, "구") ||
		strings.HasSuffix(prev, "군") ||
		strings.HasSuffix(prev, "동") ||
		strings.HasSuffix(prev, "길") ||
		strings.HasSuffix(prev, "로") ||
		numericPattern.MatchString(prev)
}

func districtFromAddress(address string) string {
	for _, token := range strings.Fields(address) {
		if strings.HasSuffix(token, "구") || strings.HasSuffix(token, "군") {
			return token
		}
	}
	return ""
}

func intPtr(value int) *int {
	return &value
}
