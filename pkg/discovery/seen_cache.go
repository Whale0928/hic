package discovery

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type SeenCacheStatus string

const (
	SeenCacheStatusRejected        SeenCacheStatus = "rejected"
	SeenCacheStatusRejectedUnknown SeenCacheStatus = "rejected_unknown"
)

type SeenCacheEntry struct {
	Seq           string
	Status        SeenCacheStatus
	ListTitleHash string
	PostedAt      time.Time
	ExpiresAt     time.Time
}

func (e SeenCacheEntry) CanSkipDetail(row BoardRow, now time.Time) bool {
	if e.Status != SeenCacheStatusRejected && e.Status != SeenCacheStatusRejectedUnknown {
		return false
	}
	if !e.ExpiresAt.IsZero() && !now.Before(e.ExpiresAt) {
		return false
	}
	if e.ListTitleHash != "" && e.ListTitleHash != SeenTitleHash(row.Title) {
		return false
	}
	if !e.PostedAt.IsZero() && !row.PostedAt.IsZero() && !sameDate(e.PostedAt, row.PostedAt) {
		return false
	}
	return true
}

func SeenTitleHash(title string) string {
	sum := sha256.Sum256([]byte(targetTitleKey(title)))
	return hex.EncodeToString(sum[:])
}

func sameDate(a time.Time, b time.Time) bool {
	return a.Format(time.DateOnly) == b.Format(time.DateOnly)
}
