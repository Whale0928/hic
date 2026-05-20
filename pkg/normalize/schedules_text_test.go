package normalize

import (
	"testing"
	"time"

	"hic/pkg/extraction"
)

func TestInferSchedulesFromTextArtifacts_신청접수기간을구조화한다(t *testing.T) {
	artifacts := []extraction.ExtractedArtifact{
		{
			Type:       extraction.ArtifactTypePDFText,
			SourceSpan: "object://hic-originals/sh/304555/1-notice.pdf",
			RawText:    "청약신청 접수기간 : 2026. 6. 5.(금) 10:00 ~ 2026. 6. 9.(화) 17:00 인터넷 청약",
		},
	}

	schedules := InferSchedulesFromTextArtifacts(artifacts, 42)

	if len(schedules) != 1 {
		t.Fatalf("len(schedules) = %d, want 1", len(schedules))
	}
	got := schedules[0]
	if got.NoticeID != 42 || got.ScheduleType != "application" || got.Label != "청약신청 접수기간" {
		t.Fatalf("schedule = %+v", got)
	}
	if !got.StartsAt.Equal(time.Date(2026, 6, 5, 10, 0, 0, 0, time.Local)) {
		t.Fatalf("StartsAt = %s", got.StartsAt)
	}
	if !got.EndsAt.Equal(time.Date(2026, 6, 9, 17, 0, 0, 0, time.Local)) {
		t.Fatalf("EndsAt = %s", got.EndsAt)
	}
	if got.SourceSpan != "object://hic-originals/sh/304555/1-notice.pdf#schedule=application" {
		t.Fatalf("SourceSpan = %q", got.SourceSpan)
	}
	if got.Confidence < 0.8 {
		t.Fatalf("Confidence = %f", got.Confidence)
	}
}

func TestInferSchedulesFromTextArtifacts_날짜만있으면종료일은하루끝으로둔다(t *testing.T) {
	artifacts := []extraction.ExtractedArtifact{
		{
			Type:       extraction.ArtifactTypeHTMLPreview,
			SourceSpan: "object://hic-artifacts/sh/304555/1-preview.html",
			RawText:    "신청접수 2026.06.05 ~ 2026.06.09",
		},
	}

	schedules := InferSchedulesFromTextArtifacts(artifacts, 42)

	if len(schedules) != 1 {
		t.Fatalf("len(schedules) = %d, want 1", len(schedules))
	}
	got := schedules[0]
	if !got.StartsAt.Equal(time.Date(2026, 6, 5, 0, 0, 0, 0, time.Local)) {
		t.Fatalf("StartsAt = %s", got.StartsAt)
	}
	if !got.EndsAt.Equal(time.Date(2026, 6, 9, 23, 59, 59, 0, time.Local)) {
		t.Fatalf("EndsAt = %s", got.EndsAt)
	}
}

func TestInferSchedulesFromTextArtifacts_SH공급절차표에서신청접수를구조화한다(t *testing.T) {
	artifacts := []extraction.ExtractedArtifact{
		{
			Type:       extraction.ArtifactTypePDFText,
			SourceSpan: "object://hic-originals/sh/304555/1-notice.pdf",
			RawText:   "▣ 공급절차 및 일정 입주자 모집 공고 ▶ 주택사전공개 ▶ 신청접수 ▶ 서류심사 대상자 발표 ▶ 계약체결 ‘26. 5. 19. ( 화 ) `26. 5. 27.( 수 ) ~5. 28.( 목 ) `26. 6. 1.( 월 ) ~ 6. 2.( 화 ) `26. 6. 15. ( 월 ) `26. 8. 10.( 월 ) ~ 8. 11.( 화 ) ※ 상기 공급일정은 변경될 수 있습니다.",
		},
	}

	schedules := InferSchedulesFromTextArtifacts(artifacts, 42)

	if len(schedules) != 1 {
		t.Fatalf("len(schedules) = %d, want 1", len(schedules))
	}
	got := schedules[0]
	if got.Label != "신청접수" {
		t.Fatalf("Label = %q", got.Label)
	}
	if !got.StartsAt.Equal(time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)) {
		t.Fatalf("StartsAt = %s", got.StartsAt)
	}
	if !got.EndsAt.Equal(time.Date(2026, 6, 2, 23, 59, 59, 0, time.Local)) {
		t.Fatalf("EndsAt = %s", got.EndsAt)
	}
}
