package extraction

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"testing"

	"hic/pkg/global"
)

func TestLocalObjectStore_PutGet_파일을보존하고해시와크기를계산한다(t *testing.T) {
	store := NewLocalObjectStore(t.TempDir())
	body := "hello hic"
	wantSum := sha256.Sum256([]byte(body))
	wantSHA := hex.EncodeToString(wantSum[:])

	stored, err := store.Put(context.Background(), global.Object{
		Key:          "originals/sh/304295/notice.pdf",
		Reader:       strings.NewReader(body),
		ContentType:  "application/pdf",
		OriginalName: "notice.pdf",
	})
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	if stored.Key != "originals/sh/304295/notice.pdf" {
		t.Fatalf("Key = %q", stored.Key)
	}
	if stored.SHA256 != wantSHA {
		t.Fatalf("SHA256 = %q, want %q", stored.SHA256, wantSHA)
	}
	if stored.SizeBytes != int64(len(body)) {
		t.Fatalf("SizeBytes = %d, want %d", stored.SizeBytes, len(body))
	}
	if stored.ContentType != "application/pdf" || stored.OriginalName != "notice.pdf" {
		t.Fatalf("stored metadata = %+v", stored)
	}

	rc, err := store.Get(context.Background(), stored.Key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(got) != body {
		t.Fatalf("stored body = %q, want %q", string(got), body)
	}
}

func TestLocalObjectStore_Put_예상해시가다르면실패한다(t *testing.T) {
	store := NewLocalObjectStore(t.TempDir())

	_, err := store.Put(context.Background(), global.Object{
		Key:            "originals/sh/304295/notice.pdf",
		Reader:         strings.NewReader("hello hic"),
		ExpectedSHA256: "wrong",
	})
	if err == nil {
		t.Fatalf("Put() error = nil, want checksum error")
	}
}

func TestLocalObjectStore_Put_객체키경로탈출을거부한다(t *testing.T) {
	store := NewLocalObjectStore(t.TempDir())

	_, err := store.Put(context.Background(), global.Object{
		Key:    "../escape.pdf",
		Reader: strings.NewReader("hello hic"),
	})
	if err == nil {
		t.Fatalf("Put() error = nil, want invalid key error")
	}
}
