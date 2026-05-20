package sh

import (
	"testing"
	"time"

	"hic/pkg/discovery"
)

func TestReconcileApplications_제목날짜로게시판후보를연결한다(t *testing.T) {
	posted := time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC)
	applications := []ApplicationNotice{{
		RecruitNoticeCode: "202620092",
		SupplyType:        "12",
		Title:             "2026년 전세임대형 든든주택 입주자 모집 공고(2026.4.29.)",
		PostedAt:          posted,
		Status:            StatusOpen,
	}}
	candidates := []discovery.Candidate{{
		Agency:    "SH",
		BoardKind: "rental",
		Seq:       "303584",
		Title:     "2026년 전세임대형 든든주택 입주자 모집공고(2026.04.29.)",
		PostedAt:  posted,
	}}

	result := ReconcileApplications(applications, candidates)

	if len(result.Linked) != 1 {
		t.Fatalf("len(Linked) = %d, want 1: %+v", len(result.Linked), result)
	}
	linked := result.Linked[0]
	if linked.RecruitNoticeCode != "202620092" || linked.BoardSeq != "303584" {
		t.Fatalf("linked = %+v", linked)
	}
	if len(result.UnmatchedApplications) != 0 {
		t.Fatalf("UnmatchedApplications = %+v", result.UnmatchedApplications)
	}
}

func TestActiveTargetTitles_청약중접수예정만대상으로한다(t *testing.T) {
	applications := []ApplicationNotice{
		{Title: "청약중 공고", Status: StatusOpen},
		{Title: "접수예정 공고", Status: StatusPending},
		{Title: "마감 공고", Status: ApplicationStatus("마감")},
	}

	titles := ActiveTargetTitles(applications)

	if len(titles) != 2 {
		t.Fatalf("titles = %+v, want 2 active titles", titles)
	}
}
