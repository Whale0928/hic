package persistence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schemaRelativePath = "schema/schema.sql"

func Migrate(ctx context.Context, databaseURL string) error {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("open postgres pool: %w", err)
	}
	defer pool.Close()

	schemaSQL, err := LoadSchemaSQL()
	if err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("run schema migration: %w", err)
	}
	return nil
}

func LoadSchemaSQL() (string, error) {
	path, err := findUpward(schemaRelativePath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read schema SQL: %w", err)
	}
	return string(data), nil
}

func findUpward(relativePath string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, relativePath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat schema SQL: %w", err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("schema SQL not found: %s", relativePath)
		}
		dir = parent
	}
}
