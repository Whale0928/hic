package persistence

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"hic/pkg/discovery"
	"hic/pkg/llm"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestValidatePersistableCandidate_모집공고계열만허용한다(t *testing.T) {
	candidate := discovery.Candidate{
		Agency:    "SH",
		BoardKind: "rental",
		Seq:       "304295",
		Title:     "2026년 휘경마을 두레주택 잔여세대 입주자 모집공고",
	}

	if err := ValidatePersistableCandidate(candidate); err != nil {
		t.Fatalf("ValidatePersistableCandidate() error = %v", err)
	}
}

func TestValidatePersistableCandidate_당첨자발표는저장전차단한다(t *testing.T) {
	candidate := discovery.Candidate{
		Agency:    "SH",
		BoardKind: "rental",
		Seq:       "296353",
		Title:     "[당첨자 발표] 2025년 1차 청년 매입임대주택 입주자모집 당첨자 및 예비자 발표",
	}

	err := ValidatePersistableCandidate(candidate)

	if err == nil {
		t.Fatal("ValidatePersistableCandidate() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "non-recruitment") {
		t.Fatalf("error = %v, want non-recruitment", err)
	}
}

func TestLLMRepairArtifactRef_첨부기반Artifact만저장대상으로허용한다(t *testing.T) {
	artifact := LLMRepairArtifact{ID: 10, NoticeID: 20, AttachmentID: 30}

	ref, err := artifact.LLMRepairAttachmentRef()
	if err != nil {
		t.Fatalf("LLMRepairAttachmentRef() error = %v", err)
	}

	if ref.NoticeID != 20 || ref.AttachmentID != 30 {
		t.Fatalf("ref = %+v", ref)
	}
}

func TestLLMRepairArtifactRef_첨부없는Artifact는차단한다(t *testing.T) {
	artifact := LLMRepairArtifact{ID: 10, NoticeID: 20}

	err := artifact.ValidateLLMRepairOfferingTarget()

	if err == nil {
		t.Fatal("ValidateLLMRepairOfferingTarget() error = nil")
	}
	if !strings.Contains(err.Error(), "attachment-backed") {
		t.Fatalf("error = %v", err)
	}
}

func TestPrepareLLMRepairOfferings_성공출력을저장할후보로변환한다(t *testing.T) {
	count := 15
	output := llm.RepairOutput{
		Offerings: []llm.Offering{{
			ApplicationUnitLabel: "청담르엘 49 일반",
			HousingName:          "청담르엘",
			SupplyCount:          &count,
			SourceSpan:           "object://hic-originals/sh/304271/13-pamphlet.pdf#page=5&row=2",
			Confidence:           0.8,
		}},
	}

	offerings := PrepareLLMRepairOfferings(output)

	if len(offerings) != 1 {
		t.Fatalf("len(offerings) = %d, want 1", len(offerings))
	}
	if offerings[0].RawRow["source"] != "llm_repair" {
		t.Fatalf("RawRow = %+v", offerings[0].RawRow)
	}
}

func TestPrepareLLMRepairOfferings_빈근거후보는제외한다(t *testing.T) {
	output := llm.RepairOutput{
		Offerings: []llm.Offering{{
			ApplicationUnitLabel: "근거 없음",
			HousingName:          "테스트",
		}},
	}

	offerings := PrepareLLMRepairOfferings(output)

	if len(offerings) != 0 {
		t.Fatalf("offerings = %+v", offerings)
	}
}

func TestRepository_ApplicationNotice_테스트별클린DB에서UpsertLink한다(t *testing.T) {
	repo := openCleanTestRepository(t)
	ctx := context.Background()
	noticeID := insertSourceNoticeFixture(t, repo, "303584", "2026년 전세임대형 든든주택 입주자 모집공고(2026.04.29.)")
	count := 500
	posted := time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC)

	appID, err := repo.UpsertApplicationNotice(ctx, ApplicationNoticeInput{
		Agency:            "SH",
		Source:            "sh_app_user",
		SupplyType:        "12",
		RecruitNoticeCode: "202620092",
		RecruitType:       "32",
		Title:             "2026년 전세임대형 든든주택 입주자 모집 공고(2026.4.29.)",
		Status:            "청약중",
		SupplyCount:       &count,
		PostedAt:          posted,
	})
	if err != nil {
		t.Fatalf("UpsertApplicationNotice() error = %v", err)
	}

	if err := repo.LinkApplicationNoticeToSourceNotice(ctx, appID, noticeID); err != nil {
		t.Fatalf("LinkApplicationNoticeToSourceNotice() error = %v", err)
	}

	var linkedNoticeID int64
	if err := repo.pool.QueryRow(ctx, `select notice_id from application_notices where recrnoti_cd = '202620092'`).Scan(&linkedNoticeID); err != nil {
		t.Fatalf("query linked application notice: %v", err)
	}
	if linkedNoticeID != noticeID {
		t.Fatalf("linked notice_id = %d, want %d", linkedNoticeID, noticeID)
	}
}

func TestRepository_DiscoverySeenCache_테스트별클린DB에서FreshCache를조회한다(t *testing.T) {
	repo := openCleanTestRepository(t)
	ctx := context.Background()
	posted := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	expires := time.Now().Add(24 * time.Hour)

	if err := repo.UpsertDiscoverySeenCache(ctx, DiscoverySeenCacheInput{
		Agency:        "SH",
		BoardKind:     "rental",
		Seq:           "304999",
		Status:        discovery.SeenCacheStatusRejected,
		Reason:        discovery.NoticeCategoryRejected,
		Title:         "당첨자 발표 안내",
		PostedAt:      posted,
		ExpiresAt:     expires,
		PolicyVersion: "test-policy",
		ParserVersion: "test-parser",
		Evidence: map[string]any{
			"matched_keywords": []string{"당첨자"},
		},
	}); err != nil {
		t.Fatalf("UpsertDiscoverySeenCache() error = %v", err)
	}

	cache, err := repo.FreshDiscoverySeenCache(ctx, "SH", "rental", time.Now())
	if err != nil {
		t.Fatalf("FreshDiscoverySeenCache() error = %v", err)
	}
	entry, ok := cache["304999"]
	if !ok {
		t.Fatalf("cache missing seq 304999: %+v", cache)
	}
	if entry.Status != discovery.SeenCacheStatusRejected {
		t.Fatalf("Status = %q", entry.Status)
	}
	if !entry.CanSkipDetail(discovery.BoardRow{Seq: "304999", Title: "당첨자 발표 안내", PostedAt: posted}, time.Now()) {
		t.Fatalf("entry should skip matching rejected row: %+v", entry)
	}
}

func openCleanTestRepository(t *testing.T) *Repository {
	t.Helper()
	ctx := context.Background()
	adminURL := "postgres://shdata:shdata@localhost:9551/shdata?sslmode=disable"
	adminPool, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Skipf("skip DB integration test; admin open failed: %v", err)
	}
	if err := adminPool.Ping(ctx); err != nil {
		adminPool.Close()
		t.Skipf("skip DB integration test; admin ping failed: %v", err)
	}
	dbName := fmt.Sprintf("hic_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "create database "+dbName); err != nil {
		adminPool.Close()
		t.Skipf("skip DB integration test; create database failed: %v", err)
	}
	testURL := "postgres://shdata:shdata@localhost:9551/" + dbName + "?sslmode=disable"
	if err := Migrate(ctx, testURL); err != nil {
		_, _ = adminPool.Exec(ctx, "drop database if exists "+dbName+" with (force)")
		adminPool.Close()
		t.Skipf("skip DB integration test; migrate failed: %v", err)
	}
	repo, err := Open(ctx, testURL)
	if err != nil {
		_, _ = adminPool.Exec(ctx, "drop database if exists "+dbName+" with (force)")
		adminPool.Close()
		t.Skipf("skip DB integration test; open failed: %v", err)
	}
	cleanTestDB(t, repo)
	t.Cleanup(func() {
		repo.Close()
		_, _ = adminPool.Exec(context.Background(), "drop database if exists "+dbName+" with (force)")
		adminPool.Close()
	})
	return repo
}

func cleanTestDB(t *testing.T, repo *Repository) {
	t.Helper()
	_, err := repo.pool.Exec(context.Background(), `
truncate table
	qa_decisions,
	llm_repair_attempts,
	notice_schedules,
	offering_conversion_estimates,
	rent_conversion_rules,
	offerings,
	extracted_artifacts,
	attachment_extractions,
	attachments,
	application_notices,
	discovery_seen_cache,
	source_notices,
	raw_documents,
	stored_objects,
	source_boards,
	collection_runs
restart identity cascade
`)
	if err != nil {
		t.Fatalf("clean test DB: %v", err)
	}
}

func insertSourceNoticeFixture(t *testing.T, repo *Repository, seq string, title string) int64 {
	t.Helper()
	var id int64
	err := repo.pool.QueryRow(context.Background(), `
insert into source_notices (agency, board_kind, seq, category, notice_type, title, posted_at)
values ('SH', 'rental', $1, 'recruitment', 'recruitment', $2, '2026-04-29')
returning id
`, seq, title).Scan(&id)
	if err != nil {
		t.Fatalf("insert source notice fixture: %v", err)
	}
	return id
}
