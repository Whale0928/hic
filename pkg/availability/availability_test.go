package availability

import (
	"testing"
	"time"
)

func TestResolve_일정경계는시작과종료시각을포함한다(t *testing.T) {
	start := time.Date(2026, 5, 26, 10, 0, 0, 0, time.Local)
	end := time.Date(2026, 5, 28, 17, 0, 0, 0, time.Local)

	for _, now := range []time.Time{start, end} {
		got := Resolve(now, []ScheduleEvidence{{StartsAt: start, EndsAt: end}}, nil)

		if got.Status != StatusOpen {
			t.Fatalf("Resolve(%s).Status = %q, want %q", now, got.Status, StatusOpen)
		}
	}
}

func TestResolve_미래일정은예정_지난일정은마감이다(t *testing.T) {
	start := time.Date(2026, 5, 26, 10, 0, 0, 0, time.Local)
	end := time.Date(2026, 5, 28, 17, 0, 0, 0, time.Local)

	pending := Resolve(start.Add(-time.Second), []ScheduleEvidence{{StartsAt: start, EndsAt: end}}, nil)
	if pending.Status != StatusPending {
		t.Fatalf("pending.Status = %q, want %q", pending.Status, StatusPending)
	}

	closed := Resolve(end.Add(time.Second), []ScheduleEvidence{{StartsAt: start, EndsAt: end}}, nil)
	if closed.Status != StatusClosed {
		t.Fatalf("closed.Status = %q, want %q", closed.Status, StatusClosed)
	}
}

func TestResolve_인터넷청약청약중은일정이없어도신청가능이다(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.Local)

	got := Resolve(now, nil, []ApplicationEvidence{{Status: "청약중"}})

	if got.Status != StatusOpen {
		t.Fatalf("Status = %q, want %q", got.Status, StatusOpen)
	}
	if got.Source != SourceApplicationNotice {
		t.Fatalf("Source = %q, want %q", got.Source, SourceApplicationNotice)
	}
}

func TestResolve_인터넷청약과일정이충돌하면신뢰도경고를남긴다(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.Local)
	futureStart := time.Date(2026, 5, 26, 10, 0, 0, 0, time.Local)
	futureEnd := time.Date(2026, 5, 28, 17, 0, 0, 0, time.Local)

	got := Resolve(
		now,
		[]ScheduleEvidence{{StartsAt: futureStart, EndsAt: futureEnd}},
		[]ApplicationEvidence{{Status: "청약중"}},
	)

	if got.Status != StatusOpen {
		t.Fatalf("Status = %q, want %q", got.Status, StatusOpen)
	}
	if !got.Conflict {
		t.Fatal("Conflict = false, want true")
	}
	if got.ScheduleStatus != StatusPending || got.ApplicationStatus != StatusOpen {
		t.Fatalf("statuses = schedule:%q application:%q", got.ScheduleStatus, got.ApplicationStatus)
	}
}
