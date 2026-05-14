package extraction

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestExtractXLSXRows_행Artifact를생성한다(t *testing.T) {
	path := filepath.Join(t.TempDir(), "units.xlsx")
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	values := [][]any{
		{"동", "호", "전용면적", "보증금", "월임대료"},
		{"101", "1201", "36.12", "4200", "21"},
	}
	for r, row := range values {
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

	artifacts, err := ExtractXLSXRows(path)
	if err != nil {
		t.Fatalf("ExtractXLSXRows() error = %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("len(artifacts) = %d, want 2", len(artifacts))
	}
	got := artifacts[1]
	if got.Type != ArtifactTypeXLSXRow || got.Status != ArtifactStatusExtracted {
		t.Fatalf("artifact type/status = %+v", got)
	}
	if got.SourceSheet != sheet || got.SourceRow != 2 || got.SourceSpan == "" {
		t.Fatalf("artifact source = %+v", got)
	}
	if got.Content["cells"] == nil {
		t.Fatalf("artifact content = %+v", got.Content)
	}
}
