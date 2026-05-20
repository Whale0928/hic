package cli

import (
	"testing"
	"time"

	"hic/pkg/discovery"
)

func TestDiscoverySeenCacheInputFromRejectedPost_거절사유별TTL과Status를정한다(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	posted := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)

	rejected := discoverySeenCacheInputFromRejectedPost(discovery.RejectedPost{
		Agency:    "SH",
		BoardKind: "rental",
		Seq:       "304999",
		Title:     "당첨자 발표 안내",
		Reason:    discovery.NoticeCategoryRejected,
		PostedAt:  posted,
	}, now)
	if rejected.Status != discovery.SeenCacheStatusRejected {
		t.Fatalf("Status = %q", rejected.Status)
	}
	if rejected.ExpiresAt != now.Add(60*24*time.Hour) {
		t.Fatalf("ExpiresAt = %s", rejected.ExpiresAt)
	}
	if rejected.PostedAt != posted {
		t.Fatalf("PostedAt = %s", rejected.PostedAt)
	}
	if rejected.Evidence["reason"] != string(discovery.NoticeCategoryRejected) {
		t.Fatalf("Evidence = %+v", rejected.Evidence)
	}

	unknown := discoverySeenCacheInputFromRejectedPost(discovery.RejectedPost{
		Agency:    "SH",
		BoardKind: "rental",
		Seq:       "305000",
		Title:     "분류 불명 공지",
		Reason:    discovery.NoticeCategoryUnknown,
	}, now)
	if unknown.Status != discovery.SeenCacheStatusRejectedUnknown {
		t.Fatalf("Status = %q", unknown.Status)
	}
	if unknown.ExpiresAt != now.Add(7*24*time.Hour) {
		t.Fatalf("ExpiresAt = %s", unknown.ExpiresAt)
	}
}
