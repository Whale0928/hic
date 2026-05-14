package global

import (
	"strings"
	"testing"
)

func TestFromEnv_기본데이터베이스포트는9551을사용한다(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	cfg := FromEnv()

	if !strings.Contains(cfg.DatabaseURL, "localhost:9551") {
		t.Fatalf("DatabaseURL = %q, want localhost:9551", cfg.DatabaseURL)
	}
}
