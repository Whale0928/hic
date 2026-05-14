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
		Short:        "House Information Collector",
		SilenceUsage: true,
	}

	root.AddCommand(
		newMigrateCommand(ctx, cfg),
		newServeCommand(ctx, cfg),
		newDiscoveryCommand(ctx, cfg),
		newExtractCommand(),
		newNormalizeCommand(),
		newLLMCommand(),
		newWorkflowCommand(),
		newQACommand(),
	)

	return root
}

func newServeCommand(ctx context.Context, cfg global.Config) *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HIC HTTP API",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := persistence.Open(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer repo.Close()
			return api.New(repo).Start(addr)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", ":9552", "HTTP listen address")
	return cmd
}

func newMigrateCommand(ctx context.Context, cfg global.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Run PostgreSQL schema migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return persistence.Migrate(ctx, cfg.DatabaseURL)
		},
	}
}

func newDiscoveryCommand(ctx context.Context, cfg global.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discovery",
		Short: "Discover recruitment notice candidates",
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
		Short: "Discover notices from SH boards",
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
	shCmd.Flags().StringVar(&boardKind, "board", "rental", "SH board kind: rental or sale")
	shCmd.Flags().IntVar(&pages, "pages", 1, "number of board pages to inspect")
	shCmd.Flags().StringVar(&seqs, "seq", "", "comma-separated source notice sequence filter")
	shCmd.Flags().BoolVar(&dryRun, "dry-run", true, "report candidates without persistence")
	shCmd.Flags().BoolVar(&showAttachments, "show-attachments", false, "print candidate attachment metadata")
	shCmd.Flags().IntVar(&maxAgeDays, "max-age-days", 30, "skip notices posted before this many days ago; set 0 to disable")
	shCmd.Flags().BoolVar(&skipExisting, "skip-existing", false, "read PostgreSQL and skip already collected recruitment notices")
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
		Short: "Extract mechanical artifacts from attachments",
	}
	cmd.AddCommand(
		placeholderCommand("attachment", "Extract a stored attachment"),
		newExtractPDFCommand(),
		newExtractXLSXCommand(),
	)
	return cmd
}

func newExtractPDFCommand() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "pdf",
		Short: "Extract text from a PDF file",
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
	cmd.Flags().StringVar(&file, "file", "", "PDF file path")
	return cmd
}

func newExtractXLSXCommand() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "xlsx",
		Short: "Extract rows from an XLSX file",
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
	cmd.Flags().StringVar(&file, "file", "", "XLSX file path")
	return cmd
}

func newNormalizeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "normalize",
		Short: "Normalize extracted artifacts into domain records",
	}
	cmd.AddCommand(
		placeholderCommand("notice", "Normalize notice metadata"),
		placeholderCommand("units", "Normalize housing units"),
		placeholderCommand("schedules", "Normalize notice schedules"),
		placeholderCommand("conversion", "Normalize rent conversion rules"),
	)
	return cmd
}

func newLLMCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "llm",
		Short: "Repair low-confidence artifacts with constrained LLM assistance",
	}
	cmd.AddCommand(placeholderCommand("repair", "Run LLM repair for an artifact"))
	return cmd
}

func newWorkflowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Orchestrate discovery, extraction, normalization, and QA",
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
		Short: "Run the SH collection workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			var knownSeqs map[string]bool
			if skipExisting && strings.TrimSpace(seqs) == "" {
				knownSeqs, err = repo.ExistingNoticeSeqs(cmd.Context(), board.Agency, board.BoardKind)
				if err != nil {
					return err
				}
			}
			report, err := discovery.NewDiscoverer(discovery.NewHTTPFetcher()).Discover(cmd.Context(), board, discovery.Options{
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
			insertedArtifacts := 0
			upsertedUnits := 0
			for _, candidate := range report.Candidates {
				preserveReport, err := collector.PreserveCandidateAttachments(cmd.Context(), board, candidate)
				if err != nil {
					return err
				}
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
					if attachment.Kind == extraction.AttachmentKindHousingUnitListXLSX {
						for _, unit := range normalize.InferHousingUnitsFromXLSXRows(artifacts) {
							artifactID := artifactIDsBySpan[unit.SourceSpan]
							if _, err := repo.UpsertHousingUnit(cmd.Context(), attachment, artifactID, unit); err != nil {
								return err
							}
							upsertedUnits++
						}
					}
				}
			}
			storedObjects, artifacts, err := repo.Counts(cmd.Context())
			if err != nil {
				return err
			}
			units, err := repo.CountHousingUnits(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "db stored_objects=%d extracted_artifacts=%d housing_units=%d inserted_artifacts=%d upserted_units=%d\n", storedObjects, artifacts, units, insertedArtifacts, upsertedUnits)
			return nil
		},
	}
	cmd.Flags().StringVar(&boardKind, "board", "rental", "SH board kind: rental or sale")
	cmd.Flags().IntVar(&pages, "pages", 1, "number of board pages to inspect")
	cmd.Flags().StringVar(&seqs, "seq", "", "comma-separated source notice sequence filter")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "discover candidates without preserving attachments")
	cmd.Flags().BoolVar(&preserveAttachments, "preserve-attachments", false, "download and preserve candidate attachments")
	cmd.Flags().StringVar(&objectRoot, "object-root", ".data/objects", "local ObjectStore root directory")
	cmd.Flags().IntVar(&maxAgeDays, "max-age-days", 30, "skip notices posted before this many days ago; set 0 to disable")
	cmd.Flags().BoolVar(&skipExisting, "skip-existing", true, "skip already collected recruitment notices when --seq is not set")
	return cmd
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

func extractPreservedAttachment(objectStore extraction.LocalObjectStore, attachment persistence.PersistedAttachment) ([]extraction.ExtractedArtifact, error) {
	path, err := objectStore.PathForKey(attachment.ObjectKey)
	if err != nil {
		return nil, err
	}
	switch attachment.Kind {
	case extraction.AttachmentKindNoticePDF, extraction.AttachmentKindSchedulePDF:
		artifact, err := extraction.ExtractPDFText(path)
		if err != nil {
			return nil, err
		}
		return []extraction.ExtractedArtifact{artifact}, nil
	case extraction.AttachmentKindHousingUnitListXLSX:
		return extraction.ExtractXLSXRows(path)
	default:
		return nil, nil
	}
}

func newQACommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qa",
		Short: "Run quality gates and sample regression checks",
	}
	cmd.AddCommand(placeholderCommand("sample", "Run a sample QA case"))
	return cmd
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
