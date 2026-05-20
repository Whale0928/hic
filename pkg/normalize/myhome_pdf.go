package normalize

import (
	"regexp"
	"strconv"
	"strings"
)

type MyHomePDFRentSummary struct {
	DepositText     string
	DepositKRW      *int64
	MonthlyRentText string
	MonthlyRentKRW  *int64
}

var myHomePDFMoneyLinePattern = regexp.MustCompile(`^[0-9]{1,3}(?:,[0-9]{3})+$`)

func MyHomePDFRentSummaryFromText(text string) (MyHomePDFRentSummary, bool) {
	lines := normalizedMyHomePDFLines(text)
	start := myHomeRentTableStart(lines)
	if start < 0 {
		return MyHomePDFRentSummary{}, false
	}
	groupSize := 2
	if myHomeHeaderHasContractBreakdown(lines, start) {
		groupSize = 4
	}
	values := myHomeRentTableMoneyValues(lines, start)
	if len(values) < groupSize {
		return MyHomePDFRentSummary{}, false
	}

	var deposits []int64
	var rents []int64
	for i := 0; i+groupSize-1 < len(values); i += groupSize {
		deposits = append(deposits, values[i])
		rents = append(rents, values[i+groupSize-1])
	}
	depositText, depositKRW := summarizeKRWValues(deposits)
	rentText, rentKRW := summarizeKRWValues(rents)
	if depositText == "" || rentText == "" {
		return MyHomePDFRentSummary{}, false
	}
	return MyHomePDFRentSummary{
		DepositText:     depositText,
		DepositKRW:      depositKRW,
		MonthlyRentText: rentText,
		MonthlyRentKRW:  rentKRW,
	}, true
}

func normalizedMyHomePDFLines(text string) []string {
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(strings.ReplaceAll(line, "\u00a0", " "))
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func myHomeRentTableStart(lines []string) int {
	for i, line := range lines {
		if !strings.Contains(line, "임대조건") {
			continue
		}
		window := strings.Join(lines[i:minInt(i+40, len(lines))], " ")
		if strings.Contains(window, "보증금") && strings.Contains(window, "임대료") {
			return i
		}
	}
	return -1
}

func myHomeHeaderHasContractBreakdown(lines []string, start int) bool {
	window := strings.Join(lines[start:minInt(start+40, len(lines))], " ")
	return strings.Contains(window, "계약금") && strings.Contains(window, "잔금")
}

func myHomeRentTableMoneyValues(lines []string, start int) []int64 {
	var values []int64
	for i := start + 1; i < len(lines) && i < start+180; i++ {
		line := strings.TrimSpace(lines[i])
		if len(values) > 0 && (line == "■" || looksLikePageBreak(lines, i)) {
			break
		}
		if !myHomePDFMoneyLinePattern.MatchString(line) {
			continue
		}
		parsed, err := strconv.ParseInt(strings.ReplaceAll(line, ",", ""), 10, 64)
		if err != nil || parsed <= 0 {
			continue
		}
		values = append(values, parsed)
	}
	return values
}

func looksLikePageBreak(lines []string, i int) bool {
	return i+2 < len(lines) && lines[i] == "-" && isIntegerLine(lines[i+1]) && lines[i+2] == "-"
}

func isIntegerLine(line string) bool {
	if line == "" {
		return false
	}
	for _, r := range line {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func summarizeKRWValues(values []int64) (string, *int64) {
	if len(values) == 0 {
		return "", nil
	}
	minValue := values[0]
	maxValue := values[0]
	for _, value := range values[1:] {
		if value < minValue {
			minValue = value
		}
		if value > maxValue {
			maxValue = value
		}
	}
	if minValue == maxValue {
		out := minValue
		return formatKRW(minValue) + "원", &out
	}
	return formatKRW(minValue) + "~" + formatKRW(maxValue) + "원", nil
}

func formatKRW(value int64) string {
	text := strconv.FormatInt(value, 10)
	if len(text) <= 3 {
		return text
	}
	var parts []string
	for len(text) > 3 {
		parts = append([]string{text[len(text)-3:]}, parts...)
		text = text[:len(text)-3]
	}
	if text != "" {
		parts = append([]string{text}, parts...)
	}
	return strings.Join(parts, ",")
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
