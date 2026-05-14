package persistence

import (
	"strings"
	"testing"

	"hic/pkg/discovery"
)

func TestValidatePersistableCandidate_모집공고계열만허용한다(t *testing.T) {
	candidate := discovery.Candidate{
		Agency:    "SH",
		BoardKind: "rental",
		Seq:       "304295",
		Title:     "2026년 휘경마을 두레주택 잔여세대 입주자 모집공고",
	}

	if err := ValidatePersistableCandidate(candidate); err != nil {
		t.Fatalf("ValidatePersistableCandidate() error = %v", err)
	}
}

func TestValidatePersistableCandidate_당첨자발표는저장전차단한다(t *testing.T) {
	candidate := discovery.Candidate{
		Agency:    "SH",
		BoardKind: "rental",
		Seq:       "296353",
		Title:     "[당첨자 발표] 2025년 1차 청년 매입임대주택 입주자모집 당첨자 및 예비자 발표",
	}

	err := ValidatePersistableCandidate(candidate)

	if err == nil {
		t.Fatal("ValidatePersistableCandidate() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "non-recruitment") {
		t.Fatalf("error = %v, want non-recruitment", err)
	}
}
