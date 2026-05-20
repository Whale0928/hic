package extraction

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	wigukRowPattern         = regexp.MustCompile(`(?:(서울시|[가-힣]+구)\s+)?([0-9]+)\s+([0-9]{3,4})\s+([0-9]+(?:\.[0-9]+)?)\s+([0-9]+(?:\.[0-9]+)?)\s+([0-9,]+)\s+([0-9,]+)\s+([0-9,]+)\s+([0-9,]+)`)
	districtAddressPattern  = regexp.MustCompile(`서울특별시\s+([가-힣]+구)\s+([가-힣0-9]+동)\s+([0-9]+(?:-[0-9]+)?)\s+[0-9]+\s+호`)
	vacancyPattern          = regexp.MustCompile(`([가-힣A-Za-z0-9_]+)\s+([0-9]{3,4})\s*호\s+([0-9]+(?:\.[0-9]+)?)\s*㎡\s+[0-9]+(?:\.[0-9]+)?\s*㎡\s+[0-9]+(?:\.[0-9]+)?\s*룸\s+([0-9,]+)\s+([0-9,]+)`)
	seoulAddressPattern     = regexp.MustCompile(`서울시\s+[가-힣]+구\s+[가-힣0-9]+\s+[0-9]+\s+길\s+[0-9]+`)
	moneyPattern            = regexp.MustCompile(`[0-9]+(?:,[0-9]{3})+`)
	dureUnitPattern         = regexp.MustCompile(`\b(20[0-9])\s+([0-9]+(?:\.[0-9]+)?)\b`)
	hopeHousingRowPattern   = regexp.MustCompile(`(내발산\s*공공기숙사|정릉\s*희망하우징|연남\s*공공원룸텔)\s*(원룸형\s*1\s*인실(?:\s*\([^)]+\))?)\s*([0-9]+(?:\.[0-9]+)?)\s*㎡?\s*((?:남성|여성)\s*[0-9]+\s*)+`)
	hopeGenderCountPattern  = regexp.MustCompile(`(남성|여성)\s*([0-9]+)`)
	hopeAddressPattern      = regexp.MustCompile(`(내발산\s*공공기숙사|정릉\s*희망하우징|연남\s*공공원룸텔)\s*·?\s*주소\s*:\s*(서울특별시\s+[가-힣]+구\s+.*?)(?:\s+·|\s+주택명|\s+정릉\s*희망하우징|\s+연남\s*공공원룸텔|$)`)
	hopeRentPattern         = regexp.MustCompile(`(내발산\s*공공기숙사|정릉\s*희망하우징|연남\s*공공원룸텔)\s*(-|[0-9,]+\s*원)\s+(-|[0-9,]+\s*원)\s+(-|[0-9,]+\s*원)`)
	jeonseLeaseCountPattern = regexp.MustCompile(`공급호수\s*[:：]\s*([0-9,]+)\s*호`)
	koreanMoneyPattern      = regexp.MustCompile(`([0-9]+)\s*(억|�)\s*(?:(?:([0-9,]+)\s*만\s*원?)|(?:([0-9,]+)\s*만원)|원)`)
	chungshinGroupStart     = regexp.MustCompile(`([0-9]+)\s*호점\s+종로구\s+충신동\s+([0-9]+-[0-9]+)`)
	chungshinSupplyCount    = regexp.MustCompile(`^\s*([0-9]+)\s*호\s+`)
	chungshinRentRow        = regexp.MustCompile(`([0-9]{3,4})\s+([0-9]+(?:\.[0-9]+)?)\s+([0-9,]+)\s+([0-9,]+)`)
	magokSupplyTypePattern  = regexp.MustCompile(`^([0-9]+)㎡형$`)
	magokUnitTypePattern    = regexp.MustCompile(`^[A-Z][0-9]?형$`)
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
	rows = append(rows, extractChungshinTheaterDureRows(compact, sourceSpan)...)
	rows = append(rows, extractHopeHousingRows(compact, sourceSpan)...)
	rows = append(rows, extractMagokChallengeHouseRows(compact, sourceSpan)...)
	rows = append(rows, extractJeonseLeaseSupportRows(compact, sourceSpan)...)
	rows = append(rows, extractLongTermJeonseRows(compact, sourceSpan)...)
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

type chungshinSupplyInfo struct {
	address     string
	gender      string
	supplyCount string
}

func extractChungshinTheaterDureRows(text string, sourceSpan string) []ExtractedArtifact {
	if !strings.Contains(text, "충신동 연극인 두레주택") || !strings.Contains(text, "호점 내 호실은 선택 불가") || !strings.Contains(text, "임대보증금 및 월임대료") {
		return nil
	}

	supplyByHojum := extractChungshinSupplyInfo(text)
	rentSection := sectionBetween(text, "임대보증금 및 월임대료", "○ 임대보증금")
	if rentSection == "" {
		rentSection = sectionBetween(text, "임대보증금 및 월임대료", "")
	}

	var rows []ExtractedArtifact
	for _, group := range splitChungshinGroups(rentSection) {
		hojum := group.hojum
		address := group.address
		unitRows := parseChungshinRentRows(group.body)
		if len(unitRows) == 0 {
			continue
		}
		info := supplyByHojum[hojum]
		if info.address != "" {
			address = info.address
		}
		gender := firstNonEmptyString(info.gender, detectChungshinGender(text, hojum))
		supplyCount := firstNonEmptyString(info.supplyCount, strconv.Itoa(len(unitRows)))
		depositText := formatKRWRange(unitRows[0].depositValue, unitRows[0].depositValue)
		minRent, maxRent := rentRange(unitRows)
		row := map[string]any{
			"source":                 "pdf_table_chungshin_theater_dure_gender_supply",
			"housing_name":           "충신동 연극인 두레주택",
			"address":                address,
			"supply_method":          "잔여세대",
			"supply_category":        "잔여세대",
			"application_category":   gender,
			"gender_requirement":     gender,
			"supply_count":           supplyCount,
			"occupancy_type":         "셰어하우스",
			"deposit_text":           depositText,
			"monthly_rent_text":      formatKRWRange(minRent, maxRent),
			"application_unit_label": buildApplicationUnitLabel("충신동 연극인 두레주택", hojum+"호점", gender),
			"unit_rows":              chungshinUnitRowsForContent(unitRows),
		}
		rows = append(rows, pdfTableRowArtifact(sourceSpan, "chungshin_theater_dure_gender_supply", len(rows)+1, group.rawText, row))
	}
	return rows
}

func extractChungshinSupplyInfo(text string) map[string]chungshinSupplyInfo {
	section := sectionBetween(text, "금회 공급대상", "나 . 임대보증금")
	if section == "" {
		section = sectionBetween(text, "공급대상 ( 호점 )", "임대보증금 및 월임대료")
	}
	out := make(map[string]chungshinSupplyInfo)
	for _, group := range splitChungshinGroups(section) {
		block := group.body
		supplyCount := ""
		if countMatch := chungshinSupplyCount.FindStringSubmatch(block); len(countMatch) == 2 {
			supplyCount = countMatch[1]
			block = strings.TrimSpace(strings.TrimPrefix(block, countMatch[0]))
		}
		hojum := group.hojum
		out[hojum] = chungshinSupplyInfo{
			address:     group.address,
			gender:      detectGenderInText(block),
			supplyCount: supplyCount,
		}
	}
	return out
}

type chungshinGroup struct {
	hojum   string
	address string
	body    string
	rawText string
}

func splitChungshinGroups(section string) []chungshinGroup {
	matches := chungshinGroupStart.FindAllStringSubmatchIndex(section, -1)
	groups := make([]chungshinGroup, 0, len(matches))
	for i, match := range matches {
		end := len(section)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		rawText := strings.TrimSpace(section[match[0]:end])
		body := strings.TrimSpace(section[match[1]:end])
		if marker := strings.Index(body, " ○ "); marker >= 0 {
			body = strings.TrimSpace(body[:marker])
			rawText = strings.TrimSpace(section[match[0] : match[1]+marker])
		}
		if marker := strings.Index(body, " 동일호점 "); marker >= 0 {
			body = strings.TrimSpace(body[:marker])
			rawText = strings.TrimSpace(section[match[0] : match[1]+marker])
		}
		if marker := strings.Index(body, " 남성은 "); marker >= 0 {
			body = strings.TrimSpace(body[:marker])
			rawText = strings.TrimSpace(section[match[0] : match[1]+marker])
		}
		groups = append(groups, chungshinGroup{
			hojum:   section[match[2]:match[3]],
			address: "서울특별시 종로구 충신동 " + section[match[4]:match[5]],
			body:    body,
			rawText: rawText,
		})
	}
	return groups
}

type chungshinRentUnit struct {
	unitNo       string
	area         string
	depositText  string
	rentText     string
	depositValue int64
	rentValue    int64
}

func parseChungshinRentRows(text string) []chungshinRentUnit {
	var rows []chungshinRentUnit
	for _, match := range chungshinRentRow.FindAllStringSubmatch(text, -1) {
		depositValue, ok := parseKRWToken(match[3])
		if !ok {
			continue
		}
		rentValue, ok := parseKRWToken(match[4])
		if !ok {
			continue
		}
		rows = append(rows, chungshinRentUnit{
			unitNo:       match[1],
			area:         match[2],
			depositText:  match[3],
			rentText:     match[4],
			depositValue: depositValue,
			rentValue:    rentValue,
		})
	}
	return rows
}

func rentRange(rows []chungshinRentUnit) (int64, int64) {
	if len(rows) == 0 {
		return 0, 0
	}
	minRent := rows[0].rentValue
	maxRent := rows[0].rentValue
	for _, row := range rows[1:] {
		if row.rentValue < minRent {
			minRent = row.rentValue
		}
		if row.rentValue > maxRent {
			maxRent = row.rentValue
		}
	}
	return minRent, maxRent
}

func parseKRWToken(text string) (int64, bool) {
	value, err := strconv.ParseInt(strings.ReplaceAll(text, ",", ""), 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func formatKRWRange(minValue int64, maxValue int64) string {
	if minValue <= 0 {
		return ""
	}
	if maxValue <= 0 || minValue == maxValue {
		return formatKRWAmount(minValue) + "원"
	}
	return formatKRWAmount(minValue) + "~" + formatKRWAmount(maxValue) + "원"
}

func detectChungshinGender(text string, hojum string) string {
	switch hojum {
	case "4":
		if strings.Contains(text, "남성은 자동으로 4호점") || strings.Contains(text, "남성은 자동으로 4 호점") {
			return "남성"
		}
	case "5":
		if strings.Contains(text, "여성은 자동으로 5호점") || strings.Contains(text, "여성은 자동으로 5 호점") {
			return "여성"
		}
	}
	return ""
}

func detectGenderInText(text string) string {
	if strings.Contains(text, "남성") {
		return "남성"
	}
	if strings.Contains(text, "여성") {
		return "여성"
	}
	return ""
}

func chungshinUnitRowsForContent(rows []chungshinRentUnit) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"unit_no":           row.unitNo,
			"exclusive_area_m2": row.area,
			"deposit_text":      row.depositText,
			"monthly_rent_text": row.rentText,
		})
	}
	return out
}

type magokSupplyGroup struct {
	supplyType string
	category   string
	count      int
	depositKRW int64
	rentKRW    int64
	unitRows   []map[string]any
}

func extractMagokChallengeHouseRows(text string, sourceSpan string) []ExtractedArtifact {
	if !strings.Contains(text, "마곡 도전숙") || !strings.Contains(text, "공급현황") || !strings.Contains(text, "공급유형별 면적 및 임대료") {
		return nil
	}
	section := sectionBetween(text, "공급유형 주택유형", "세대 내 옵션")
	if section == "" {
		section = sectionBetween(text, "공급유형 주택유형", "임대 금액 상세")
	}
	if section == "" {
		return nil
	}
	groups := parseMagokSupplyGroups(section)
	rows := make([]ExtractedArtifact, 0, len(groups))
	for _, group := range groups {
		label := buildApplicationUnitLabel("마곡 도전숙", group.supplyType, group.category)
		row := map[string]any{
			"source":                 "hwp_text_magok_challenge_house_supply",
			"housing_name":           "마곡 도전숙",
			"supply_method":          "일자리연계형 지원주택",
			"supply_category":        "일자리연계형 지원주택",
			"application_category":   group.category,
			"application_unit_label": label,
			"supply_count":           strconv.Itoa(group.count),
			"exclusive_area_m2":      strings.TrimSuffix(group.supplyType, "㎡형"),
			"deposit_text":           formatKRWAmount(group.depositKRW) + "원",
			"monthly_rent_text":      formatKRWAmount(group.rentKRW) + "원",
			"unit_rows":              group.unitRows,
		}
		rows = append(rows, pdfTableRowArtifact(sourceSpan, "magok_challenge_house_supply", len(rows)+1, label, row))
	}
	return rows
}

func parseMagokSupplyGroups(section string) []magokSupplyGroup {
	tokens := strings.Fields(section)
	groupsByType := make(map[string]*magokSupplyGroup)
	var order []string
	currentSupplyType := ""
	var currentDepositKRW int64
	var currentRentKRW int64

	for i := 0; i < len(tokens); i++ {
		token := cleanToken(tokens[i])
		if match := magokSupplyTypePattern.FindStringSubmatch(token); len(match) == 2 {
			currentSupplyType = token
			currentDepositKRW = 0
			currentRentKRW = 0
			if i+1 >= len(tokens) {
				continue
			}
			unitType := cleanToken(tokens[i+1])
			if !magokUnitTypePattern.MatchString(unitType) {
				continue
			}
			next, ok := parseMagokUnitRow(tokens, i+2, currentSupplyType, unitType, &currentDepositKRW, &currentRentKRW, groupsByType, &order)
			if ok {
				i = next - 1
			}
			continue
		}
		if currentSupplyType == "" || !magokUnitTypePattern.MatchString(token) {
			continue
		}
		next, ok := parseMagokUnitRow(tokens, i+1, currentSupplyType, token, &currentDepositKRW, &currentRentKRW, groupsByType, &order)
		if ok {
			i = next - 1
		}
	}

	out := make([]magokSupplyGroup, 0, len(order))
	for _, supplyType := range order {
		group := groupsByType[supplyType]
		if group != nil && group.count > 0 && group.depositKRW > 0 && group.rentKRW > 0 {
			out = append(out, *group)
		}
	}
	return out
}

func parseMagokUnitRow(tokens []string, start int, supplyType string, unitType string, currentDepositKRW *int64, currentRentKRW *int64, groupsByType map[string]*magokSupplyGroup, order *[]string) (int, bool) {
	i := start
	category := ""
	if i < len(tokens) && strings.Contains(tokens[i], "주거약자") {
		category = "주거약자"
		i++
	}
	if i >= len(tokens) || !looksLikeCountToken(cleanToken(tokens[i])) {
		return start, false
	}
	count, err := strconv.Atoi(cleanToken(tokens[i]))
	if err != nil {
		return start, false
	}
	i++
	depositKRW := *currentDepositKRW
	rentKRW := *currentRentKRW
	if i+1 < len(tokens) && moneyPattern.MatchString(cleanToken(tokens[i])) && moneyPattern.MatchString(cleanToken(tokens[i+1])) {
		if deposit, ok := parseKRWToken(cleanToken(tokens[i])); ok {
			depositKRW = deposit * 1000
			*currentDepositKRW = depositKRW
		}
		if rent, ok := parseKRWToken(cleanToken(tokens[i+1])); ok {
			rentKRW = rent
			*currentRentKRW = rentKRW
		}
		i += 2
	}
	if depositKRW == 0 || rentKRW == 0 || i >= len(tokens) || !looksLikeArea(cleanToken(tokens[i])) {
		return start, false
	}
	area := cleanToken(tokens[i])
	i++
	group := groupsByType[supplyType]
	if group == nil {
		group = &magokSupplyGroup{supplyType: supplyType, depositKRW: depositKRW, rentKRW: rentKRW}
		groupsByType[supplyType] = group
		*order = append(*order, supplyType)
	}
	if category != "" {
		group.category = category
	}
	group.count += count
	group.unitRows = append(group.unitRows, map[string]any{
		"unit_type":         unitType,
		"supply_count":      strconv.Itoa(count),
		"exclusive_area_m2": area,
		"deposit_text":      formatKRWAmount(depositKRW) + "원",
		"monthly_rent_text": formatKRWAmount(rentKRW) + "원",
	})
	return i, true
}

func extractHopeHousingRows(text string, sourceSpan string) []ExtractedArtifact {
	if !strings.Contains(text, "희망하우징") || !strings.Contains(text, "공급대상") {
		return nil
	}
	addressByHousing := extractHopeAddresses(text)
	rentByHousing := extractHopeRentConditions(text)

	var rows []ExtractedArtifact
	for _, match := range hopeHousingRowPattern.FindAllStringSubmatch(text, -1) {
		housingName := canonicalHopeHousingName(match[1])
		occupancyType := compactPDFText(match[2])
		area := match[3]
		for _, genderMatch := range hopeGenderCountPattern.FindAllStringSubmatch(match[0], -1) {
			gender := genderMatch[1]
			rent := rentByHousing[housingName]
			row := map[string]any{
				"source":                 "pdf_table_hope_housing_supply",
				"housing_name":           housingName,
				"address":                addressByHousing[housingName],
				"exclusive_area_m2":      area,
				"occupancy_type":         occupancyType,
				"gender_requirement":     gender,
				"supply_count":           genderMatch[2],
				"application_unit_label": buildApplicationUnitLabel(housingName, area, "", gender),
			}
			if rent.deposit != "" && rent.deposit != "-" {
				row["deposit_text"] = rent.deposit
			}
			if rent.monthlyRent != "" && rent.monthlyRent != "-" {
				row["monthly_rent_text"] = rent.monthlyRent
			}
			if rent.dormitoryFee != "" && rent.dormitoryFee != "-" {
				row["dormitory_fee_text"] = rent.dormitoryFee
			}
			rows = append(rows, pdfTableRowArtifact(sourceSpan, "hope_housing_supply", len(rows)+1, match[0], row))
		}
	}
	return rows
}

type hopeRentCondition struct {
	deposit      string
	monthlyRent  string
	dormitoryFee string
}

func extractHopeRentConditions(text string) map[string]hopeRentCondition {
	out := make(map[string]hopeRentCondition)
	for _, match := range hopeRentPattern.FindAllStringSubmatch(text, -1) {
		out[canonicalHopeHousingName(match[1])] = hopeRentCondition{
			deposit:      match[2],
			monthlyRent:  match[3],
			dormitoryFee: match[4],
		}
	}
	return out
}

func extractHopeAddresses(text string) map[string]string {
	out := make(map[string]string)
	for _, match := range hopeAddressPattern.FindAllStringSubmatch(text, -1) {
		out[canonicalHopeHousingName(match[1])] = strings.TrimSpace(match[2])
	}
	return out
}

func canonicalHopeHousingName(value string) string {
	key := strings.ReplaceAll(compactPDFText(value), " ", "")
	switch key {
	case "내발산공공기숙사":
		return "내발산 공공기숙사"
	case "정릉희망하우징":
		return "정릉 희망하우징"
	case "연남공공원룸텔":
		return "연남 공공원룸텔"
	default:
		return compactPDFText(value)
	}
}

func extractJeonseLeaseSupportRows(text string, sourceSpan string) []ExtractedArtifact {
	if !strings.Contains(text, "전세임대형 든든주택") || !strings.Contains(text, "공급호수") {
		return nil
	}
	countMatch := jeonseLeaseCountPattern.FindStringSubmatch(text)
	if len(countMatch) < 2 {
		return nil
	}
	supplyCount := strings.ReplaceAll(countMatch[1], ",", "")
	supportSection := sectionBetween(text, "지원금액", "월 임대료")
	if supportSection == "" {
		supportSection = text
	}
	amounts := extractKoreanMoneyAmounts(supportSection)
	supportStandard := firstPositiveAmount(amounts)
	guaranteeLimit := maxAmount(amounts)
	if guaranteeLimit == 0 && supportStandard > 0 {
		guaranteeLimit = supportStandard * 3 / 2
	}
	maxSupport := int64(0)
	tenantContribution := int64(0)
	if supportStandard > 0 {
		maxSupport = supportStandard * 8 / 10
		tenantContribution = supportStandard * 2 / 10
	}

	address := "서울특별시"
	if strings.Contains(text, "서울특별시 전 지역") {
		address = "서울특별시 전 지역"
	}
	housingName := "전세임대형 든든주택"
	row := map[string]any{
		"source":                 "pdf_table_jeonse_lease_support",
		"housing_name":           housingName,
		"address":                address,
		"supply_method":          "전세임대",
		"supply_count":           supplyCount,
		"application_unit_label": buildApplicationUnitLabel(housingName, address, supplyCount+"호"),
	}
	if guaranteeLimit > 0 {
		row["jeonse_deposit_text"] = "보증금한도액 " + formatKRWAmount(guaranteeLimit) + "원"
	}
	if tenantContribution > 0 {
		row["contract_deposit_text"] = "입주자부담금 " + formatKRWAmount(tenantContribution) + "원"
	}
	if maxSupport > 0 {
		row["balance_payment_text"] = "최대지원금액 " + formatKRWAmount(maxSupport) + "원"
	}
	return []ExtractedArtifact{
		pdfTableRowArtifact(sourceSpan, "jeonse_lease_support", 1, countMatch[0], row),
	}
}

func extractKoreanMoneyAmounts(text string) []int64 {
	matches := koreanMoneyPattern.FindAllStringSubmatch(text, -1)
	amounts := make([]int64, 0, len(matches))
	for _, match := range matches {
		eok, err := strconv.ParseInt(strings.ReplaceAll(match[1], ",", ""), 10, 64)
		if err != nil {
			continue
		}
		if eok > 30 {
			continue
		}
		amount := eok * 100000000
		manwon := firstNonEmptyString(match[3], match[4])
		if manwon != "" {
			value, err := strconv.ParseInt(strings.ReplaceAll(manwon, ",", ""), 10, 64)
			if err == nil {
				amount += value * 10000
			}
		}
		amounts = append(amounts, amount)
	}
	return amounts
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstPositiveAmount(amounts []int64) int64 {
	for _, amount := range amounts {
		if amount > 0 {
			return amount
		}
	}
	return 0
}

func maxAmount(amounts []int64) int64 {
	var max int64
	for _, amount := range amounts {
		if amount > max {
			max = amount
		}
	}
	return max
}

func formatKRWAmount(value int64) string {
	raw := strconv.FormatInt(value, 10)
	if len(raw) <= 3 {
		return raw
	}
	var b strings.Builder
	prefix := len(raw) % 3
	if prefix == 0 {
		prefix = 3
	}
	b.WriteString(raw[:prefix])
	for i := prefix; i < len(raw); i += 3 {
		b.WriteByte(',')
		b.WriteString(raw[i : i+3])
	}
	return b.String()
}

func extractLongTermJeonseRows(text string, sourceSpan string) []ExtractedArtifact {
	if !strings.Contains(text, "전세금액") || !strings.Contains(text, "모집") || !strings.Contains(text, "호수") {
		return nil
	}
	tokens := strings.Fields(text)
	var rows []ExtractedArtifact
	currentDistrict := ""
	lastHousingName := ""
	lastArea := ""
	nameStart := 0
	for i := 0; i < len(tokens); i++ {
		token := cleanToken(tokens[i])
		if isDistrictToken(token) {
			currentDistrict = token
			nameStart = i + 1
			continue
		}
		row, end, ok := parseLongTermRow(tokens, i, currentDistrict, lastHousingName, lastArea, nameStart)
		if !ok {
			continue
		}
		if row.housingName != "" {
			lastHousingName = row.housingName
		}
		lastArea = row.area
		rows = append(rows, pdfTableRowArtifact(sourceSpan, "long_term_jeonse", len(rows)+1, strings.Join(tokens[i:end], " "), row.content()))
		i = end - 1
		nameStart = end
		if i+1 < len(tokens) && isApplicationCategory(cleanToken(tokens[i+1])) && lastArea != "" {
			nameStart = i + 1
		}
	}
	return rows
}

type longTermRow struct {
	district            string
	housingName         string
	area                string
	applicationCategory string
	supplyCount         string
	reservedCount       string
	jeonseDeposit       string
	contractDeposit     string
	balancePayment      string
	heatingMethod       string
}

func (row longTermRow) content() map[string]any {
	content := map[string]any{
		"source":                 "pdf_table_long_term_jeonse",
		"district":               row.district,
		"housing_name":           row.housingName,
		"exclusive_area_m2":      row.area,
		"application_category":   row.applicationCategory,
		"supply_count":           row.supplyCount,
		"jeonse_deposit_text":    row.jeonseDeposit,
		"contract_deposit_text":  row.contractDeposit,
		"balance_payment_text":   row.balancePayment,
		"money_unit":             "천원",
		"heating_method":         row.heatingMethod,
		"application_unit_label": buildApplicationUnitLabel(row.housingName, row.area, row.applicationCategory, ""),
	}
	if row.reservedCount != "" {
		content["reserved_count"] = row.reservedCount
	}
	return content
}

func parseLongTermRow(tokens []string, areaIndex int, district string, lastHousingName string, lastArea string, nameStart int) (longTermRow, int, bool) {
	area := cleanToken(tokens[areaIndex])
	j := areaIndex + 1
	nameEnd := areaIndex
	if !looksLikeArea(area) {
		if lastArea == "" || !isApplicationCategory(area) {
			return longTermRow{}, 0, false
		}
		area = lastArea
		j = areaIndex
		nameEnd = areaIndex
	}
	category := ""
	if j < len(tokens) && isApplicationCategory(cleanToken(tokens[j])) {
		category = cleanToken(tokens[j])
		j++
	}
	if j >= len(tokens) || !looksLikeCountToken(cleanToken(tokens[j])) {
		return longTermRow{}, 0, false
	}
	supplyCount := cleanToken(tokens[j])
	depositIndex := -1
	for k := j + 1; k < len(tokens) && k <= j+4; k++ {
		if moneyPattern.MatchString(cleanToken(tokens[k])) {
			depositIndex = k
			break
		}
	}
	if depositIndex < 0 || depositIndex+2 >= len(tokens) {
		return longTermRow{}, 0, false
	}
	reservedCount := ""
	if depositIndex-j >= 3 && looksLikeCountToken(cleanToken(tokens[depositIndex-1])) {
		reservedCount = cleanToken(tokens[depositIndex-1])
	}
	if !moneyPattern.MatchString(cleanToken(tokens[depositIndex+1])) || !moneyPattern.MatchString(cleanToken(tokens[depositIndex+2])) {
		return longTermRow{}, 0, false
	}
	heatingIndex := findHeatingIndex(tokens, depositIndex+3)
	if heatingIndex < 0 {
		return longTermRow{}, 0, false
	}
	housingName := cleanHousingName(tokens, nameStart, nameEnd)
	if housingName == "" {
		housingName = lastHousingName
	}
	if district == "" || housingName == "" {
		return longTermRow{}, 0, false
	}
	heatingMethod := cleanToken(tokens[heatingIndex])
	if heatingMethod == "개별" || heatingMethod == "지역" {
		heatingMethod += "난방"
		heatingIndex++
	}
	return longTermRow{
		district:            district,
		housingName:         housingName,
		area:                area,
		applicationCategory: category,
		supplyCount:         supplyCount,
		reservedCount:       reservedCount,
		jeonseDeposit:       cleanToken(tokens[depositIndex]),
		contractDeposit:     cleanToken(tokens[depositIndex+1]),
		balancePayment:      cleanToken(tokens[depositIndex+2]),
		heatingMethod:       heatingMethod,
	}, heatingIndex + 1, true
}

func findHeatingIndex(tokens []string, start int) int {
	for i := start; i < len(tokens) && i <= start+8; i++ {
		token := cleanToken(tokens[i])
		if token == "개별난방" || token == "지역난방" {
			return i
		}
		if (token == "개별" || token == "지역") && i+1 < len(tokens) && cleanToken(tokens[i+1]) == "난방" {
			return i
		}
	}
	return -1
}

func cleanHousingName(tokens []string, start int, end int) string {
	if start < 0 || start >= end {
		return ""
	}
	parts := make([]string, 0, end-start)
	for _, token := range tokens[start:end] {
		token = cleanToken(token)
		if token == "" || isHeaderToken(token) || isDistrictToken(token) {
			continue
		}
		parts = append(parts, token)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func cleanToken(token string) string {
	return strings.Trim(token, " \t\r\n,.;:()[]{}")
}

func looksLikeArea(token string) bool {
	if !regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`).MatchString(token) {
		return false
	}
	switch token {
	case "29", "33", "35", "36", "38", "39", "41", "43", "44", "45", "47", "48", "49", "50", "51", "54", "56", "57", "59", "65", "66", "70", "71", "73", "74", "79", "84", "101", "114":
		return true
	}
	return strings.Contains(token, ".")
}

func looksLikeCountToken(token string) bool {
	return regexp.MustCompile(`^[0-9]+$`).MatchString(token) || token == "-"
}

func isApplicationCategory(token string) bool {
	return token == "일반" || token == "주거약자" || token == "우선" || token == "우선공급"
}

func isDistrictToken(token string) bool {
	switch token {
	case "강남구", "강동구", "강북구", "강서구", "관악구", "광진구", "구로구", "금천구", "노원구", "도봉구", "동대문구", "동작구", "마포구", "서대문구", "서초구", "성동구", "성북구", "송파구", "양천구", "영등포구", "용산구", "은평구", "종로구", "중구", "중랑구", "의정부시":
		return true
	default:
		return false
	}
}

func isHeaderToken(token string) bool {
	switch token {
	case "자치구", "단지명", "전용", "면적", "유형", "모집", "호수", "예비", "전세금액", "세대", "당", "계약면적", "난방", "방식", "계", "계약금", "잔금", "주거", "공용", "기타", "합계":
		return true
	default:
		return false
	}
}

func buildApplicationUnitLabel(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, " ")
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
