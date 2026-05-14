package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"hic/pkg/discovery"

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
		{"normalize", "--help"},
		{"llm", "--help"},
		{"workflow", "--help"},
		{"workflow", "collect-sh", "--help"},
		{"qa", "--help"},
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

func TestNewRootCommand_ExtractXLSX_Artifact개수를출력한다(t *testing.T) {
	path := filepath.Join(t.TempDir(), "units.xlsx")
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
		"inserted_artifacts": 31,
		"upserted_units":     25,
		"stored_objects":     int64(7),
		"total_artifacts":    int64(31),
		"total_units":        int64(25),
	}
	for key, want := range tests {
		if got := stats[key]; got != want {
			t.Fatalf("stats[%s] = %#v, want %#v", key, got, want)
		}
	}
}
