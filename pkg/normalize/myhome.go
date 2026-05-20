package normalize

import (
	"fmt"
	"strings"
	"time"

	lhdiscovery "hic/pkg/discovery/lh"
)

type NoticeScheduleCandidate struct {
	NoticeID         int64
	SourceArtifactID int64
	ScheduleType     string
	Label            string
	StartsAt         time.Time
	EndsAt           time.Time
	DateText         string
	Channel          string
	Note             string
	SourceText       string
	SourceSpan       string
	Confidence       float64
}

func OfferingFromMyHomeItem(item lhdiscovery.MyHomeNoticeItem, sourceSpan string) OfferingCandidate {
	labelParts := []string{item.ComplexName, item.SupplyType, item.HouseType}
	if item.SupplyCount != nil && *item.SupplyCount > 0 {
		labelParts = append(labelParts, fmt.Sprintf("%d호", *item.SupplyCount))
	}
	return OfferingCandidate{
		ApplicationUnitLabel: joinApplicationLabel(labelParts...),
		SupplyCategory:       item.SupplyType,
		District:             strings.TrimSpace(strings.Join(nonEmptyStrings(item.Province, item.City), " ")),
		Address:              item.Address,
		LegalDong:            item.LegalDong,
		HousingName:          item.ComplexName,
		ComplexName:          item.ComplexName,
		UnitType:             item.HouseType,
		DepositKRW:           cloneInt64Ptr(item.DepositKRW),
		MonthlyRentKRW:       cloneInt64Ptr(item.MonthlyRent),
		SupplyCount:          cloneIntPtr(item.SupplyCount),
		HeatingMethod:        item.HeatingMethod,
		SourceSpan:           sourceSpan,
		RawRow: map[string]any{
			"source":     "myhome_api",
			"pblanc_id":  item.NoticeID,
			"house_sn":   item.HouseSN,
			"detail_url": item.DetailURL,
			"source_url": item.SourceURL,
		},
		Confidence: 0.95,
	}
}

func ApplicationScheduleFromMyHomeItem(item lhdiscovery.MyHomeNoticeItem, noticeID int64, sourceSpan string) (NoticeScheduleCandidate, bool) {
	start, okStart := parseMyHomeDate(item.ApplicationBeg, false)
	end, okEnd := parseMyHomeDate(item.ApplicationEnd, true)
	if !okStart && !okEnd {
		return NoticeScheduleCandidate{}, false
	}
	return NoticeScheduleCandidate{
		NoticeID:     noticeID,
		ScheduleType: "application",
		Label:        "신청접수",
		StartsAt:     start,
		EndsAt:       end,
		DateText:     strings.TrimSpace(item.ApplicationBeg + "~" + item.ApplicationEnd),
		Channel:      "myhome",
		SourceText:   strings.TrimSpace(item.ApplicationBeg + " " + item.ApplicationEnd),
		SourceSpan:   sourceSpan + "#schedule=application",
		Confidence:   1,
	}, true
}

func parseMyHomeDate(value string, endOfDay bool) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if len(value) != 8 {
		return time.Time{}, false
	}
	parsed, err := time.ParseInLocation("20060102", value, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Second)
	}
	return parsed, true
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func joinApplicationLabel(values ...string) string {
	return strings.Join(nonEmptyStrings(values...), " ")
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	out := *value
	return &out
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	out := *value
	return &out
}
