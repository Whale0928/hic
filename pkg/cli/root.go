package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"hic/pkg/api"
	"hic/pkg/discovery"
	lhdiscovery "hic/pkg/discovery/lh"
	shdiscovery "hic/pkg/discovery/sh"
	"hic/pkg/extraction"
	"hic/pkg/global"
	"hic/pkg/llm"
	"hic/pkg/normalize"
	"hic/pkg/persistence"
	"hic/pkg/workflow"

	"github.com/spf13/cobra"
)

func NewRootCommand(ctx context.Context) *cobra.Command {
	cfg := global.FromEnv()
	root := &cobra.Command{
		Use:          "hic",
		Short:        "공공주택 모집공고 수집/정규화 도구",
		SilenceUsage: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpCommand(newHelpCommand())

	root.AddCommand(
		newCompletionCommand(),
		newMigrateCommand(ctx, cfg),
		newServeCommand(ctx, cfg),
		newDiscoveryCommand(ctx, cfg),
		newExtractCommand(),
		newNormalizeCommand(),
		newLLMCommand(ctx, cfg),
		newWorkflowCommand(ctx, cfg),
		newQACommand(ctx, cfg),
	)

	return root
}

func newServeCommand(ctx context.Context, cfg global.Config) *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "HIC HTTP API 서버를 시작합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := persistence.Open(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer repo.Close()
			return api.New(repo).Start(addr)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", ":9552", "HTTP 수신 주소")
	return cmd
}

func newMigrateCommand(ctx context.Context, cfg global.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "PostgreSQL 스키마 마이그레이션을 실행합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			return persistence.Migrate(ctx, cfg.DatabaseURL)
		},
	}
}

func newDiscoveryCommand(ctx context.Context, cfg global.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discovery",
		Short: "모집공고 후보를 발견합니다",
	}

	var boardKind string
	var pages int
	var dryRun bool
	var seqs string
	var showAttachments bool
	var maxAgeDays int
	var skipExisting bool
	shCmd := &cobra.Command{
		Use:   "sh",
		Short: "SH 게시판에서 모집공고 후보를 발견합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !dryRun {
				return fmt.Errorf("discovery persistence is not implemented yet; use --dry-run")
			}
			registry := discovery.NewStaticSiteRegistry()
			board, ok := registry.Get("SH", boardKind)
			if !ok {
				return fmt.Errorf("unknown SH board: %s", boardKind)
			}
			knownSeqs, err := knownSeqsForDiscovery(ctx, cfg, board, skipExisting)
			if err != nil {
				return err
			}
			report, err := discovery.NewDiscoverer(discovery.NewHTTPFetcher()).Discover(ctx, board, discovery.Options{
				Pages:      pages,
				Seqs:       splitCSV(seqs),
				CutoffDate: cutoffDate(maxAgeDays),
				KnownSeqs:  knownSeqs,
			})
			if err != nil {
				return err
			}
			writeDiscoveryReport(cmd.OutOrStdout(), report, showAttachments)
			return nil
		},
	}
	shCmd.Flags().StringVar(&boardKind, "board", "rental", "SH 게시판 종류: rental 또는 sale")
	shCmd.Flags().IntVar(&pages, "pages", 1, "조회할 게시판 페이지 수")
	shCmd.Flags().StringVar(&seqs, "seq", "", "쉼표로 구분한 원본 공고 seq 필터")
	shCmd.Flags().BoolVar(&dryRun, "dry-run", true, "DB 저장 없이 후보만 보고합니다")
	shCmd.Flags().BoolVar(&showAttachments, "show-attachments", false, "후보 첨부 메타데이터를 출력합니다")
	shCmd.Flags().IntVar(&maxAgeDays, "max-age-days", 30, "지정 일수보다 오래된 공고는 제외합니다. 0이면 비활성화합니다")
	shCmd.Flags().BoolVar(&skipExisting, "skip-existing", false, "PostgreSQL을 조회해 이미 수집한 모집공고를 건너뜁니다")
	cmd.AddCommand(shCmd)
	cmd.AddCommand(newDiscoverySHApplicationsCommand(ctx))

	return cmd
}

func newDiscoverySHApplicationsCommand(ctx context.Context) *cobra.Command {
	var splyTy string
	var allActive bool
	var dryRun bool
	var showItems bool

	cmd := &cobra.Command{
		Use:   "sh-applications",
		Short: "SH 인터넷청약 상태 공고를 발견합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !dryRun {
				return fmt.Errorf("SH application discovery persistence is handled by workflow collect-sh; use --dry-run")
			}
			endpoints, err := shdiscovery.DefaultApplicationEndpoints().Select(splitCSV(splyTy))
			if err != nil {
				return err
			}
			client := shdiscovery.NewHTTPClient()
			applications := make([]shdiscovery.ApplicationNotice, 0)
			for _, endpoint := range endpoints {
				rows, err := client.ListApplications(ctx, endpoint)
				if err != nil {
					return err
				}
				applications = append(applications, rows...)
			}
			if allActive {
				applications = filterActiveApplications(applications)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "agency=SH source=sh_app_user endpoints=%d applications=%d dry_run=%t\n", len(endpoints), len(applications), dryRun)
			if showItems {
				writeSHApplications(cmd.OutOrStdout(), applications)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&splyTy, "sply-ty", "", "쉼표로 구분한 SH 공급유형 코드. 비우면 기본 인터넷청약 전체를 조회합니다")
	cmd.Flags().BoolVar(&allActive, "all-active", false, "청약중/접수예정 상태만 출력합니다")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "DB 저장 없이 SH 인터넷청약 상태만 보고합니다")
	cmd.Flags().BoolVar(&showItems, "show-items", false, "발견한 SH 인터넷청약 상태 행을 출력합니다")
	return cmd
}

func knownSeqsForDiscovery(ctx context.Context, cfg global.Config, board discovery.Board, enabled bool) (map[string]bool, error) {
	if !enabled {
		return nil, nil
	}
	repo, err := persistence.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	defer repo.Close()
	return repo.ExistingNoticeSeqs(ctx, board.Agency, board.BoardKind)
}

func cutoffDate(maxAgeDays int) time.Time {
	if maxAgeDays <= 0 {
		return time.Time{}
	}
	return time.Now().AddDate(0, 0, -maxAgeDays)
}

func newCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "지정한 셸의 자동완성 스크립트를 생성합니다",
		Long:  "bash, zsh, fish, powershell용 자동완성 스크립트를 생성합니다.",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "bash",
			Short: "bash 자동완성 스크립트를 생성합니다",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Root().GenBashCompletionV2(cmd.OutOrStdout(), true)
			},
		},
		&cobra.Command{
			Use:   "zsh",
			Short: "zsh 자동완성 스크립트를 생성합니다",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			},
		},
		&cobra.Command{
			Use:   "fish",
			Short: "fish 자동완성 스크립트를 생성합니다",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			},
		},
		&cobra.Command{
			Use:   "powershell",
			Short: "PowerShell 자동완성 스크립트를 생성합니다",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			},
		},
	)
	return cmd
}

func newHelpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "명령 도움말을 표시합니다",
		Long:  "지정한 명령의 상세 도움말을 표시합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, _, err := cmd.Root().Find(args)
			if err != nil || target == nil {
				return fmt.Errorf("알 수 없는 도움말 주제: %s", strings.Join(args, " "))
			}
			return target.Help()
		},
	}
}

func collectionRunStats(report discovery.Report, downloaded int, upsertedArtifacts int, upsertedOfferings int, storedObjects int64, totalArtifacts int64, totalOfferings int64) map[string]any {
	return map[string]any{
		"pages":              report.Pages,
		"list_rows":          report.ListRows,
		"details":            report.Details,
		"candidates":         len(report.Candidates),
		"rejected":           len(report.Rejected),
		"skipped_old":        report.SkippedOld,
		"skipped_known":      report.SkippedKnown,
		"skipped_seen":       report.SkippedSeen,
		"stopped_by_cutoff":  report.StoppedByCutoff,
		"downloaded":         downloaded,
		"upserted_artifacts": upsertedArtifacts,
		"upserted_offerings": upsertedOfferings,
		"stored_objects":     storedObjects,
		"total_artifacts":    totalArtifacts,
		"total_offerings":    totalOfferings,
	}
}

func writeDiscoveryReport(w io.Writer, report discovery.Report, showAttachments bool) {
	fmt.Fprintln(w, report.String())
	for _, candidate := range report.Candidates {
		fmt.Fprintf(w, "candidate seq=%s title=%q attachments=%d\n", candidate.Seq, candidate.Title, len(candidate.Attachments))
		if !showAttachments {
			continue
		}
		for _, attachment := range candidate.Attachments {
			fmt.Fprintf(
				w,
				"  attachment seq=%s file_seq=%s filename=%q size=%s\n",
				firstNonEmptyString(attachment.Seq, candidate.Seq),
				attachment.FileSeq,
				attachment.DisplayFilename(),
				attachment.DisplaySize(),
			)
		}
	}
	for _, rejected := range report.Rejected {
		fmt.Fprintf(w, "rejected seq=%s reason=%s title=%q\n", rejected.Seq, rejected.Reason, rejected.Title)
	}
}

func writeSHApplications(w io.Writer, applications []shdiscovery.ApplicationNotice) {
	for _, application := range applications {
		fmt.Fprintf(
			w,
			"application recrnoti_cd=%s sply_ty=%s recr_ty=%s status=%s supply_count=%s posted_at=%s title=%q\n",
			application.RecruitNoticeCode,
			application.SupplyType,
			application.RecruitType,
			application.Status,
			intPtrString(application.SupplyCount),
			dateString(application.PostedAt),
			application.Title,
		)
	}
}

func filterActiveApplications(applications []shdiscovery.ApplicationNotice) []shdiscovery.ApplicationNotice {
	filtered := make([]shdiscovery.ApplicationNotice, 0, len(applications))
	for _, application := range applications {
		if application.Status == shdiscovery.StatusOpen || application.Status == shdiscovery.StatusPending {
			filtered = append(filtered, application)
		}
	}
	return filtered
}

func collectSHApplicationNotices(ctx context.Context, supplyTypes []string) ([]shdiscovery.ApplicationNotice, error) {
	endpoints, err := shdiscovery.DefaultApplicationEndpoints().Select(supplyTypes)
	if err != nil {
		return nil, err
	}
	client := shdiscovery.NewHTTPClient()
	applications := make([]shdiscovery.ApplicationNotice, 0)
	for _, endpoint := range endpoints {
		rows, err := client.ListApplications(ctx, endpoint)
		if err != nil {
			return nil, err
		}
		applications = append(applications, rows...)
	}
	return filterActiveApplications(applications), nil
}

func writeSHApplicationLinks(w io.Writer, result shdiscovery.ReconcileResult) {
	for _, link := range result.Linked {
		fmt.Fprintf(w, "active_application recrnoti_cd=%s sply_ty=%s status=%s board_seq=%s title=%q\n", link.RecruitNoticeCode, link.SupplyType, link.Status, link.BoardSeq, link.Title)
	}
	for _, application := range result.UnmatchedApplications {
		fmt.Fprintf(w, "unmatched_application recrnoti_cd=%s sply_ty=%s status=%s title=%q\n", application.RecruitNoticeCode, application.SupplyType, application.Status, application.Title)
	}
}

func applicationNoticeInput(application shdiscovery.ApplicationNotice) persistence.ApplicationNoticeInput {
	return persistence.ApplicationNoticeInput{
		Agency:            "SH",
		Source:            "sh_app_user",
		SupplyType:        application.SupplyType,
		RecruitNoticeCode: application.RecruitNoticeCode,
		RecruitType:       application.RecruitType,
		NoticeNoHouse:     application.NoticeNoHouse,
		RegionPriority:    application.RegionPriority,
		Title:             application.Title,
		Status:            string(application.Status),
		SupplyCount:       application.SupplyCount,
		PostedAt:          application.PostedAt,
		RawMetadata: map[string]any{
			"raw_status": application.RawStatus,
		},
	}
}

func dateString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.DateOnly)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func newExtractCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extract",
		Short: "첨부 원본에서 기계 추출 artifact를 생성합니다",
	}
	cmd.AddCommand(
		placeholderCommand("attachment", "저장된 첨부 원본을 추출합니다"),
		newExtractHWPCommand(),
		newExtractHTMLCommand(),
		newExtractHWPXCommand(),
		newExtractPDFCommand(),
		newExtractXLSXCommand(),
	)
	return cmd
}

func newExtractHWPCommand() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "hwp",
		Short: "HWP 파일에서 텍스트 artifact를 추출합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			artifact, err := extraction.ExtractHWPTextWithSource(file, "")
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "type=%s status=%s chars=%d source=%s\n", artifact.Type, artifact.Status, len([]rune(artifact.RawText)), artifact.SourceSpan)
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "HWP 파일 경로")
	return cmd
}

func newExtractHTMLCommand() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "html",
		Short: "HTML 미리보기에서 텍스트 artifact를 추출합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			artifact, err := extraction.ExtractHTMLPreview(file)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "type=%s status=%s chars=%d source=%s\n", artifact.Type, artifact.Status, len([]rune(artifact.RawText)), artifact.SourceSpan)
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "HTML 파일 경로")
	return cmd
}

func newExtractHWPXCommand() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "hwpx",
		Short: "HWPX 파일에서 텍스트 artifact를 추출합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			artifact, err := extraction.ExtractHWPXText(file)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "type=%s status=%s chars=%d source=%s\n", artifact.Type, artifact.Status, len([]rune(artifact.RawText)), artifact.SourceSpan)
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "HWPX 파일 경로")
	return cmd
}

func newExtractPDFCommand() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "pdf",
		Short: "PDF 파일에서 텍스트를 추출합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			artifact, err := extraction.ExtractPDFText(file)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "type=%s status=%s chars=%d source=%s\n", artifact.Type, artifact.Status, len([]rune(artifact.RawText)), artifact.SourceSpan)
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "PDF 파일 경로")
	return cmd
}

func newExtractXLSXCommand() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "xlsx",
		Short: "XLSX 파일에서 행 artifact를 추출합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			artifacts, err := extraction.ExtractXLSXRows(file)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "type=%s artifacts=%d\n", extraction.ArtifactTypeXLSXRow, len(artifacts))
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "XLSX 파일 경로")
	return cmd
}

func newNormalizeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "normalize",
		Short: "추출 artifact를 도메인 레코드로 정규화합니다",
	}
	cmd.AddCommand(
		placeholderCommand("notice", "공고 메타데이터를 정규화합니다"),
		placeholderCommand("offerings", "공급항목을 정규화합니다"),
		placeholderCommand("schedules", "공고 일정을 정규화합니다"),
		placeholderCommand("conversion", "임대료-보증금 전환 규칙을 정규화합니다"),
	)
	return cmd
}

func newLLMCommand(ctx context.Context, cfg global.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "llm",
		Short: "낮은 신뢰도의 artifact를 LLM 보조로 보정합니다",
	}
	cmd.AddCommand(
		newLLMCandidatesCommand(ctx, cfg),
		newLLMRepairCommand(ctx, cfg),
	)
	return cmd
}

func newLLMCandidatesCommand(ctx context.Context, cfg global.Config) *cobra.Command {
	var limit int32
	var includeApprovedNotices bool
	cmd := &cobra.Command{
		Use:   "candidates",
		Short: "LLM 보정 후보 artifact를 조회합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := persistence.Open(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer repo.Close()
			candidates, err := repo.ListLLMRepairCandidates(ctx, limit, includeApprovedNotices)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), formatLLMRepairCandidates(candidates))
			return nil
		},
	}
	cmd.Flags().Int32Var(&limit, "limit", 20, "조회할 LLM 보정 후보 수")
	cmd.Flags().BoolVar(&includeApprovedNotices, "include-approved-notices", false, "이미 QA-approved Offering이 있는 공고의 artifact도 후보에 포함합니다")
	return cmd
}

func formatLLMRepairCandidates(candidates []persistence.LLMRepairArtifact) string {
	var b strings.Builder
	fmt.Fprintf(&b, "llm_candidates=%d\n", len(candidates))
	for _, candidate := range candidates {
		fmt.Fprintf(
			&b,
			"artifact_id=%d seq=%s type=%s raw_chars=%d file=%q source=%s\n",
			candidate.ID,
			candidate.NoticeSeq,
			candidate.ArtifactType,
			len([]rune(candidate.RawText)),
			candidate.OriginalFilename,
			candidate.SourceSpan,
		)
	}
	return b.String()
}

func newLLMRepairCommand(ctx context.Context, cfg global.Config) *cobra.Command {
	var artifactID int64
	var dryRun bool
	var maxInputChars int
	var maxAttempts int
	var model string

	cmd := &cobra.Command{
		Use:   "repair",
		Short: "artifact를 GPT 기반 LLM 보조로 보정합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			if artifactID <= 0 {
				return fmt.Errorf("--artifact-id is required")
			}
			if maxAttempts <= 0 || maxAttempts > 1500 {
				return fmt.Errorf("--max-attempts must be between 1 and 1500")
			}
			repo, err := persistence.Open(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer repo.Close()

			artifact, err := repo.GetLLMRepairArtifact(ctx, artifactID)
			if err != nil {
				return err
			}
			input := llm.RepairInput{
				ArtifactID:    artifact.ID,
				ArtifactType:  artifact.ArtifactType,
				NoticeSeq:     artifact.NoticeSeq,
				NoticeTitle:   artifact.NoticeTitle,
				OriginalFile:  artifact.OriginalFilename,
				SourceSpan:    artifact.SourceSpan,
				RawText:       artifact.RawText,
				ContentJSON:   json.RawMessage(artifact.ContentJSON),
				Confidence:    artifact.Confidence,
				MaxInputChars: maxInputChars,
			}

			if dryRun {
				request, inputHash, err := llm.BuildRepairRequest(input, model)
				if err != nil {
					return err
				}
				requestJSON, err := json.MarshalIndent(request, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "llm repair artifact_id=%d model=%s input_hash=%s max_attempts=%d dry_run=true\n", artifactID, firstNonEmptyString(model, cfg.OpenAIModel, llm.DefaultModel), inputHash, maxAttempts)
				fmt.Fprintln(cmd.OutOrStdout(), string(requestJSON))
				return nil
			}

			attemptCount, err := repo.CountLLMRepairAttempts(ctx)
			if err != nil {
				return err
			}
			if err := validateLLMRepairAttemptLimit(attemptCount, maxAttempts); err != nil {
				return err
			}

			client := llm.Client{
				APIKey:   cfg.OpenAIAPIKey,
				Model:    model,
				Endpoint: cfg.OpenAIBaseURL,
			}
			output, attempt, repairErr := client.RepairOfferings(ctx, input)
			attemptID, insertErr := repo.InsertLLMRepairAttempt(ctx, attempt)
			if insertErr != nil {
				return insertErr
			}
			if repairErr != nil {
				return fmt.Errorf("llm repair failed attempt_id=%d: %w", attemptID, repairErr)
			}
			upsertedOfferings, err := persistLLMRepairOfferings(ctx, repo, artifact, output)
			if err != nil {
				return fmt.Errorf("llm repair succeeded attempt_id=%d but offering persistence failed: %w", attemptID, err)
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"llm repair attempt_id=%d artifact_id=%d status=%s offerings=%d upserted_offerings=%d confidence=%.2f input_hash=%s output_hash=%s\n",
				attemptID,
				artifactID,
				attempt.Status,
				len(output.Offerings),
				upsertedOfferings,
				output.Confidence,
				attempt.InputHash,
				attempt.OutputHash,
			)
			return nil
		},
	}
	cmd.Flags().Int64Var(&artifactID, "artifact-id", 0, "보정할 extracted_artifacts.id")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "OpenAI API 호출 없이 요청 JSON만 출력합니다")
	cmd.Flags().StringVar(&model, "model", cfg.OpenAIModel, "사용할 OpenAI 모델")
	cmd.Flags().IntVar(&maxInputChars, "max-input-chars", 12000, "LLM 입력 raw_text 최대 문자 수")
	cmd.Flags().IntVar(&maxAttempts, "max-attempts", cfg.LLMMaxAttempts, "LLM 정제 최대 시도 횟수(최대 1500)")
	return cmd
}

func validateLLMRepairAttemptLimit(existingAttempts int64, maxAttempts int) error {
	if maxAttempts <= 0 || maxAttempts > 1500 {
		return fmt.Errorf("--max-attempts must be between 1 and 1500")
	}
	if existingAttempts >= int64(maxAttempts) {
		return fmt.Errorf("maximum LLM repair attempts reached: existing=%d max=%d", existingAttempts, maxAttempts)
	}
	return nil
}

type llmRepairOfferingStore interface {
	UpsertOffering(ctx context.Context, attachment persistence.PersistedAttachment, sourceArtifactID int64, offering normalize.OfferingCandidate) (int64, error)
}

type llmRepairOfferingCleaner interface {
	DeleteLLMRepairOfferings(ctx context.Context, sourceArtifactID int64) error
}

func persistLLMRepairOfferings(ctx context.Context, store llmRepairOfferingStore, artifact persistence.LLMRepairArtifact, output llm.RepairOutput) (int, error) {
	attachment, err := artifact.LLMRepairAttachmentRef()
	if err != nil {
		return 0, err
	}
	if cleaner, ok := store.(llmRepairOfferingCleaner); ok {
		if err := cleaner.DeleteLLMRepairOfferings(ctx, artifact.ID); err != nil {
			return 0, err
		}
	}
	offerings := persistence.PrepareLLMRepairOfferings(output)
	upserted := 0
	for _, offering := range offerings {
		if _, err := store.UpsertOffering(ctx, attachment, artifact.ID, offering); err != nil {
			return upserted, err
		}
		upserted++
	}
	return upserted, nil
}

func newWorkflowCommand(ctx context.Context, cfg global.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "discovery, extraction, normalization, QA를 오케스트레이션합니다",
	}
	cmd.AddCommand(
		newWorkflowCollectSHCommand(),
		newWorkflowCollectLHCommand(ctx, cfg),
	)
	return cmd
}

func newWorkflowCollectLHCommand(ctx context.Context, cfg global.Config) *cobra.Command {
	var kind string
	var pages int
	var numRows int
	var dryRun bool
	var serviceKey string
	var allPages bool
	var agencyFilter string
	var showItems bool
	var fetchDetails bool

	cmd := &cobra.Command{
		Use:   "collect-lh",
		Short: "LH/MyHome OpenAPI 수집 워크플로우를 실행합니다",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			endpoint, err := myHomeEndpointFromKind(kind)
			if err != nil {
				return err
			}
			key := firstNonEmptyString(serviceKey, cfg.MyHomeAPIKey)
			if strings.TrimSpace(key) == "" {
				return fmt.Errorf("MYHOME_API_KEY or --service-key is required")
			}
			client := lhdiscovery.MyHomeClient{ServiceKey: key}
			var items []lhdiscovery.MyHomeNoticeItem
			totalCount := 0
			pagesToFetch := pages
			for page := 1; page <= pagesToFetch; page++ {
				result, err := client.ListNotices(ctx, endpoint, page, numRows)
				if err != nil {
					return err
				}
				if result.TotalCount > totalCount {
					totalCount = result.TotalCount
				}
				if page == 1 {
					pagesToFetch = lhdiscovery.MyHomePagesToFetch(totalCount, numRows, allPages, pages)
				}
				items = append(items, result.Items...)
			}
			rawItems := len(items)
			items = lhdiscovery.FilterMyHomeItemsByAgency(items, agencyFilter)
			fmt.Fprintf(cmd.OutOrStdout(), "agency=LH source=myhome kind=%s endpoint=%s pages=%d total_count=%d items=%d raw_items=%d agency_filter=%q dry_run=%t\n", kind, endpoint, pagesToFetch, totalCount, len(items), rawItems, agencyFilter, dryRun)
			if dryRun {
				if showItems {
					for _, item := range items {
						fmt.Fprintf(cmd.OutOrStdout(), "candidate seq=%s agency=%s title=%q supply_count=%s\n", item.SourceSeq(), item.Agency, item.Title, intPtrString(item.SupplyCount))
					}
				}
				return nil
			}

			if err := persistence.Migrate(ctx, cfg.DatabaseURL); err != nil {
				return err
			}
			repo, err := persistence.Open(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer repo.Close()

			runID, err := repo.CreateCollectionRun(ctx, "myhome:"+kind)
			if err != nil {
				return err
			}
			upsertedArtifacts := 0
			upsertedDetailArtifacts := 0
			upsertedOfferings := 0
			upsertedSchedules := 0
			defer func() {
				status := persistence.CollectionRunStatusSucceeded
				errorText := ""
				if err != nil {
					status = persistence.CollectionRunStatusFailed
					errorText = err.Error()
				}
				stats := map[string]any{
					"kind":                      kind,
					"endpoint":                  string(endpoint),
					"pages":                     pages,
					"pages_fetched":             pagesToFetch,
					"num_rows":                  numRows,
					"total_count":               totalCount,
					"items":                     len(items),
					"raw_items":                 rawItems,
					"agency_filter":             agencyFilter,
					"upserted_artifacts":        upsertedArtifacts,
					"upserted_detail_artifacts": upsertedDetailArtifacts,
					"upserted_offerings":        upsertedOfferings,
					"upserted_schedules":        upsertedSchedules,
				}
				finishErr := repo.FinishCollectionRun(ctx, runID, status, stats, errorText)
				if err == nil && finishErr != nil {
					err = finishErr
				}
			}()

			for _, item := range items {
				noticeID, err := repo.SaveMyHomeNotice(ctx, endpoint, item)
				if err != nil {
					return err
				}
				artifactID, sourceSpan, err := repo.InsertMyHomeArtifact(ctx, endpoint, item)
				if err != nil {
					return err
				}
				upsertedArtifacts++
				offeringArtifactID := artifactID
				if fetchDetails && shouldFetchMyHomeDetail(item) {
					detail, detailErr := client.GetNoticeDetail(ctx, item.DetailURL)
					if detailErr == nil {
						item = item.WithDetail(detail)
						detailArtifactID, _, err := repo.InsertMyHomeDetailArtifact(ctx, endpoint, item, detail)
						if err != nil {
							return err
						}
						offeringArtifactID = detailArtifactID
						upsertedDetailArtifacts++
					}
				}
				offering := normalize.OfferingFromMyHomeItem(item, sourceSpan)
				if _, err := repo.UpsertMyHomeOffering(ctx, noticeID, offeringArtifactID, item.Agency, offering); err != nil {
					return err
				}
				upsertedOfferings++
				if schedule, ok := normalize.ApplicationScheduleFromMyHomeItem(item, noticeID, sourceSpan); ok {
					schedule.SourceArtifactID = artifactID
					if _, err := repo.UpsertNoticeSchedule(ctx, schedule); err != nil {
						return err
					}
					upsertedSchedules++
				}
			}
			storedObjects, totalArtifacts, err := repo.Counts(ctx)
			if err != nil {
				return err
			}
			totalOfferings, err := repo.CountOfferings(ctx)
			if err != nil {
				return err
			}
			qaSummary, err := repo.PromoteOfferingsQA(ctx)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "db stored_objects=%d extracted_artifacts=%d offerings=%d upserted_artifacts=%d upserted_offerings=%d upserted_schedules=%d qa_approved=%d qa_rejected=%d qa_pending=%d\n",
				storedObjects,
				totalArtifacts,
				totalOfferings,
				upsertedArtifacts+upsertedDetailArtifacts,
				upsertedOfferings,
				upsertedSchedules,
				qaSummary.Approved,
				qaSummary.Rejected,
				qaSummary.Pending,
			)
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "rental", "MyHome 모집공고 종류: rental 또는 sale")
	cmd.Flags().IntVar(&pages, "pages", 1, "조회할 API 페이지 수")
	cmd.Flags().IntVar(&numRows, "num-rows", 200, "페이지당 요청 건수")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "DB 저장 없이 MyHome 후보만 보고합니다")
	cmd.Flags().StringVar(&serviceKey, "service-key", cfg.MyHomeAPIKey, "MyHome OpenAPI serviceKey")
	cmd.Flags().BoolVar(&allPages, "all-pages", false, "첫 페이지 totalCount 기준으로 모든 MyHome 페이지를 조회합니다")
	cmd.Flags().StringVar(&agencyFilter, "agency-filter", "", "특정 공급기관명만 저장/출력합니다. 예: LH")
	cmd.Flags().BoolVar(&showItems, "show-items", false, "dry-run에서 MyHome 후보 목록을 상세 출력합니다")
	cmd.Flags().BoolVar(&fetchDetails, "fetch-details", true, "금액/공급호수 보강이 필요한 MyHome 상세 HTML을 조회합니다")
	return cmd
}

func myHomeEndpointFromKind(kind string) (lhdiscovery.MyHomeEndpoint, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", "rental", "rent", "rsdt":
		return lhdiscovery.MyHomeRental, nil
	case "sale", "lt", "분양":
		return lhdiscovery.MyHomeSale, nil
	default:
		return "", fmt.Errorf("unknown MyHome kind: %s", kind)
	}
}

func shouldFetchMyHomeDetail(item lhdiscovery.MyHomeNoticeItem) bool {
	if strings.TrimSpace(item.DetailURL) == "" {
		return false
	}
	if item.SupplyCount == nil || *item.SupplyCount <= 0 {
		return true
	}
	if strings.TrimSpace(item.SupplyType) == "" {
		return false
	}
	if item.DepositKRW != nil && item.MonthlyRent != nil {
		return false
	}
	switch strings.TrimSpace(item.SupplyType) {
	case "전세임대", "매입임대":
		return true
	default:
		return false
	}
}

func newWorkflowCollectSHCommand() *cobra.Command {
	var boardKind string
	var pages int
	var dryRun bool
	var preserveAttachments bool
	var objectRoot string
	var seqs string
	var maxAgeDays int
	var skipExisting bool
	var activeApplications bool
	var activeSplyTy string
	var activeMaxPages int
	var discoveryCache bool

	cmd := &cobra.Command{
		Use:   "collect-sh",
		Short: "SH 수집 워크플로우를 실행합니다",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			registry := discovery.NewStaticSiteRegistry()
			board, ok := registry.Get("SH", boardKind)
			if !ok {
				return fmt.Errorf("unknown SH board: %s", boardKind)
			}
			if err := persistence.Migrate(cmd.Context(), global.FromEnv().DatabaseURL); err != nil {
				return err
			}
			repo, err := persistence.Open(cmd.Context(), global.FromEnv().DatabaseURL)
			if err != nil {
				return err
			}
			defer repo.Close()

			var report discovery.Report
			var runID int64
			var downloaded int
			var upsertedArtifacts int
			var upsertedOfferings int
			var upsertedSchedules int
			var upsertedApplications int
			var linkedApplications int
			var upsertedDiscoverySeen int
			var storedObjects int64
			var totalArtifacts int64
			var totalOfferings int64
			if !dryRun && preserveAttachments {
				runID, err = repo.CreateCollectionRun(cmd.Context(), strings.ToLower(board.Agency)+":"+board.BoardKind)
				if err != nil {
					return err
				}
				defer func() {
					status := persistence.CollectionRunStatusSucceeded
					errorText := ""
					if err != nil {
						status = persistence.CollectionRunStatusFailed
						errorText = err.Error()
					}
					stats := collectionRunStats(report, downloaded, upsertedArtifacts, upsertedOfferings, storedObjects, totalArtifacts, totalOfferings)
					stats["upserted_schedules"] = upsertedSchedules
					stats["upserted_application_notices"] = upsertedApplications
					stats["linked_application_notices"] = linkedApplications
					stats["upserted_discovery_seen_cache"] = upsertedDiscoverySeen
					finishErr := repo.FinishCollectionRun(cmd.Context(), runID, status, stats, errorText)
					if err == nil && finishErr != nil {
						err = finishErr
					}
				}()
			}

			var knownSeqs map[string]bool
			if skipExisting && strings.TrimSpace(seqs) == "" {
				knownSeqs, err = repo.ExistingNoticeSeqs(cmd.Context(), board.Agency, board.BoardKind)
				if err != nil {
					return err
				}
			}
			var existingCandidates []discovery.Candidate
			existingNoticeIDBySeq := make(map[string]int64)
			if activeApplications && skipExisting && strings.TrimSpace(seqs) == "" {
				existingCandidates, existingNoticeIDBySeq, err = repo.ExistingNoticeCandidates(cmd.Context(), board.Agency, board.BoardKind)
				if err != nil {
					return err
				}
			}
			var seenCache map[string]discovery.SeenCacheEntry
			if discoveryCache && strings.TrimSpace(seqs) == "" {
				seenCache, err = repo.FreshDiscoverySeenCache(cmd.Context(), board.Agency, board.BoardKind, time.Now())
				if err != nil {
					return err
				}
			}
			var applications []shdiscovery.ApplicationNotice
			if activeApplications {
				applications, err = collectSHApplicationNotices(cmd.Context(), splitCSV(activeSplyTy))
				if err != nil {
					return err
				}
				if activeMaxPages > pages {
					pages = activeMaxPages
				}
			}
			report, err = discovery.NewDiscoverer(discovery.NewHTTPFetcher()).Discover(cmd.Context(), board, discovery.Options{
				Pages:        pages,
				Seqs:         splitCSV(seqs),
				CutoffDate:   cutoffDate(maxAgeDays),
				KnownSeqs:    knownSeqs,
				TargetTitles: shdiscovery.ActiveTargetTitles(applications),
				SeenCache:    seenCache,
			})
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), report.String())
			reconcileCandidates := append([]discovery.Candidate{}, existingCandidates...)
			reconcileCandidates = append(reconcileCandidates, report.Candidates...)
			reconcileResult := shdiscovery.ReconcileApplications(applications, reconcileCandidates)
			if activeApplications {
				fmt.Fprintf(cmd.OutOrStdout(), "active_applications=%d linked=%d unmatched=%d\n", len(applications), len(reconcileResult.Linked), len(reconcileResult.UnmatchedApplications))
				writeSHApplicationLinks(cmd.OutOrStdout(), reconcileResult)
			}
			if discoveryCache && !dryRun {
				upsertedDiscoverySeen, err = upsertDiscoverySeenCache(cmd.Context(), repo, report, time.Now())
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "discovery_cache skipped_seen=%d upserted=%d\n", report.SkippedSeen, upsertedDiscoverySeen)
			}
			if dryRun || !preserveAttachments {
				return nil
			}

			objectStore := extraction.NewLocalObjectStore(objectRoot)
			collector := workflow.NewCollector(workflow.NewSHAttachmentFetcher(), objectStore)
			noticeIDBySeq := make(map[string]int64, len(report.Candidates))
			for seq, noticeID := range existingNoticeIDBySeq {
				noticeIDBySeq[seq] = noticeID
			}
			for _, candidate := range report.Candidates {
				preserveReport, err := collector.PreserveCandidateAttachments(cmd.Context(), board, candidate)
				if err != nil {
					return err
				}
				downloaded += preserveReport.Downloaded
				persisted, err := repo.SaveCandidatePreservation(cmd.Context(), board, candidate, preserveReport)
				if err != nil {
					return err
				}
				if len(persisted) > 0 {
					noticeIDBySeq[candidate.Seq] = persisted[0].NoticeID
				}
				fmt.Fprintln(cmd.OutOrStdout(), preserveReport.String())
				for _, object := range preserveReport.Objects {
					fmt.Fprintf(cmd.OutOrStdout(), "object seq=%s file_seq=%s kind=%s key=%s sha256=%s size=%d\n", object.Seq, object.FileSeq, object.Kind, object.StoredObject.Key, object.StoredObject.SHA256, object.StoredObject.SizeBytes)
				}
				for _, attachment := range persisted {
					artifacts, err := extractPreservedAttachment(objectStore, attachment)
					if err != nil {
						return err
					}
					previewArtifacts, err := workflow.ExtractPreservedPreview(objectStore, attachment.PreviewObjectKey)
					if err != nil {
						return err
					}
					artifacts = append(artifacts, previewArtifacts...)
					artifactIDsBySpan := make(map[string]int64, len(artifacts))
					for _, artifact := range artifacts {
						storedObjectID := attachment.StoredObjectID
						if artifact.Type == extraction.ArtifactTypeHTMLPreview && attachment.PreviewStoredObjectID > 0 {
							storedObjectID = attachment.PreviewStoredObjectID
						}
						artifactID, err := repo.InsertArtifact(cmd.Context(), attachment.AttachmentID, storedObjectID, artifact)
						if err != nil {
							return err
						}
						artifactIDsBySpan[artifact.SourceSpan] = artifactID
						upsertedArtifacts++
					}
					for _, offering := range normalizeOfferingsFromArtifacts(attachment.Kind, artifacts) {
						artifactID := artifactIDsBySpan[offering.SourceSpan]
						if _, err := repo.UpsertOffering(cmd.Context(), attachment, artifactID, offering); err != nil {
							return err
						}
						upsertedOfferings++
					}
					for _, schedule := range normalize.InferSchedulesFromTextArtifacts(artifacts, attachment.NoticeID) {
						schedule.SourceArtifactID = sourceArtifactIDForSchedule(schedule.SourceSpan, artifactIDsBySpan)
						if schedule.SourceArtifactID == 0 {
							continue
						}
						if _, err := repo.UpsertNoticeSchedule(cmd.Context(), schedule); err != nil {
							return err
						}
						upsertedSchedules++
					}
				}
			}
			applicationIDByCode := make(map[string]int64, len(applications))
			for _, application := range applications {
				applicationID, err := repo.UpsertApplicationNotice(cmd.Context(), applicationNoticeInput(application))
				if err != nil {
					return err
				}
				applicationIDByCode[application.RecruitNoticeCode] = applicationID
				upsertedApplications++
			}
			for _, link := range reconcileResult.Linked {
				noticeID := noticeIDBySeq[link.BoardSeq]
				applicationID := applicationIDByCode[link.RecruitNoticeCode]
				if noticeID == 0 || applicationID == 0 {
					continue
				}
				if err := repo.LinkApplicationNoticeToSourceNotice(cmd.Context(), applicationID, noticeID); err != nil {
					return err
				}
				linkedApplications++
			}
			storedObjects, totalArtifacts, err = repo.Counts(cmd.Context())
			if err != nil {
				return err
			}
			totalOfferings, err = repo.CountOfferings(cmd.Context())
			if err != nil {
				return err
			}
			qaSummary, err := repo.PromoteOfferingsQA(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), formatCollectionSummary(downloaded, upsertedArtifacts, upsertedOfferings, storedObjects, totalArtifacts, totalOfferings, qaSummary))
			fmt.Fprintf(cmd.OutOrStdout(), "schedules upserted_schedules=%d\n", upsertedSchedules)
			if activeApplications {
				fmt.Fprintf(cmd.OutOrStdout(), "application_notices upserted=%d linked=%d\n", upsertedApplications, linkedApplications)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&boardKind, "board", "rental", "SH 게시판 종류: rental 또는 sale")
	cmd.Flags().IntVar(&pages, "pages", 1, "조회할 게시판 페이지 수")
	cmd.Flags().StringVar(&seqs, "seq", "", "쉼표로 구분한 원본 공고 seq 필터")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "첨부 보존 없이 후보만 발견합니다")
	cmd.Flags().BoolVar(&preserveAttachments, "preserve-attachments", false, "후보 첨부를 다운로드하고 보존합니다")
	cmd.Flags().StringVar(&objectRoot, "object-root", ".data/objects", "로컬 ObjectStore 루트 디렉터리")
	cmd.Flags().IntVar(&maxAgeDays, "max-age-days", 30, "지정 일수보다 오래된 공고는 제외합니다. 0이면 비활성화합니다")
	cmd.Flags().BoolVar(&skipExisting, "skip-existing", true, "--seq가 없을 때 이미 수집한 모집공고를 건너뜁니다")
	cmd.Flags().BoolVar(&activeApplications, "active-applications", false, "SH 인터넷청약 청약중/접수예정 공고를 먼저 수집해 게시판 탐색 목표로 사용합니다")
	cmd.Flags().StringVar(&activeSplyTy, "active-sply-ty", "", "쉼표로 구분한 SH 인터넷청약 공급유형 코드. 비우면 기본 전체를 조회합니다")
	cmd.Flags().IntVar(&activeMaxPages, "active-max-pages", 10, "인터넷청약 활성 공고 reconcile을 위해 탐색할 최대 게시판 페이지 수")
	cmd.Flags().BoolVar(&discoveryCache, "discovery-cache", true, "비모집/거절 공고 상세 재조회 방지를 위한 discovery seen cache를 사용합니다")
	return cmd
}

const (
	discoverySeenCachePolicyVersion = "sh-discovery-cache-v1"
	discoverySeenCacheParserVersion = "sh-board-parser-v1"
	discoverySeenCacheRejectedTTL   = 60 * 24 * time.Hour
	discoverySeenCacheUnknownTTL    = 7 * 24 * time.Hour
)

func upsertDiscoverySeenCache(ctx context.Context, repo *persistence.Repository, report discovery.Report, now time.Time) (int, error) {
	upserted := 0
	for _, rejected := range report.Rejected {
		if strings.TrimSpace(rejected.Seq) == "" {
			continue
		}
		if err := repo.UpsertDiscoverySeenCache(ctx, discoverySeenCacheInputFromRejectedPost(rejected, now)); err != nil {
			return upserted, err
		}
		upserted++
	}
	return upserted, nil
}

func discoverySeenCacheInputFromRejectedPost(rejected discovery.RejectedPost, now time.Time) persistence.DiscoverySeenCacheInput {
	status := discovery.SeenCacheStatusRejected
	ttl := discoverySeenCacheRejectedTTL
	if rejected.Reason == discovery.NoticeCategoryUnknown {
		status = discovery.SeenCacheStatusRejectedUnknown
		ttl = discoverySeenCacheUnknownTTL
	}
	return persistence.DiscoverySeenCacheInput{
		Agency:        rejected.Agency,
		BoardKind:     rejected.BoardKind,
		Seq:           rejected.Seq,
		Status:        status,
		Reason:        rejected.Reason,
		Title:         rejected.Title,
		PostedAt:      rejected.PostedAt,
		ExpiresAt:     now.Add(ttl),
		PolicyVersion: discoverySeenCachePolicyVersion,
		ParserVersion: discoverySeenCacheParserVersion,
		Evidence: map[string]any{
			"reason": string(rejected.Reason),
			"title":  rejected.Title,
		},
	}
}

func formatCollectionSummary(downloaded int, upsertedArtifacts int, upsertedOfferings int, storedObjects int64, totalArtifacts int64, totalOfferings int64, qaSummary persistence.QASummary) string {
	return fmt.Sprintf(
		"db stored_objects=%d extracted_artifacts=%d offerings=%d upserted_artifacts=%d upserted_offerings=%d qa_approved=%d qa_rejected=%d qa_pending=%d\n",
		storedObjects,
		totalArtifacts,
		totalOfferings,
		upsertedArtifacts,
		upsertedOfferings,
		qaSummary.Approved,
		qaSummary.Rejected,
		qaSummary.Pending,
	)
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func normalizeOfferingsFromArtifacts(kind extraction.AttachmentKind, artifacts []extraction.ExtractedArtifact) []normalize.OfferingCandidate {
	switch kind {
	case extraction.AttachmentKindOfferingListXLSX:
		return normalize.InferOfferingsFromXLSXRows(artifacts)
	case extraction.AttachmentKindNoticePDF:
		offerings := make([]normalize.OfferingCandidate, 0)
		for _, artifact := range artifacts {
			offerings = append(offerings, normalize.InferOfferingsFromPDFText(artifact)...)
		}
		offerings = append(offerings, normalize.InferOfferingsFromPDFTableRows(artifacts)...)
		return offerings
	case extraction.AttachmentKindNoticeHWP:
		return normalize.InferOfferingsFromPDFTableRows(artifacts)
	default:
		return nil
	}
}

func sourceArtifactIDForSchedule(scheduleSourceSpan string, artifactIDsBySpan map[string]int64) int64 {
	base, _, _ := strings.Cut(scheduleSourceSpan, "#schedule=")
	return artifactIDsBySpan[base]
}

func extractPreservedAttachment(objectStore extraction.LocalObjectStore, attachment persistence.PersistedAttachment) ([]extraction.ExtractedArtifact, error) {
	return workflow.ExtractPreservedAttachment(objectStore, workflow.PreservedAttachmentRef{
		ObjectKey: attachment.ObjectKey,
		Filename:  attachment.Filename,
		Kind:      attachment.Kind,
	})
}

func newQACommand(ctx context.Context, cfg global.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qa",
		Short: "품질 게이트와 샘플 회귀 검사를 실행합니다",
	}
	cmd.AddCommand(
		placeholderCommand("sample", "샘플 QA 케이스를 실행합니다"),
		newQAPDFOfferingsCommand(),
		newQAPromoteOfferingsCommand(ctx, cfg),
	)
	return cmd
}

func newQAPDFOfferingsCommand() *cobra.Command {
	var file string
	var debugText bool
	cmd := &cobra.Command{
		Use:   "pdf-offerings",
		Short: "PDF 파일에서 공급항목 후보를 추출해 검증용 표로 출력합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(file) == "" {
				return fmt.Errorf("--file is required")
			}
			artifacts, err := extraction.ExtractPDFArtifacts(file)
			if err != nil {
				return err
			}
			if debugText {
				fmt.Fprint(cmd.OutOrStdout(), formatPDFDebugText(artifacts))
			}
			offerings := normalizeOfferingsFromArtifacts(extraction.AttachmentKindNoticePDF, artifacts)
			fmt.Fprintf(cmd.OutOrStdout(), "file=%s artifacts=%d ", filepath.Clean(file), len(artifacts))
			fmt.Fprint(cmd.OutOrStdout(), formatPDFOfferings(offerings))
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "PDF 파일 경로")
	cmd.Flags().BoolVar(&debugText, "debug-text", false, "PDF 추출 원문에서 공급 관련 스니펫을 함께 출력합니다")
	return cmd
}

func newQAPromoteOfferingsCommand(ctx context.Context, cfg global.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "promote-offerings",
		Short: "QA를 통과한 pending 공급항목을 승인합니다",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := persistence.Open(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer repo.Close()

			summary, err := repo.PromoteOfferingsQA(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), formatQASummary(summary))
			return nil
		},
	}
}

func formatQASummary(summary persistence.QASummary) string {
	return fmt.Sprintf(
		"qa approved=%d rejected=%d pending=%d\n",
		summary.Approved,
		summary.Rejected,
		summary.Pending,
	)
}

func formatPDFOfferings(offerings []normalize.OfferingCandidate) string {
	var b strings.Builder
	fmt.Fprintf(&b, "offerings=%d\n", len(offerings))
	if len(offerings) == 0 {
		b.WriteString("no offerings extracted\n")
		return b.String()
	}
	b.WriteString("| # | 신청 가능 단위 | 주택명 | 호실 | 면적(㎡) | 공급호수 | 전세금액 | 보증금 | 월임대료 | 기숙사비 | 성별 | source | confidence |\n")
	b.WriteString("|---|---|---|---|---:|---:|---:|---:|---:|---:|---|---|---:|\n")
	for i, offering := range offerings {
		fmt.Fprintf(
			&b,
			"| %d | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %.2f |\n",
			i+1,
			tableCell(offering.ApplicationUnitLabel),
			tableCell(firstNonEmptyString(offering.HousingName, offering.ComplexName)),
			tableCell(offering.UnitNo),
			floatPtrString(offering.ExclusiveAreaM2),
			intPtrString(offering.SupplyCount),
			int64PtrString(offering.JeonseDepositKRW),
			int64PtrString(offering.DepositKRW),
			int64PtrString(offering.MonthlyRentKRW),
			int64PtrString(offering.DormitoryFeeKRW),
			tableCell(offering.GenderRequirement),
			tableCell(offering.SourceSpan),
			offering.Confidence,
		)
	}
	return b.String()
}

func formatPDFDebugText(artifacts []extraction.ExtractedArtifact) string {
	for _, artifact := range artifacts {
		if artifact.Type != extraction.ArtifactTypePDFText {
			continue
		}
		text := strings.TrimSpace(strings.Join(strings.Fields(strings.ReplaceAll(artifact.RawText, "\x00", "")), " "))
		for _, keyword := range []string{"공급대상", "공급 대상", "공급현황", "주택명"} {
			index := strings.Index(text, keyword)
			if index < 0 {
				continue
			}
			start := index - 400
			if start < 0 {
				start = 0
			}
			end := index + 1600
			if end > len(text) {
				end = len(text)
			}
			return fmt.Sprintf("debug_text keyword=%q\n%s\n", keyword, text[start:end])
		}
		return fmt.Sprintf("debug_text chars=%d keyword_not_found\n", len([]rune(text)))
	}
	return "debug_text pdf_text_artifact_not_found\n"
}

func tableCell(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.TrimSpace(value)
}

func floatPtrString(value *float64) string {
	if value == nil {
		return ""
	}
	if *value == float64(int64(*value)) {
		return fmt.Sprintf("%.0f", *value)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", *value), "0"), ".")
}

func intPtrString(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func int64PtrString(value *int64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func placeholderCommand(use string, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
}
