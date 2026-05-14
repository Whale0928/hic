package global

import (
	"context"
	"io"
	"testing"
	"time"
)

type fakeObjectStore struct{}

func (fakeObjectStore) Put(ctx context.Context, object Object) (StoredObject, error) {
	return StoredObject{Key: object.Key}, nil
}
func (fakeObjectStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(nil), nil
}

type fakeQueue struct{}

func (fakeQueue) Enqueue(ctx context.Context, job Job) error { return nil }
func (fakeQueue) Dequeue(ctx context.Context, queue string) (Job, error) {
	return Job{Queue: queue}, nil
}

type fakeLock struct{}

func (fakeLock) WithLock(ctx context.Context, key string, ttl time.Duration, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestPorts_인프라계약을정의한다(t *testing.T) {
	var _ ObjectStore = fakeObjectStore{}
	var _ Queue = fakeQueue{}
	var _ Lock = fakeLock{}
}
