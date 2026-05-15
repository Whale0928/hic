package extraction

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	wigukRowPattern        = regexp.MustCompile(`(?:(서울시|[가-힣]+구)\s+)?([0-9]+)\s+([0-9]{3,4})\s+([0-9]+(?:\.[0-9]+)?)\s+([0-9]+(?:\.[0-9]+)?)\s+([0-9,]+)\s+([0-9,]+)\s+([0-9,]+)\s+([0-9,]+)`)
	districtAddressPattern = regexp.MustCompile(`서울특별시\s+([가-힣]+구)\s+([가-힣0-9]+동)\s+([0-9]+(?:-[0-9]+)?)\s+[0-9]+\s+호`)
	vacancyPattern         = regexp.MustCompile(`([가-힣A-Za-z0-9_]+)\s+([0-9]{3,4})\s*호\s+([0-9]+(?:\.[0-9]+)?)\s*㎡\s+[0-9]+(?:\.[0-9]+)?\s*㎡\s+[0-9]+(?:\.[0-9]+)?\s*룸\s+([0-9,]+)\s+([0-9,]+)`)
	seoulAddressPattern    = regexp.MustCompile(`서울시\s+[가-힣]+구\s+[가-힣0-9]+\s+[0-9]+\s+길\s+[0-9]+`)
	moneyPattern           = regexp.MustCompile(`[0-9]+(?:,[0-9]{3})+`)
	dureUnitPattern        = regexp.MustCompile(`\b(20[0-9])\s+([0-9]+(?:\.[0-9]+)?)\b`)
)

func ExtractPDFTableRowsFromText(text string, sourceSpan string) []ExtractedArtifact {
	compact := compactPDFText(text)
	if compact == "" {
		return nil
	}

	var rows []ExtractedArtifact
	rows = append(rows, extractWigukUnitConditionRows(compact, sourceSpan)...)
	rows = append(rows, extractVacancyRows(compact, sourceSpan)...)
	rows = append(rows, extractDureHouseRows(compact, sourceSpan)...)
	return rows
}

func extractWigukUnitConditionRows(text string, sourceSpan string) []ExtractedArtifact {
	section := sectionBetween(text, "호실별 규모 및 임대조건", "기타사항")
	if section == "" {
		return nil
	}
	addressByDistrict := extractDistrictAddresses(text)

	var rows []ExtractedArtifact
	currentDistrict := ""
	for _, match := range wigukRowPattern.FindAllStringSubmatch(section, -1) {
		if match[1] != "" {
			currentDistrict = match[1]
		}
		if currentDistrict == "" {
			continue
		}
		row := map[string]any{
			"source":            "pdf_table_wiguk_unit_conditions",
			"district":          currentDistrict,
			"address":           addressByDistrict[currentDistrict],
			"list_no":           match[2],
			"unit_no":           match[3],
			"exclusive_area_m2": match[4],
			"supply_area_m2":    match[5],
			"deposit_text":      match[6],
			"monthly_rent_text": match[7],
			"deposit_option_2":  match[8],
			"rent_option_2":     match[9],
		}
		rows = append(rows, pdfTableRowArtifact(sourceSpan, "unit_conditions", len(rows)+1, match[0], row))
	}
	return rows
}

func extractDistrictAddresses(text string) map[string]string {
	out := make(map[string]string)
	for _, match := range districtAddressPattern.FindAllStringSubmatch(text, -1) {
		district := match[1]
		if _, exists := out[district]; exists {
			continue
		}
		out[district] = "서울특별시 " + match[1] + " " + match[2] + " " + match[3]
	}
	return out
}

func extractVacancyRows(text string, sourceSpan string) []ExtractedArtifact {
	section := sectionBetween(text, "공실 정보", "임대 기간")
	if section == "" {
		return nil
	}
	address := formatPDFAddress(firstMatch(seoulAddressPattern, text))

	var rows []ExtractedArtifact
	for _, match := range vacancyPattern.FindAllStringSubmatch(section, -1) {
		row := map[string]any{
			"source":            "pdf_table_vacancy",
			"housing_name":      match[1],
			"address":           address,
			"unit_no":           match[2] + "호",
			"exclusive_area_m2": match[3],
			"deposit_text":      match[4],
			"monthly_rent_text": match[5],
		}
		rows = append(rows, pdfTableRowArtifact(sourceSpan, "vacancy", len(rows)+1, match[0], row))
	}
	return rows
}

func extractDureHouseRows(text string, sourceSpan string) []ExtractedArtifact {
	if !strings.Contains(text, "휘경마을 두레주택") || !strings.Contains(text, "임대보증금 및 월임대료") {
		return nil
	}

	section := sectionBetween(text, "임대보증금 및 월임대료", "임대기간")
	if section == "" {
		section = text
	}
	money := moneyPattern.FindAllString(section, -1)
	if len(money) < 4 {
		return nil
	}
	deposit := money[0] + " 원"
	rent := money[3] + " 원"
	address := "서울시 동대문구 망우로18다길 31-5"

	var rows []ExtractedArtifact
	for _, match := range dureUnitPattern.FindAllStringSubmatch(section, -1) {
		row := map[string]any{
			"source":            "pdf_table_dure_rent_options",
			"housing_name":      "휘경마을 두레주택",
			"address":           address,
			"unit_no":           match[1],
			"exclusive_area_m2": match[2],
			"deposit_text":      deposit,
			"monthly_rent_text": rent,
		}
		rows = append(rows, pdfTableRowArtifact(sourceSpan, "dure_rent_options", len(rows)+1, match[0], row))
	}
	return rows
}

func pdfTableRowArtifact(sourceSpan string, tableName string, rowNo int, rawText string, content map[string]any) ExtractedArtifact {
	span := fmt.Sprintf("%s#table=%s&row=%d", sourceSpan, tableName, rowNo)
	return ExtractedArtifact{
		Type:          ArtifactTypePDFTableRow,
		Extractor:     "pdf-table-candidate",
		Status:        ArtifactStatusExtracted,
		SchemaVersion: "v1",
		SourceRow:     rowNo,
		SourceSpan:    span,
		RawText:       rawText,
		Content:       content,
		Confidence:    0.78,
	}
}

func compactPDFText(text string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(strings.ReplaceAll(text, "\x00", "")), " "))
}

func firstMatch(pattern *regexp.Regexp, text string) string {
	match := pattern.FindString(text)
	return strings.TrimSpace(match)
}

func formatPDFAddress(address string) string {
	address = compactPDFText(address)
	address = regexp.MustCompile(`(대로|로)\s+([0-9]+)\s+길`).ReplaceAllString(address, "${1}${2}길")
	return address
}

func sectionBetween(text string, start string, end string) string {
	startIndex := strings.Index(text, start)
	if startIndex < 0 {
		return ""
	}
	section := text[startIndex:]
	if end == "" {
		return section
	}
	endIndex := strings.Index(section, end)
	if endIndex > 0 {
		return section[:endIndex]
	}
	return section
}
