package global

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL    string
	ObjectRoot     string
	OpenAIAPIKey   string
	OpenAIModel    string
	OpenAIBaseURL  string
	MyHomeAPIKey   string
	LLMMaxAttempts int
}

func FromEnv() Config {
	return Config{
		DatabaseURL:   env("DATABASE_URL", "postgres://shdata:shdata@localhost:9551/shdata?sslmode=disable"),
		ObjectRoot:    env("HIC_OBJECT_ROOT", ".data/objects"),
		OpenAIAPIKey:  env("OPENAI_API_KEY", env("GPT_API_KEY", "")),
		OpenAIModel:   env("HIC_OPENAI_MODEL", "gpt-5.4-mini"),
		OpenAIBaseURL: env("HIC_OPENAI_BASE_URL", ""),
		MyHomeAPIKey:  env("MYHOME_API_KEY", ""),
		LLMMaxAttempts: envInt(
			"HIC_LLM_MAX_ATTEMPTS",
			1500,
		),
	}
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		value = dotenvValue(key)
	}
	if value == "" {
		value = fallback
	}
	return value
}

func dotenvValue(key string) string {
	data, err := os.ReadFile(".env")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(name) != key {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		return value
	}
	return ""
}

func envInt(key string, fallback int) int {
	value := env(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > 1500 {
		return 1500
	}
	return parsed
}
