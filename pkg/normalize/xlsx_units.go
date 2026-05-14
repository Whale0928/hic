package normalize

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"hic/pkg/extraction"
)

type HousingUnitCandidate struct {
	SupplyCategory  string
	ListNo          string
	District        string
	Address         string
	LegalDong       string
	AddressDetail   string
	HousingName     string
	ComplexName     string
	BuildingName    string
	UnitNo          string
	FloorNo         *int
	UnitType        string
	StructureType   string
	ExclusiveAreaM2 *float64
	AreaPyeong      *float64
	DepositText     string
	DepositKRW      *int64
	MonthlyRentText string
	MonthlyRentKRW  *int64
	SupplyCount     *int
	Direction       string
	Status          string
	SourceSheet     string
	SourceRow       int
	SourceCell      string
	SourcePage      int
	SourceSpan      string
	RawRow          map[string]any
	Confidence      float64
}

type headerMap struct {
	supplyCategory int
	listNo         int
	district       int
	address        int
	legalDong      int
	addressDetail  int
	housingName    int
	building       int
	unit           int
	floor          int
	unitType       int
	structureType  int
	area           int
	pyeong         int
	deposit        int
	rent           int
	count          int
	direction      int
	status         int
	valid          bool
}

var numericPattern = regexp.MustCompile(`[-+]?[0-9]+(?:,[0-9]{3})*(?:\.[0-9]+)?|[-+]?[0-9]+(?:\.[0-9]+)?`)

func InferHousingUnitsFromXLSXRows(artifacts []extraction.ExtractedArtifact) []HousingUnitCandidate {
	var units []HousingUnitCandidate
	headersBySheet := make(map[string]headerMap)
	labelsBySheet := make(map[string][]string)

	for _, artifact := range artifacts {
		if artifact.Type != extraction.ArtifactTypeXLSXRow {
			continue
		}
		cells := artifactCells(artifact)
		sheet := artifact.SourceSheet
		if header := inferHeader(cells); header.valid || looksLikeApplicantHeader(cells) {
			headersBySheet[sheet] = header
			labelsBySheet[sheet] = append([]string(nil), cells...)
			continue
		}
		header := headersBySheet[sheet]
		if !header.valid || isBlankRow(cells) {
			continue
		}

		unit := buildHousingUnitCandidate(artifact, cells, labelsBySheet[sheet], header)
		if unit.UnitNo == "" && unit.SupplyCount == nil {
			continue
		}
		units = append(units, unit)
	}

	return units
}

func inferHeader(cells []string) headerMap {
	if looksLikeApplicantHeader(cells) {
		return headerMap{}
	}

	header := headerMap{
		supplyCategory: -1,
		listNo:         -1,
		district:       -1,
		address:        -1,
		legalDong:      -1,
		addressDetail:  -1,
		housingName:    -1,
		building:       -1,
		unit:           -1,
		floor:          -1,
		unitType:       -1,
		structureType:  -1,
		area:           -1,
		pyeong:         -1,
		deposit:        -1,
		rent:           -1,
		count:          -1,
		direction:      -1,
		status:         -1,
	}
	for i, cell := range cells {
		key := normalizeHeader(cell)
		switch {
		case key == "공급유형" || key == "유형" || key == "공급구분":
			header.supplyCategory = i
		case key == "번호" || key == "연번" || key == "목록번호":
			header.listNo = i
		case key == "자치구" || key == "구":
			header.district = i
		case key == "주소" || strings.Contains(key, "소재지"):
			header.address = i
		case key == "법정동" || key == "동명":
			header.legalDong = i
		case strings.Contains(key, "상세주소"):
			header.addressDetail = i
		case key == "주택명" || key == "단지명" || key == "건물명":
			header.housingName = i
		case key == "동" || key == "주택동" || key == "건물동":
			header.building = i
		case key == "호" || key == "호수" || key == "동호" || key == "동호수":
			header.unit = i
		case key == "층" || key == "해당층":
			header.floor = i
		case strings.Contains(key, "타입") || strings.Contains(key, "형별") || strings.Contains(key, "주택형"):
			header.unitType = i
		case strings.Contains(key, "구조"):
			header.structureType = i
		case strings.Contains(key, "전용면적") || strings.Contains(key, "전용"):
			header.area = i
		case strings.Contains(key, "평"):
			header.pyeong = i
		case strings.Contains(key, "보증금"):
			header.deposit = i
		case strings.Contains(key, "월임대료") || key == "임대료" || strings.Contains(key, "월세"):
			header.rent = i
		case strings.Contains(key, "공급호수") || strings.Contains(key, "세대수") || strings.Contains(key, "호수계"):
			header.count = i
		case strings.Contains(key, "방향") || strings.Contains(key, "향"):
			header.direction = i
		case strings.Contains(key, "상태") || strings.Contains(key, "공가"):
			header.status = i
		}
	}
	header.valid = (header.unit >= 0 || header.count >= 0) && (header.area >= 0 || header.unitType >= 0 || header.deposit >= 0 || header.rent >= 0)
	return header
}

func buildHousingUnitCandidate(artifact extraction.ExtractedArtifact, cells []string, labels []string, header headerMap) HousingUnitCandidate {
	depositText := cellAt(cells, header.deposit)
	rentText := cellAt(cells, header.rent)
	if rentText == "" && header.deposit >= 0 && strings.Contains(normalizeHeader(cellAt(labels, header.deposit)), "임대료") {
		rentText = cellAt(cells, header.deposit+1)
	}

	return HousingUnitCandidate{
		SupplyCategory:  cellAt(cells, header.supplyCategory),
		ListNo:          cellAt(cells, header.listNo),
		District:        cellAt(cells, header.district),
		Address:         cellAt(cells, header.address),
		LegalDong:       cellAt(cells, header.legalDong),
		AddressDetail:   cellAt(cells, header.addressDetail),
		HousingName:     cellAt(cells, header.housingName),
		ComplexName:     cellAt(cells, header.housingName),
		BuildingName:    cellAt(cells, header.building),
		UnitNo:          cellAt(cells, header.unit),
		FloorNo:         parseIntPtr(cellAt(cells, header.floor)),
		UnitType:        cellAt(cells, header.unitType),
		StructureType:   cellAt(cells, header.structureType),
		ExclusiveAreaM2: parseFloatPtr(cellAt(cells, header.area)),
		AreaPyeong:      parseFloatPtr(cellAt(cells, header.pyeong)),
		DepositText:     depositText,
		DepositKRW:      parseKRWPtr(depositText),
		MonthlyRentText: rentText,
		MonthlyRentKRW:  parseKRWPtr(rentText),
		SupplyCount:     parseIntPtr(cellAt(cells, header.count)),
		Direction:       cellAt(cells, header.direction),
		Status:          cellAt(cells, header.status),
		SourceSheet:     artifact.SourceSheet,
		SourceRow:       artifact.SourceRow,
		SourceCell:      artifact.SourceCell,
		SourcePage:      artifact.SourcePage,
		SourceSpan:      artifact.SourceSpan,
		RawRow:          rowToMap(labels, cells),
		Confidence:      0.72,
	}
}

func artifactCells(artifact extraction.ExtractedArtifact) []string {
	rawCells, ok := artifact.Content["cells"]
	if !ok {
		return nil
	}
	switch cells := rawCells.(type) {
	case []string:
		return normalizeCells(cells)
	case []any:
		out := make([]string, 0, len(cells))
		for _, cell := range cells {
			out = append(out, fmt.Sprint(cell))
		}
		return normalizeCells(out)
	default:
		return nil
	}
}

func looksLikeApplicantHeader(cells []string) bool {
	joined := normalizeHeader(strings.Join(cells, " "))
	return strings.Contains(joined, "접수번호") || strings.Contains(joined, "성명") || strings.Contains(joined, "생월일")
}

func normalizeCells(cells []string) []string {
	out := make([]string, len(cells))
	for i, cell := range cells {
		out[i] = strings.TrimSpace(strings.Join(strings.Fields(cell), " "))
	}
	return out
}

func normalizeHeader(cell string) string {
	replacer := strings.NewReplacer(" ", "", "\n", "", "\t", "", "(", "", ")", "", "[", "", "]", "")
	return replacer.Replace(strings.TrimSpace(cell))
}

func rowToMap(labels []string, cells []string) map[string]any {
	row := make(map[string]any, len(cells))
	for i, cell := range cells {
		key := fmt.Sprintf("col_%d", i+1)
		if i < len(labels) && strings.TrimSpace(labels[i]) != "" {
			key = labels[i]
		}
		row[key] = cell
	}
	return row
}

func isBlankRow(cells []string) bool {
	for _, cell := range cells {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func cellAt(cells []string, idx int) string {
	if idx < 0 || idx >= len(cells) {
		return ""
	}
	return cells[idx]
}

func parseIntPtr(text string) *int {
	numeric := strings.ReplaceAll(numericPattern.FindString(text), ",", "")
	if numeric == "" {
		return nil
	}
	n, err := strconv.Atoi(strings.Split(numeric, ".")[0])
	if err != nil {
		return nil
	}
	return &n
}

func parseFloatPtr(text string) *float64 {
	numeric := strings.ReplaceAll(numericPattern.FindString(text), ",", "")
	if numeric == "" {
		return nil
	}
	n, err := strconv.ParseFloat(numeric, 64)
	if err != nil {
		return nil
	}
	return &n
}

func parseKRWPtr(text string) *int64 {
	numeric := strings.ReplaceAll(numericPattern.FindString(text), ",", "")
	if numeric == "" {
		return nil
	}
	n, err := strconv.ParseFloat(numeric, 64)
	if err != nil {
		return nil
	}
	krw := int64(n)
	return &krw
}
