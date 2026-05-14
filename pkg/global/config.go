package global

import "os"

type Config struct {
	DatabaseURL string
	ObjectRoot  string
}

func FromEnv() Config {
	return Config{
		DatabaseURL: env("DATABASE_URL", "postgres://shdata:shdata@localhost:9551/shdata?sslmode=disable"),
		ObjectRoot:  env("HIC_OBJECT_ROOT", ".data/objects"),
	}
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
