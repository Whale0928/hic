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

func TestUpsertNoticeSQL_Null게시일은기존값을덮지않는다(t *testing.T) {
	sql := readPocSQL(t)

	want := "posted_at = coalesce(excluded.posted_at, source_notices.posted_at)"
	if !strings.Contains(sql, want) {
		t.Fatalf("UpsertNotice SQL should preserve posted_at when seq-only collection has no list date; missing %q", want)
	}
}

func TestPromoteOfferingsQASQL_필수도메인필드를검증한다(t *testing.T) {
	sql := readPocSQL(t)

	if strings.Contains(sql, "offering_type") {
		t.Fatalf("PromoteOfferingsQA SQL should not depend on offering_type")
	}
	if strings.Contains(sql, "where qa_status = 'pending'") {
		t.Fatalf("PromoteOfferingsQA SQL should re-evaluate approved offerings when QA rules change")
	}
	if !strings.Contains(sql, "min(id) filter (where passes)") ||
		!strings.Contains(sql, "canonical_id") ||
		!strings.Contains(sql, "label_key") ||
		!strings.Contains(sql, "then 'approved'") ||
		!strings.Contains(sql, "else 'rejected'") {
		t.Fatalf("PromoteOfferingsQA SQL should approve only one duplicate per application-selectable label/source identity and reject the rest")
	}

	for _, want := range []string{
		"exists (",
		"sn.category = 'recruitment'",
		"(attachment_id is not null or source = 'myhome')",
		"trim(application_unit_label) <> ''",
		"coalesce(trim(unit_no), '') <> ''",
		"trim(list_no) <> ''",
		"supply_count > 0",
		"trim(unit_type) <> ''",
		"trim(supply_category) <> ''",
		"trim(gender_requirement) <> ''",
		"source = 'myhome'",
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
		"contract_deposit_krw is not null",
		"contract_deposit_krw >= 0",
		"balance_payment_krw is not null",
		"balance_payment_krw >= 0",
		"trim(source_span) <> ''",
		"source_span like 'object://%'",
		"source_span like 'myhome://%'",
		"char_length(application_unit_label) <= 256",
		"char_length(coalesce(nullif(housing_name, ''), complex_name)) <= 128",
		"position('�' in application_unit_label) = 0",
		"position('�' in coalesce(nullif(housing_name, ''), complex_name)) = 0",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("PromoteOfferingsQA SQL missing QA rule %q", want)
		}
	}
}

func TestSchemaSQL_신청가능단위명은PrefixIndex를사용한다(t *testing.T) {
	sql := readSchemaSQL(t)

	if strings.Contains(sql, "idx_offerings_application_unit_label on offerings(application_unit_label)") {
		t.Fatalf("application_unit_label full btree index can exceed PostgreSQL index row size")
	}
	if !strings.Contains(sql, "idx_offerings_application_unit_label_prefix") ||
		!strings.Contains(sql, "left(application_unit_label, 512)") {
		t.Fatalf("schema should use bounded prefix index for application_unit_label")
	}
	if strings.Contains(sql, "coalesce(nullif(housing_name, ''), complex_name))") {
		t.Fatalf("source identity unique index should not btree-index unbounded housing_name/complex_name text")
	}
	if !strings.Contains(sql, "md5(coalesce(unit_no, ''))") ||
		!strings.Contains(sql, "md5(coalesce(nullif(housing_name, ''), complex_name, ''))") ||
		!strings.Contains(sql, "md5(coalesce(application_unit_label, ''))") ||
		!strings.Contains(sql, "md5(coalesce(source_span, ''))") {
		t.Fatalf("source identity unique index should hash unbounded text fields and include application unit/source evidence")
	}
}

func TestJSONBytesOrEmpty_빈LLM응답을빈JSON객체로저장한다(t *testing.T) {
	got := string(jsonBytesOrEmpty(nil))

	if got != "{}" {
		t.Fatalf("jsonBytesOrEmpty(nil) = %q, want {}", got)
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

func readSchemaSQL(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../schema/schema.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	return string(b)
}
