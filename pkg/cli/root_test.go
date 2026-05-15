package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"hic/pkg/discovery"
	"hic/pkg/extraction"
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
		{"normalize", "--help"},
		{"llm", "--help"},
		{"workflow", "--help"},
		{"workflow", "collect-sh", "--help"},
		{"qa", "--help"},
		{"qa", "promote-units", "--help"},
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

func TestNewRootCommand_QAHelp가승격명령을노출한다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"qa", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "promote-units") {
		t.Fatalf("qa help = %q, want promote-units", out.String())
	}
}

func TestFormatQASummary_승격결과를출력한다(t *testing.T) {
	got := formatQASummary(persistence.QASummary{Approved: 25, Rejected: 2, Pending: 1})

	if got != "qa approved=25 rejected=2 pending=1\n" {
		t.Fatalf("formatQASummary() = %q", got)
	}
}

func TestNormalizeHousingUnitsFromArtifacts_PDF신청주택표를정규화한다(t *testing.T) {
	units := normalizeHousingUnitsFromArtifacts(extraction.AttachmentKindNoticePDF, []extraction.ExtractedArtifact{{
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

	if len(units) != 1 {
		t.Fatalf("normalizeHousingUnitsFromArtifacts() len = %d, want 1", len(units))
	}
	if units[0].UnitNo != "502호" {
		t.Fatalf("UnitNo = %q", units[0].UnitNo)
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
