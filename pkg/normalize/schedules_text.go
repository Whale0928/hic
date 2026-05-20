package normalize

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"hic/pkg/extraction"
)

var applicationSchedulePattern = regexp.MustCompile(`(?s)(청약신청\s*접수기간|신청\s*접수|신청접수|접수기간)\s*[:：]?\s*([0-9]{4})\s*[.\-/년]\s*([0-9]{1,2})\s*[.\-/월]\s*([0-9]{1,2})\s*[.\-/일]?\s*(?:\([^)]*\))?\s*(?:(\d{1,2})\s*:\s*(\d{2}))?\s*(?:부터)?\s*[~∼-]\s*([0-9]{4})\s*[.\-/년]\s*([0-9]{1,2})\s*[.\-/월]\s*([0-9]{1,2})\s*[.\-/일]?\s*(?:\([^)]*\))?\s*(?:(\d{1,2})\s*:\s*(\d{2}))?`)
var procedureDatePattern = regexp.MustCompile("[‘'`]?([0-9]{2,4})\\s*[.]\\s*([0-9]{1,2})\\s*[.]\\s*([0-9]{1,2})\\s*[.]?\\s*(?:\\([^)]*\\))?\\s*(?:(\\d{1,2})\\s*:\\s*(\\d{2}))?\\s*(?:[~∼-]\\s*[‘'`]?(?:([0-9]{2,4})\\s*[.]\\s*)?([0-9]{1,2})\\s*[.]\\s*([0-9]{1,2})\\s*[.]?\\s*(?:\\([^)]*\\))?\\s*(?:(\\d{1,2})\\s*:\\s*(\\d{2}))?)?")
var firstDateMarkerPattern = regexp.MustCompile("[‘'`]?[0-9]{2,4}\\s*[.]\\s*[0-9]{1,2}\\s*[.]\\s*[0-9]{1,2}")
var procedureLabelSeparatorPattern = regexp.MustCompile("[▶➤]")

func InferSchedulesFromTextArtifacts(artifacts []extraction.ExtractedArtifact, noticeID int64) []NoticeScheduleCandidate {
	var schedules []NoticeScheduleCandidate
	seen := make(map[string]bool)
	for _, artifact := range artifacts {
		if !isTextScheduleArtifact(artifact.Type) || strings.TrimSpace(artifact.RawText) == "" {
			continue
		}
		for _, schedule := range inferApplicationSchedulesFromText(artifact, noticeID) {
			key := schedule.ScheduleType + "|" + schedule.DateText
			if seen[key] {
				continue
			}
			seen[key] = true
			schedules = append(schedules, schedule)
		}
	}
	return schedules
}

func isTextScheduleArtifact(artifactType extraction.ArtifactType) bool {
	switch artifactType {
	case extraction.ArtifactTypePDFText, extraction.ArtifactTypeHTMLPreview, extraction.ArtifactTypeHWPXText:
		return true
	default:
		return false
	}
}

func inferApplicationSchedulesFromText(artifact extraction.ExtractedArtifact, noticeID int64) []NoticeScheduleCandidate {
	text := compactScheduleText(artifact.RawText)
	matches := applicationSchedulePattern.FindAllStringSubmatch(text, -1)
	schedules := make([]NoticeScheduleCandidate, 0, len(matches))
	for _, match := range matches {
		start, okStart := scheduleTime(match[2], match[3], match[4], match[5], match[6], false)
		end, okEnd := scheduleTime(match[7], match[8], match[9], match[10], match[11], true)
		if !okStart || !okEnd {
			continue
		}
		label := strings.TrimSpace(match[1])
		dateText := strings.TrimSpace(match[0])
		schedules = append(schedules, NoticeScheduleCandidate{
			NoticeID:     noticeID,
			ScheduleType: "application",
			Label:        label,
			StartsAt:     start,
			EndsAt:       end,
			DateText:     dateText,
			Channel:      "source_text",
			SourceText:   dateText,
			SourceSpan:   artifact.SourceSpan + "#schedule=application",
			Confidence:   0.86,
		})
	}
	schedules = append(schedules, inferProcedureTableApplicationSchedule(artifact, noticeID, text)...)
	return schedules
}

func inferProcedureTableApplicationSchedule(artifact extraction.ExtractedArtifact, noticeID int64, text string) []NoticeScheduleCandidate {
	startIndex := procedureScheduleStartIndex(text)
	if startIndex < 0 {
		return nil
	}
	section := text[startIndex:]
	if marker := strings.Index(section, "※"); marker > 0 {
		section = section[:marker]
	}
	firstDate := firstDateMarkerPattern.FindStringIndex(section)
	if firstDate == nil {
		return nil
	}
	labelText := strings.TrimSpace(section[:firstDate[0]])
	dateText := strings.TrimSpace(section[firstDate[0]:])
	labels := procedureLabels(labelText)
	dateMatches := procedureDatePattern.FindAllStringSubmatch(dateText, -1)
	if len(labels) == 0 || len(dateMatches) == 0 {
		return nil
	}
	for i, label := range labels {
		if !strings.Contains(label, "신청접수") || i >= len(dateMatches) {
			continue
		}
		start, end, ok := procedureDateRange(dateMatches[i])
		if !ok {
			return nil
		}
		sourceText := strings.TrimSpace(dateMatches[i][0])
		return []NoticeScheduleCandidate{{
			NoticeID:     noticeID,
			ScheduleType: "application",
			Label:        label,
			StartsAt:     start,
			EndsAt:       end,
			DateText:     sourceText,
			Channel:      "source_text",
			SourceText:   sourceText,
			SourceSpan:   artifact.SourceSpan + "#schedule=application",
			Confidence:   0.82,
		}}
	}
	return nil
}

func procedureScheduleStartIndex(text string) int {
	markers := []string{"공급절차 및 일정", "■ 공급일정"}
	for _, marker := range markers {
		if index := strings.Index(text, marker); index >= 0 {
			return index
		}
	}
	return -1
}

func procedureLabels(text string) []string {
	text = strings.TrimSpace(strings.TrimPrefix(text, "공급절차 및 일정"))
	text = strings.TrimSpace(strings.TrimPrefix(text, "■ 공급일정"))
	parts := procedureLabelSeparatorPattern.Split(text, -1)
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			labels = append(labels, part)
		}
	}
	return labels
}

func procedureDateRange(match []string) (time.Time, time.Time, bool) {
	startYear := normalizeScheduleYear(match[1])
	start, okStart := scheduleTime(strconv.Itoa(startYear), match[2], match[3], match[4], match[5], false)
	if !okStart {
		return time.Time{}, time.Time{}, false
	}
	endYear := startYear
	endMonth := match[7]
	endDay := match[8]
	if match[6] != "" {
		endYear = normalizeScheduleYear(match[6])
	}
	if endMonth == "" || endDay == "" {
		return start, time.Date(startYear, time.Month(start.Month()), start.Day(), 23, 59, 59, 0, time.Local), true
	}
	end, okEnd := scheduleTime(strconv.Itoa(endYear), endMonth, endDay, match[9], match[10], true)
	return start, end, okEnd
}

func normalizeScheduleYear(raw string) int {
	year, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	if year < 100 {
		return 2000 + year
	}
	return year
}

func scheduleTime(year string, month string, day string, hour string, minute string, endOfDay bool) (time.Time, bool) {
	y, err := strconv.Atoi(year)
	if err != nil {
		return time.Time{}, false
	}
	m, err := strconv.Atoi(month)
	if err != nil {
		return time.Time{}, false
	}
	d, err := strconv.Atoi(day)
	if err != nil {
		return time.Time{}, false
	}
	h := 0
	min := 0
	if hour != "" {
		parsedHour, err := strconv.Atoi(hour)
		if err != nil {
			return time.Time{}, false
		}
		h = parsedHour
	}
	if minute != "" {
		parsedMinute, err := strconv.Atoi(minute)
		if err != nil {
			return time.Time{}, false
		}
		min = parsedMinute
	}
	if endOfDay && hour == "" && minute == "" {
		return time.Date(y, time.Month(m), d, 23, 59, 59, 0, time.Local), true
	}
	return time.Date(y, time.Month(m), d, h, min, 0, 0, time.Local), true
}

func compactScheduleText(text string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(strings.ReplaceAll(text, "\x00", "")), " "))
}
