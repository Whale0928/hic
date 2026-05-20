package extraction

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

func ExtractXLSXRows(path string) ([]ExtractedArtifact, error) {
	return ExtractXLSXRowsWithSource(path, "")
}

func ExtractXLSXRowsWithSource(path string, sourceSpan string) ([]ExtractedArtifact, error) {
	f, err := excelize.OpenFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	var artifacts []ExtractedArtifact
	for _, sheet := range f.GetSheetList() {
		rows, err := f.Rows(sheet)
		if err != nil {
			return nil, fmt.Errorf("open sheet %s: %w", sheet, err)
		}
		rowNumber := 0
		for rows.Next() {
			rowNumber++
			cols, err := rows.Columns()
			if err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("read sheet %s row %d: %w", sheet, rowNumber, err)
			}
			cells := normalizeCells(cols)
			artifacts = append(artifacts, ExtractedArtifact{
				Type:          ArtifactTypeXLSXRow,
				Extractor:     "excelize",
				Status:        ArtifactStatusExtracted,
				SchemaVersion: "v1",
				SourceSheet:   sheet,
				SourceRow:     rowNumber,
				SourceSpan:    xlsxSourceSpan(sourceSpan, sheet, rowNumber),
				Content: map[string]any{
					"cells": cells,
				},
				Confidence: 1,
			})
		}
		if err := rows.Close(); err != nil {
			return nil, fmt.Errorf("close sheet %s rows: %w", sheet, err)
		}
	}

	return artifacts, nil
}

func xlsxSourceSpan(sourceSpan string, sheet string, rowNumber int) string {
	sourceSpan = strings.TrimSpace(sourceSpan)
	if sourceSpan == "" {
		return fmt.Sprintf("xlsx://%s!%d", sheet, rowNumber)
	}
	return fmt.Sprintf("%s#sheet=%s&row=%d", sourceSpan, url.QueryEscape(sheet), rowNumber)
}

func normalizeCells(cells []string) []string {
	out := make([]string, len(cells))
	for i, cell := range cells {
		out[i] = strings.TrimSpace(strings.Join(strings.Fields(cell), " "))
	}
	return out
}
