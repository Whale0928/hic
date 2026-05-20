package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"hic/pkg/discovery"
	"hic/pkg/extraction"
	"hic/pkg/llm"
	"hic/pkg/normalize"
	"hic/pkg/persistence"

	"github.com/xuri/excelize/v2"
)

func TestNewRootCommand_도메인서브커맨드를노출한다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	help := out.String()
	for _, want := range []string{"serve", "discovery", "extract", "normalize", "llm", "workflow", "qa", "migrate"} {
		if !strings.Contains(help, want) {
			t.Fatalf("root help missing %q:\n%s", want, help)
		}
	}
}

func TestNewRootCommand_루트Help명령설명은한글이다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	help := out.String()
	for _, want := range []string{
		"completion  지정한 셸의 자동완성 스크립트를 생성합니다",
		"discovery   모집공고 후보를 발견합니다",
		"extract     첨부 원본에서 기계 추출 artifact를 생성합니다",
		"help        명령 도움말을 표시합니다",
		"llm         낮은 신뢰도의 artifact를 LLM 보조로 보정합니다",
		"migrate     PostgreSQL 스키마 마이그레이션을 실행합니다",
		"normalize   추출 artifact를 도메인 레코드로 정규화합니다",
		"qa          품질 게이트와 샘플 회귀 검사를 실행합니다",
		"serve       HIC HTTP API 서버를 시작합니다",
		"workflow    discovery, extraction, normalization, QA를 오케스트레이션합니다",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("root help missing Korean command description %q:\n%s", want, help)
		}
	}
}

func TestNewRootCommand_internal프로토타입명령을노출하지않는다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	help := out.String()
	for _, blocked := range []string{"collect-sh"} {
		if strings.Contains(help, blocked) {
			t.Fatalf("root help should not expose internal prototype command %q:\n%s", blocked, help)
		}
	}
}

func TestNewRootCommand_도메인서브커맨드Help가동작한다(t *testing.T) {
	tests := [][]string{
		{"discovery", "--help"},
		{"discovery", "sh", "--help"},
		{"serve", "--help"},
		{"extract", "--help"},
		{"extract", "hwp", "--help"},
		{"extract", "html", "--help"},
		{"extract", "hwpx", "--help"},
		{"normalize", "--help"},
		{"llm", "--help"},
		{"llm", "candidates", "--help"},
		{"llm", "repair", "--help"},
		{"workflow", "--help"},
		{"workflow", "collect-sh", "--help"},
		{"workflow", "collect-lh", "--help"},
		{"qa", "--help"},
		{"qa", "promote-offerings", "--help"},
		{"qa", "pdf-offerings", "--help"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd := NewRootCommand(context.Background())
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(args)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute(%v) error = %v", args, err)
			}
			if out.Len() == 0 {
				t.Fatalf("Execute(%v) produced empty help", args)
			}
		})
	}
}

func TestNewRootCommand_LLMRepairHelp가GPT보정옵션을노출한다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"llm", "repair", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	help := out.String()
	for _, want := range []string{"--artifact-id", "--dry-run", "--max-input-chars", "--max-attempts", "--model"} {
		if !strings.Contains(help, want) {
			t.Fatalf("llm repair help missing %q:\n%s", want, help)
		}
	}
}

func TestNewRootCommand_LLMCandidatesHelp가Limit옵션을노출한다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"llm", "candidates", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	help := out.String()
	for _, want := range []string{"--limit", "--include-approved-notices", "LLM"} {
		if !strings.Contains(help, want) {
			t.Fatalf("llm candidates help missing %q:\n%s", want, help)
		}
	}
}

func TestFormatLLMRepairCandidates_후보목록을출력한다(t *testing.T) {
	got := formatLLMRepairCandidates([]persistence.LLMRepairArtifact{{
		ID:               48,
		NoticeSeq:        "304555",
		ArtifactType:     "pdf_text",
		SourceSpan:       "object://hic-originals/sh/304555/notice.pdf",
		RawText:          "입주자 모집공고",
		OriginalFilename: "notice.pdf",
	}})

	for _, want := range []string{
		"llm_candidates=1",
		"artifact_id=48",
		"seq=304555",
		"raw_chars=8",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatLLMRepairCandidates() missing %q:\n%s", want, got)
		}
	}
}

func TestValidateLLMRepairAttemptLimit_최대시도수이상이면차단한다(t *testing.T) {
	err := validateLLMRepairAttemptLimit(1500, 1500)

	if err == nil {
		t.Fatal("validateLLMRepairAttemptLimit() error = nil")
	}
	if !strings.Contains(err.Error(), "maximum LLM repair attempts reached") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateLLMRepairAttemptLimit_최대시도수미만이면허용한다(t *testing.T) {
	if err := validateLLMRepairAttemptLimit(1499, 1500); err != nil {
		t.Fatalf("validateLLMRepairAttemptLimit() error = %v", err)
	}
}

func TestPersistLLMRepairOfferings_성공결과를PendingOffering으로저장한다(t *testing.T) {
	count := 15
	store := &fakeLLMRepairOfferingStore{}
	artifact := persistence.LLMRepairArtifact{ID: 10, NoticeID: 20, AttachmentID: 30}
	output := llm.RepairOutput{
		Offerings: []llm.Offering{{
			ApplicationUnitLabel: "청담르엘 49 일반",
			HousingName:          "청담르엘",
			SupplyCount:          &count,
			SourceSpan:           "object://hic-originals/sh/304271/13-pamphlet.pdf#page=5&row=2",
			Confidence:           0.8,
		}},
	}

	saved, err := persistLLMRepairOfferings(context.Background(), store, artifact, output)
	if err != nil {
		t.Fatalf("persistLLMRepairOfferings() error = %v", err)
	}

	if saved != 1 || len(store.upserts) != 1 {
		t.Fatalf("saved=%d upserts=%d", saved, len(store.upserts))
	}
	if store.deletedArtifactID != 10 {
		t.Fatalf("deletedArtifactID = %d, want 10", store.deletedArtifactID)
	}
	got := store.upserts[0]
	if got.attachment.NoticeID != 20 || got.attachment.AttachmentID != 30 || got.sourceArtifactID != 10 {
		t.Fatalf("upsert ref = %+v sourceArtifactID=%d", got.attachment, got.sourceArtifactID)
	}
	if got.offering.ApplicationUnitLabel != "청담르엘 49 일반" || got.offering.SourceSpan == "" {
		t.Fatalf("offering = %+v", got.offering)
	}
}

func TestNewRootCommand_LHCollectHelp가MyHome옵션을노출한다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"workflow", "collect-lh", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	help := out.String()
	for _, want := range []string{"--kind", "--service-key", "--num-rows", "--all-pages", "--agency-filter", "--show-items", "MyHome"} {
		if !strings.Contains(help, want) {
			t.Fatalf("collect-lh help missing %q:\n%s", want, help)
		}
	}
}

type fakeLLMRepairOfferingStore struct {
	deletedArtifactID int64
	upserts           []fakeLLMRepairOfferingUpsert
}

type fakeLLMRepairOfferingUpsert struct {
	attachment       persistence.PersistedAttachment
	sourceArtifactID int64
	offering         normalize.OfferingCandidate
}

func (f *fakeLLMRepairOfferingStore) UpsertOffering(ctx context.Context, attachment persistence.PersistedAttachment, sourceArtifactID int64, offering normalize.OfferingCandidate) (int64, error) {
	f.upserts = append(f.upserts, fakeLLMRepairOfferingUpsert{
		attachment:       attachment,
		sourceArtifactID: sourceArtifactID,
		offering:         offering,
	})
	return int64(len(f.upserts)), nil
}

func (f *fakeLLMRepairOfferingStore) DeleteLLMRepairOfferings(ctx context.Context, sourceArtifactID int64) error {
	f.deletedArtifactID = sourceArtifactID
	return nil
}

func TestNewRootCommand_SHApplicationsHelp를노출한다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"discovery", "sh-applications", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	help := out.String()
	for _, want := range []string{"--sply-ty", "--all-active", "--show-items"} {
		if !strings.Contains(help, want) {
			t.Fatalf("sh-applications help missing %q:\n%s", want, help)
		}
	}
}

func TestNewRootCommand_CollectSHHelp가ActiveApplication옵션을노출한다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"workflow", "collect-sh", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	help := out.String()
	for _, want := range []string{"--active-applications", "--active-sply-ty", "--active-max-pages", "--discovery-cache"} {
		if !strings.Contains(help, want) {
			t.Fatalf("collect-sh help missing %q:\n%s", want, help)
		}
	}
}

func TestNewRootCommand_HWPXExtractHelp가File옵션을노출한다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"extract", "hwpx", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	help := out.String()
	for _, want := range []string{"HWPX", "--file"} {
		if !strings.Contains(help, want) {
			t.Fatalf("extract hwpx help missing %q:\n%s", want, help)
		}
	}
}

func TestNewRootCommand_HWPExtractHelp가File옵션을노출한다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"extract", "hwp", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	help := out.String()
	for _, want := range []string{"HWP", "--file"} {
		if !strings.Contains(help, want) {
			t.Fatalf("extract hwp help missing %q:\n%s", want, help)
		}
	}
}

func TestNewRootCommand_Serve기본포트는9552다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"serve", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), ":9552") {
		t.Fatalf("serve help = %q, want default :9552", out.String())
	}
}

func TestNewRootCommand_QAHelp가승격명령을노출한다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"qa", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "promote-offerings") {
		t.Fatalf("qa help = %q, want promote-offerings", out.String())
	}
	if !strings.Contains(out.String(), "pdf-offerings") {
		t.Fatalf("qa help = %q, want pdf-offerings", out.String())
	}
}

func TestFormatQASummary_승격결과를출력한다(t *testing.T) {
	got := formatQASummary(persistence.QASummary{Approved: 25, Rejected: 2, Pending: 1})

	if got != "qa approved=25 rejected=2 pending=1\n" {
		t.Fatalf("formatQASummary() = %q", got)
	}
}

func TestFormatPDFOfferings_공급항목을마크다운표로출력한다(t *testing.T) {
	count := 34
	area := 59.0
	jeonse := int64(514020000)
	offering := normalize.OfferingCandidate{
		ApplicationUnitLabel: "세곡2지구 59㎡ 일반 여성",
		HousingName:          "세곡2지구",
		UnitNo:               "",
		ExclusiveAreaM2:      &area,
		SupplyCount:          &count,
		JeonseDepositKRW:     &jeonse,
		GenderRequirement:    "여성",
		SourceSpan:           "pdf://sample.pdf#table=1&row=1",
		Confidence:           0.82,
	}

	got := formatPDFOfferings([]normalize.OfferingCandidate{offering})

	for _, want := range []string{
		"offerings=1",
		"| # | 신청 가능 단위 | 주택명 | 호실 | 면적(㎡) | 공급호수 | 전세금액 | 보증금 | 월임대료 | 기숙사비 | 성별 | source | confidence |",
		"| 1 | 세곡2지구 59㎡ 일반 여성 | 세곡2지구 |  | 59 | 34 | 514020000 |  |  |  | 여성 | pdf://sample.pdf#table=1&row=1 | 0.82 |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatPDFOfferings() missing %q:\n%s", want, got)
		}
	}
}

func TestFormatCollectionSummary_QA승격결과를함께출력한다(t *testing.T) {
	got := formatCollectionSummary(7, 31, 28, 31, 42, 28, persistence.QASummary{
		Approved: 28,
		Rejected: 0,
		Pending:  0,
	})

	want := "db stored_objects=31 extracted_artifacts=42 offerings=28 upserted_artifacts=31 upserted_offerings=28 qa_approved=28 qa_rejected=0 qa_pending=0\n"
	if got != want {
		t.Fatalf("formatCollectionSummary() = %q, want %q", got, want)
	}
}

func TestNormalizeOfferingsFromArtifacts_PDF신청주택표를정규화한다(t *testing.T) {
	offerings := normalizeOfferingsFromArtifacts(extraction.AttachmentKindNoticePDF, []extraction.ExtractedArtifact{{
		Type:       extraction.ArtifactTypePDFText,
		SourceSpan: "pdf://sample.pdf",
		RawText: `
신청 주택 주소 서울특별시 금천구 시흥대로 88 길 18
공급호실 방 개수 면적 ( ㎡ ) 임대조건 ( 원 )
계 전용 공용 보증금 임대료
502 호 0 60.64 47.09 13.55 42,000,000 495,300
입주가능일 2026.08.20.
`,
	}})

	if len(offerings) != 1 {
		t.Fatalf("normalizeOfferingsFromArtifacts() len = %d, want 1", len(offerings))
	}
	if offerings[0].UnitNo != "502호" {
		t.Fatalf("UnitNo = %q", offerings[0].UnitNo)
	}
}

func TestNewRootCommand_ExtractXLSX_Artifact개수를출력한다(t *testing.T) {
	path := filepath.Join(t.TempDir(), "offerings.xlsx")
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	for r, row := range [][]any{{"동", "호"}, {"101", "1201"}} {
		for c, value := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				t.Fatalf("CoordinatesToCellName() error = %v", err)
			}
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				t.Fatalf("SetCellValue() error = %v", err)
			}
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs() error = %v", err)
	}

	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"extract", "xlsx", "--file", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "artifacts=2") {
		t.Fatalf("output = %q, want artifacts=2", out.String())
	}
}

func TestWriteDiscoveryReport_첨부상세옵션이면파일명을출력한다(t *testing.T) {
	report := discovery.Report{
		Agency:    "SH",
		BoardKind: "rental",
		Candidates: []discovery.Candidate{
			{
				Seq:   "304000",
				Title: "입주자 모집공고",
				Attachments: []discovery.AttachmentMeta{
					{FileSeq: "1", Filename: "공고문.pdf", Size: "100"},
					{FileSeq: "2", Filename: "공급대상 주택목록.xlsx", Size: "200"},
				},
			},
		},
	}
	var out bytes.Buffer

	writeDiscoveryReport(&out, report, true)

	got := out.String()
	if !strings.Contains(got, "candidate seq=304000") {
		t.Fatalf("output missing candidate line:\n%s", got)
	}
	if !strings.Contains(got, `attachment seq=304000 file_seq=2 filename="공급대상 주택목록.xlsx" size=200`) {
		t.Fatalf("output missing attachment detail:\n%s", got)
	}
}

func TestCollectionRunStats_수집통계를구성한다(t *testing.T) {
	report := discovery.Report{
		Pages:           5,
		ListRows:        50,
		Details:         44,
		SkippedOld:      6,
		SkippedKnown:    3,
		StoppedByCutoff: true,
		Candidates:      make([]discovery.Candidate, 14),
		Rejected:        make([]discovery.RejectedPost, 30),
	}

	stats := collectionRunStats(report, 7, 31, 25, 7, 31, 25)

	tests := map[string]any{
		"pages":              5,
		"list_rows":          50,
		"details":            44,
		"candidates":         14,
		"rejected":           30,
		"skipped_old":        6,
		"skipped_known":      3,
		"stopped_by_cutoff":  true,
		"downloaded":         7,
		"upserted_artifacts": 31,
		"upserted_offerings": 25,
		"stored_objects":     int64(7),
		"total_artifacts":    int64(31),
		"total_offerings":    int64(25),
	}
	for key, want := range tests {
		if got := stats[key]; got != want {
			t.Fatalf("stats[%s] = %#v, want %#v", key, got, want)
		}
	}
}
