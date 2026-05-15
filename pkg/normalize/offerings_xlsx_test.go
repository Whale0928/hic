package normalize

import (
	"testing"

	"hic/pkg/extraction"
)

func TestInferOfferingsFromXLSXRows_주택목록행을공급항목후보로변환한다(t *testing.T) {
	artifacts := []extraction.ExtractedArtifact{
		xlsxRow("주택목록", 1, []string{"단지명", "동", "호", "주택형", "전용면적", "보증금", "월임대료"}),
		xlsxRow("주택목록", 2, []string{"행복주택 A", "101", "1201", "36A", "36.52", "50,000,000", "230,000"}),
	}

	offerings := InferOfferingsFromXLSXRows(artifacts)

	if len(offerings) != 1 {
		t.Fatalf("len(offerings) = %d, want 1: %+v", len(offerings), offerings)
	}
	got := offerings[0]
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

func TestInferOfferingsFromXLSXRows_당첨자명단헤더는제외한다(t *testing.T) {
	artifacts := []extraction.ExtractedArtifact{
		xlsxRow("당첨자", 1, []string{"접수번호", "성명", "생월일", "동호"}),
		xlsxRow("당첨자", 2, []string{"A-1", "홍길동", "0101", "101-1201"}),
	}

	offerings := InferOfferingsFromXLSXRows(artifacts)

	if len(offerings) != 0 {
		t.Fatalf("len(offerings) = %d, want 0: %+v", len(offerings), offerings)
	}
}

func TestInferOfferingsFromXLSXRows_동호수없는신청가능단위를공급항목후보로변환한다(t *testing.T) {
	artifacts := []extraction.ExtractedArtifact{
		xlsxRow("공급현황", 1, []string{"단지명", "전용면적", "유형", "공급호수", "예비", "전세금액(천원)", "계약금(계약시)", "잔금(입주시)", "성별", "주택유형", "기숙사비", "난방방식", "입주시작(예정)"}),
		xlsxRow("공급현황", 2, []string{"세곡2지구", "59.00", "일반", "34", "5", "514,020", "51,402", "462,618", "여성", "원룸형 1인실", "120,000", "지역난방", "2026.12"}),
	}

	offerings := InferOfferingsFromXLSXRows(artifacts)

	if len(offerings) != 1 {
		t.Fatalf("len(offerings) = %d, want 1: %+v", len(offerings), offerings)
	}
	got := offerings[0]
	if got.UnitNo != "" {
		t.Fatalf("UnitNo = %q, want empty", got.UnitNo)
	}
	if got.ApplicationUnitLabel != "세곡2지구 59㎡ 일반 여성" {
		t.Fatalf("ApplicationUnitLabel = %q", got.ApplicationUnitLabel)
	}
	if got.SupplyCount == nil || *got.SupplyCount != 34 {
		t.Fatalf("SupplyCount = %v", got.SupplyCount)
	}
	if got.ReservedCount == nil || *got.ReservedCount != 5 {
		t.Fatalf("ReservedCount = %v", got.ReservedCount)
	}
	if got.JeonseDepositKRW == nil || *got.JeonseDepositKRW != 514020000 {
		t.Fatalf("JeonseDepositKRW = %v", got.JeonseDepositKRW)
	}
	if got.ContractDepositKRW == nil || *got.ContractDepositKRW != 51402000 {
		t.Fatalf("ContractDepositKRW = %v", got.ContractDepositKRW)
	}
	if got.BalancePaymentKRW == nil || *got.BalancePaymentKRW != 462618000 {
		t.Fatalf("BalancePaymentKRW = %v", got.BalancePaymentKRW)
	}
	if got.GenderRequirement != "여성" || got.OccupancyType != "원룸형 1인실" {
		t.Fatalf("application attributes = %+v", got)
	}
	if got.DormitoryFeeKRW == nil || *got.DormitoryFeeKRW != 120000 {
		t.Fatalf("DormitoryFeeKRW = %v", got.DormitoryFeeKRW)
	}
	if got.HeatingMethod != "지역난방" || got.MoveInStartText != "2026.12" {
		t.Fatalf("facility/schedule attributes = %+v", got)
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
