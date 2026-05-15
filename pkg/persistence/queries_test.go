package persistence

import (
	"os"
	"strings"
	"testing"
)

func TestUpsertOfferingSQL_승인된QA상태는재수집때유지한다(t *testing.T) {
	sql := readPocSQL(t)

	want := "case when offerings.qa_status = 'approved' then 'approved'"
	if !strings.Contains(sql, want) {
		t.Fatalf("UpsertOffering SQL should preserve approved qa_status on conflict; missing %q", want)
	}
}

func TestPromoteOfferingsQASQL_필수도메인필드를검증한다(t *testing.T) {
	sql := readPocSQL(t)

	if strings.Contains(sql, "offering_type") {
		t.Fatalf("PromoteOfferingsQA SQL should not depend on offering_type")
	}

	for _, want := range []string{
		"exists (",
		"sn.category = 'recruitment'",
		"trim(application_unit_label) <> ''",
		"coalesce(trim(unit_no), '') <> ''",
		"supply_count > 0",
		"exclusive_area_m2 > 0",
		"trim(occupancy_type) <> ''",
		"deposit_krw is not null",
		"deposit_krw >= 0",
		"monthly_rent_krw is not null",
		"monthly_rent_krw >= 0",
		"jeonse_deposit_krw is not null",
		"jeonse_deposit_krw >= 0",
		"dormitory_fee_krw is not null",
		"dormitory_fee_krw >= 0",
		"trim(source_span) <> ''",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("PromoteOfferingsQA SQL missing QA rule %q", want)
		}
	}
}

func readPocSQL(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("queries/poc.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	return string(b)
}
