package normalize

import (
	"testing"

	"hic/pkg/extraction"
)

func TestInferHousingUnitsFromXLSXRows_주택목록행을주택후보로변환한다(t *testing.T) {
	artifacts := []extraction.ExtractedArtifact{
		xlsxRow("주택목록", 1, []string{"단지명", "동", "호", "주택형", "전용면적", "보증금", "월임대료"}),
		xlsxRow("주택목록", 2, []string{"행복주택 A", "101", "1201", "36A", "36.52", "50,000,000", "230,000"}),
	}

	units := InferHousingUnitsFromXLSXRows(artifacts)

	if len(units) != 1 {
		t.Fatalf("len(units) = %d, want 1: %+v", len(units), units)
	}
	got := units[0]
	if got.HousingName != "행복주택 A" || got.BuildingName != "101" || got.UnitNo != "1201" {
		t.Fatalf("unit identity = %+v", got)
	}
	if got.UnitType != "36A" {
		t.Fatalf("UnitType = %q", got.UnitType)
	}
	if got.ExclusiveAreaM2 == nil || *got.ExclusiveAreaM2 != 36.52 {
		t.Fatalf("ExclusiveAreaM2 = %v", got.ExclusiveAreaM2)
	}
	if got.DepositKRW == nil || *got.DepositKRW != 50000000 {
		t.Fatalf("DepositKRW = %v", got.DepositKRW)
	}
	if got.MonthlyRentKRW == nil || *got.MonthlyRentKRW != 230000 {
		t.Fatalf("MonthlyRentKRW = %v", got.MonthlyRentKRW)
	}
	if got.SourceSheet != "주택목록" || got.SourceRow != 2 || got.SourceSpan != "xlsx://주택목록!2" {
		t.Fatalf("source = %+v", got)
	}
}

func TestInferHousingUnitsFromXLSXRows_당첨자명단헤더는제외한다(t *testing.T) {
	artifacts := []extraction.ExtractedArtifact{
		xlsxRow("당첨자", 1, []string{"접수번호", "성명", "생월일", "동호"}),
		xlsxRow("당첨자", 2, []string{"A-1", "홍길동", "0101", "101-1201"}),
	}

	units := InferHousingUnitsFromXLSXRows(artifacts)

	if len(units) != 0 {
		t.Fatalf("len(units) = %d, want 0: %+v", len(units), units)
	}
}

func xlsxRow(sheet string, row int, cells []string) extraction.ExtractedArtifact {
	return extraction.ExtractedArtifact{
		Type:          extraction.ArtifactTypeXLSXRow,
		SchemaVersion: "v1",
		SourceSheet:   sheet,
		SourceRow:     row,
		SourceSpan:    "xlsx://" + sheet + "!" + string(rune('0'+row)),
		Content: map[string]any{
			"cells": cells,
		},
	}
}
