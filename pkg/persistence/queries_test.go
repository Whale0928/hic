package persistence

import (
	"os"
	"strings"
	"testing"
)

func TestUpsertHousingUnitSQL_승인된QA상태는재수집때유지한다(t *testing.T) {
	sql := readPocSQL(t)

	want := "case when housing_units.qa_status = 'approved' then 'approved'"
	if !strings.Contains(sql, want) {
		t.Fatalf("UpsertHousingUnit SQL should preserve approved qa_status on conflict; missing %q", want)
	}
}

func TestPromoteHousingUnitsQASQL_필수도메인필드를검증한다(t *testing.T) {
	sql := readPocSQL(t)

	for _, want := range []string{
		"exists (",
		"sn.category = 'recruitment'",
		"trim(unit_no) <> ''",
		"trim(address) <> ''",
		"exclusive_area_m2 > 0",
		"deposit_krw is not null",
		"deposit_krw >= 0",
		"monthly_rent_krw is not null",
		"monthly_rent_krw >= 0",
		"trim(source_span) <> ''",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("PromoteHousingUnitsQA SQL missing QA rule %q", want)
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
