package sh

import (
	"regexp"
	"strings"
	"time"

	"hic/pkg/discovery"
)

type ApplicationBoardLink struct {
	RecruitNoticeCode string
	SupplyType        string
	BoardSeq          string
	Title             string
	Status            ApplicationStatus
}

type ReconcileResult struct {
	Linked                []ApplicationBoardLink
	UnmatchedApplications []ApplicationNotice
}

func ReconcileApplications(applications []ApplicationNotice, candidates []discovery.Candidate) ReconcileResult {
	candidatesByKey := make(map[string]discovery.Candidate, len(candidates))
	for _, candidate := range candidates {
		candidatesByKey[reconcileKey(candidate.Title, candidate.PostedAt)] = candidate
	}

	result := ReconcileResult{}
	for _, application := range applications {
		candidate, ok := candidatesByKey[reconcileKey(application.Title, application.PostedAt)]
		if !ok {
			result.UnmatchedApplications = append(result.UnmatchedApplications, application)
			continue
		}
		result.Linked = append(result.Linked, ApplicationBoardLink{
			RecruitNoticeCode: application.RecruitNoticeCode,
			SupplyType:        application.SupplyType,
			BoardSeq:          candidate.Seq,
			Title:             candidate.Title,
			Status:            application.Status,
		})
	}
	return result
}

func ActiveTargetTitles(applications []ApplicationNotice) []string {
	titles := make([]string, 0, len(applications))
	seen := map[string]bool{}
	for _, application := range applications {
		if application.Status != StatusOpen && application.Status != StatusPending {
			continue
		}
		normalized := NormalizeNoticeTitle(application.Title)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		titles = append(titles, normalized)
	}
	return titles
}

func reconcileKey(title string, postedAt time.Time) string {
	if postedAt.IsZero() {
		return NormalizeNoticeTitle(title)
	}
	return NormalizeNoticeTitle(title) + "|" + postedAt.Format(time.DateOnly)
}

var titleNoisePattern = regexp.MustCompile(`[\s\p{P}\p{S}]+`)

func NormalizeNoticeTitle(title string) string {
	normalized := strings.ToLower(strings.TrimSpace(title))
	normalized = strings.ReplaceAll(normalized, "모집 공고", "모집공고")
	normalized = strings.ReplaceAll(normalized, "입주자 모집", "입주자모집")
	normalized = strings.ReplaceAll(normalized, "2026.04.29", "20260429")
	normalized = strings.ReplaceAll(normalized, "2026.4.29", "20260429")
	normalized = titleNoisePattern.ReplaceAllString(normalized, "")
	return normalized
}
