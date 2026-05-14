package global

import (
	"context"
	"io"
	"time"
)

type Object struct {
	Key            string
	Reader         io.Reader
	ContentType    string
	OriginalName   string
	ExpectedSHA256 string
	Metadata       map[string]string
}

type StoredObject struct {
	Key          string
	ContentType  string
	OriginalName string
	SHA256       string
	SizeBytes    int64
	StoredAt     time.Time
	Metadata     map[string]string
}

type ObjectStore interface {
	Put(ctx context.Context, object Object) (StoredObject, error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

type Job struct {
	ID        string
	Queue     string
	Kind      string
	Payload   []byte
	CreatedAt time.Time
}

type Queue interface {
	Enqueue(ctx context.Context, job Job) error
	Dequeue(ctx context.Context, queue string) (Job, error)
}

type Lock interface {
	WithLock(ctx context.Context, key string, ttl time.Duration, fn func(context.Context) error) error
}
