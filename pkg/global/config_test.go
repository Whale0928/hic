package global

import (
	"os"
	"path/filepath"
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

func TestFromEnv_GPTAPIKey를OpenAIKeyFallback으로사용한다(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GPT_API_KEY", "gpt-key")

	cfg := FromEnv()

	if cfg.OpenAIAPIKey != "gpt-key" {
		t.Fatalf("OpenAIAPIKey = %q, want GPT_API_KEY fallback", cfg.OpenAIAPIKey)
	}
}

func TestFromEnv_MyHomeAPIKey를읽는다(t *testing.T) {
	t.Setenv("MYHOME_API_KEY", "myhome-key")

	cfg := FromEnv()

	if cfg.MyHomeAPIKey != "myhome-key" {
		t.Fatalf("MyHomeAPIKey = %q", cfg.MyHomeAPIKey)
	}
}

func TestFromEnv_LLMMaxAttempts기본값은1500이다(t *testing.T) {
	t.Setenv("HIC_LLM_MAX_ATTEMPTS", "")

	cfg := FromEnv()

	if cfg.LLMMaxAttempts != 1500 {
		t.Fatalf("LLMMaxAttempts = %d, want 1500", cfg.LLMMaxAttempts)
	}
}

func TestFromEnv_DotEnv파일의GPTAPIKey를읽는다(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GPT_API_KEY", "")
	t.Chdir(t.TempDir())
	if err := os.WriteFile(filepath.Join(".", ".env"), []byte("GPT_API_KEY=dotenv-gpt-key\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := FromEnv()

	if cfg.OpenAIAPIKey != "dotenv-gpt-key" {
		t.Fatalf("OpenAIAPIKey = %q, want .env GPT_API_KEY", cfg.OpenAIAPIKey)
	}
}
