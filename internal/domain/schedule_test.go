package domain

import (
	"testing"
	"time"
)

func TestNextDue(t *testing.T) {
	t.Parallel()
	if got := NextDue(t0, time.Hour, 0, 0); !got.Equal(t0.Add(time.Hour)) {
		t.Errorf("healthy next due = %v", got)
	}
	week := 7 * 24 * time.Hour
	if got := NextDue(t0, week, 3, 0); !got.Equal(t0.Add(week)) {
		t.Errorf("an interval longer than the ceiling must stand: %v", got)
	}
}

func TestBackoffAndRetryAfter(t *testing.T) {
	t.Parallel()
	waits := map[int]time.Duration{
		1: 5 * time.Minute, 2: 10 * time.Minute, 3: 20 * time.Minute, 4: 40 * time.Minute, 20: MaxBackoff,
	}
	for failures, want := range waits {
		if got := NextDue(t0, 5*time.Minute, failures, 0).Sub(t0); got != want {
			t.Errorf("failures %d: wait %v, want %v", failures, got, want)
		}
	}
	if got := NextDue(t0, 5*time.Minute, 1, 2*time.Hour).Sub(t0); got != 2*time.Hour {
		t.Errorf("Retry-After not honoured: %v", got)
	}
	if got := NextDue(t0, 5*time.Minute, 1, 30*24*time.Hour).Sub(t0); got != MaxBackoff {
		t.Errorf("Retry-After not capped: %v", got)
	}
	if got := NextDue(t0, time.Hour, 1, time.Minute).Sub(t0); got != time.Hour {
		t.Errorf("a short Retry-After must not shorten the wait: %v", got)
	}
}
