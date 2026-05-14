package extraction

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hic/pkg/global"
)

type LocalObjectStore struct {
	root string
}

func NewLocalObjectStore(root string) LocalObjectStore {
	return LocalObjectStore{root: root}
}

func (s LocalObjectStore) Put(ctx context.Context, object global.Object) (global.StoredObject, error) {
	if object.Reader == nil {
		return global.StoredObject{}, fmt.Errorf("object reader is required")
	}
	path, err := s.pathForKey(object.Key)
	if err != nil {
		return global.StoredObject{}, err
	}
	if err := ctx.Err(); err != nil {
		return global.StoredObject{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return global.StoredObject{}, fmt.Errorf("create object directory: %w", err)
	}

	tmp := path + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return global.StoredObject{}, fmt.Errorf("create object file: %w", err)
	}

	hasher := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(file, hasher), object.Reader)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return global.StoredObject{}, fmt.Errorf("write object: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return global.StoredObject{}, fmt.Errorf("close object file: %w", closeErr)
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	if object.ExpectedSHA256 != "" && !strings.EqualFold(object.ExpectedSHA256, sum) {
		_ = os.Remove(tmp)
		return global.StoredObject{}, fmt.Errorf("object checksum mismatch: got %s want %s", sum, object.ExpectedSHA256)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return global.StoredObject{}, fmt.Errorf("commit object file: %w", err)
	}

	return global.StoredObject{
		Key:          object.Key,
		ContentType:  object.ContentType,
		OriginalName: object.OriginalName,
		SHA256:       sum,
		SizeBytes:    size,
		StoredAt:     time.Now(),
		Metadata:     object.Metadata,
	}, nil
}

func (s LocalObjectStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	path, err := s.PathForKey(key)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open object: %w", err)
	}
	return file, nil
}

func (s LocalObjectStore) PathForKey(key string) (string, error) {
	key = strings.TrimSpace(filepath.ToSlash(key))
	if key == "" {
		return "", fmt.Errorf("object key is required")
	}
	if strings.HasPrefix(key, "/") || strings.Contains(key, "../") || key == ".." {
		return "", fmt.Errorf("invalid object key: %s", key)
	}
	cleanRoot := filepath.Clean(s.root)
	path := filepath.Join(cleanRoot, filepath.FromSlash(key))
	rel, err := filepath.Rel(cleanRoot, path)
	if err != nil {
		return "", fmt.Errorf("validate object path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid object key: %s", key)
	}
	return path, nil
}

func (s LocalObjectStore) pathForKey(key string) (string, error) {
	return s.PathForKey(key)
}
