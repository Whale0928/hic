package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"hic/pkg/api"
	"hic/pkg/discovery"
	"hic/pkg/extraction"
	"hic/pkg/global"
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
		newLLMCommand(),
		newWorkflowCommand(),
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

func collectionRunStats(report discovery.Report, downloaded int, insertedArtifacts int, upsertedOfferings int, storedObjects int64, totalArtifacts int64, totalOfferings int64) map[string]any {
	return map[string]any{
		"pages":              report.Pages,
		"list_rows":          report.ListRows,
		"details":            report.Details,
		"candidates":         len(report.Candidates),
		"rejected":           len(report.Rejected),
		"skipped_old":        report.SkippedOld,
		"skipped_known":      report.SkippedKnown,
		"stopped_by_cutoff":  report.StoppedByCutoff,
		"downloaded":         downloaded,
		"inserted_artifacts": insertedArtifacts,
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
		newExtractPDFCommand(),
		newExtractXLSXCommand(),
	)
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

func newLLMCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "llm",
		Short: "낮은 신뢰도의 artifact를 LLM 보조로 보정합니다",
	}
	cmd.AddCommand(placeholderCommand("repair", "artifact LLM 보정을 실행합니다"))
	return cmd
}

func newWorkflowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "discovery, extraction, normalization, QA를 오케스트레이션합니다",
	}
	cmd.AddCommand(newWorkflowCollectSHCommand())
	return cmd
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
			var insertedArtifacts int
			var upsertedOfferings int
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
					finishErr := repo.FinishCollectionRun(cmd.Context(), runID, status, collectionRunStats(report, downloaded, insertedArtifacts, upsertedOfferings, storedObjects, totalArtifacts, totalOfferings), errorText)
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
			report, err = discovery.NewDiscoverer(discovery.NewHTTPFetcher()).Discover(cmd.Context(), board, discovery.Options{
				Pages:      pages,
				Seqs:       splitCSV(seqs),
				CutoffDate: cutoffDate(maxAgeDays),
				KnownSeqs:  knownSeqs,
			})
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), report.String())
			if dryRun || !preserveAttachments {
				return nil
			}

			objectStore := extraction.NewLocalObjectStore(objectRoot)
			collector := workflow.NewCollector(workflow.NewSHAttachmentFetcher(), objectStore)
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
				fmt.Fprintln(cmd.OutOrStdout(), preserveReport.String())
				for _, object := range preserveReport.Objects {
					fmt.Fprintf(cmd.OutOrStdout(), "object seq=%s file_seq=%s kind=%s key=%s sha256=%s size=%d\n", object.Seq, object.FileSeq, object.Kind, object.StoredObject.Key, object.StoredObject.SHA256, object.StoredObject.SizeBytes)
				}
				for _, attachment := range persisted {
					artifacts, err := extractPreservedAttachment(objectStore, attachment)
					if err != nil {
						return err
					}
					artifactIDsBySpan := make(map[string]int64, len(artifacts))
					for _, artifact := range artifacts {
						artifactID, err := repo.InsertArtifact(cmd.Context(), attachment.AttachmentID, attachment.StoredObjectID, artifact)
						if err != nil {
							return err
						}
						artifactIDsBySpan[artifact.SourceSpan] = artifactID
						insertedArtifacts++
					}
					for _, offering := range normalizeOfferingsFromArtifacts(attachment.Kind, artifacts) {
						artifactID := artifactIDsBySpan[offering.SourceSpan]
						if _, err := repo.UpsertOffering(cmd.Context(), attachment, artifactID, offering); err != nil {
							return err
						}
						upsertedOfferings++
					}
				}
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
			fmt.Fprint(cmd.OutOrStdout(), formatCollectionSummary(downloaded, insertedArtifacts, upsertedOfferings, storedObjects, totalArtifacts, totalOfferings, qaSummary))
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
	return cmd
}

func formatCollectionSummary(downloaded int, insertedArtifacts int, upsertedOfferings int, storedObjects int64, totalArtifacts int64, totalOfferings int64, qaSummary persistence.QASummary) string {
	return fmt.Sprintf(
		"db stored_objects=%d extracted_artifacts=%d offerings=%d inserted_artifacts=%d upserted_offerings=%d qa_approved=%d qa_rejected=%d qa_pending=%d\n",
		storedObjects,
		totalArtifacts,
		totalOfferings,
		insertedArtifacts,
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
	default:
		return nil
	}
}

func extractPreservedAttachment(objectStore extraction.LocalObjectStore, attachment persistence.PersistedAttachment) ([]extraction.ExtractedArtifact, error) {
	path, err := objectStore.PathForKey(attachment.ObjectKey)
	if err != nil {
		return nil, err
	}
	switch attachment.Kind {
	case extraction.AttachmentKindNoticePDF, extraction.AttachmentKindSchedulePDF:
		artifacts, err := extraction.ExtractPDFArtifacts(path)
		if err != nil {
			return nil, err
		}
		return artifacts, nil
	case extraction.AttachmentKindOfferingListXLSX:
		return extraction.ExtractXLSXRows(path)
	default:
		return nil, nil
	}
}

func newQACommand(ctx context.Context, cfg global.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qa",
		Short: "품질 게이트와 샘플 회귀 검사를 실행합니다",
	}
	cmd.AddCommand(
		placeholderCommand("sample", "샘플 QA 케이스를 실행합니다"),
		newQAPromoteOfferingsCommand(ctx, cfg),
	)
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

func placeholderCommand(use string, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
}
