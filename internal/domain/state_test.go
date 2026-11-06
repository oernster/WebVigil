package domain

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)

func TestLight(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		s    SiteState
		want Light
	}{
		{"never checked", SiteState{}, LightGrey},
		{"failed, no baseline", SiteState{LastOutcome: OutcomeFailure}, LightRed},
		{"healthy", SiteState{LastOutcome: OutcomeSuccess, BaselineHash: "h"}, LightGreen},
		{"changed", SiteState{LastOutcome: OutcomeSuccess, BaselineHash: "h", PendingChangeID: 1}, LightAmber},
		{"changed then failed", SiteState{LastOutcome: OutcomeFailure, BaselineHash: "h", PendingChangeID: 1}, LightRed},
	}
	for _, c := range cases {
		if got := c.s.Light(); got != c.want {
			t.Errorf("%s: light = %s, want %s", c.name, got, c.want)
		}
	}
}

func TestFailureDoesNotHidePendingChange(t *testing.T) {
	t.Parallel()
	s := SiteState{LastOutcome: OutcomeSuccess, BaselineHash: "h", BaselineSnapshotID: 7, PendingChangeID: 3}
	s = s.AfterFailure(CheckFacts{At: t0, HTTPStatus: 503, Category: CategoryHTTP503, Detail: "d"}, time.Hour)
	if s.Light() != LightRed || !s.HasPendingAlert() || s.BaselineHash != "h" || s.BaselineSnapshotID != 7 {
		t.Errorf("state lost a fact: %+v", s)
	}
	if s.LastHTTPStatus != 503 || s.LastCategory != CategoryHTTP503 || s.ConsecutiveFailures != 1 ||
		!s.LastCheckedAt.Equal(t0) || !s.NextDueAt.Equal(t0.Add(time.Hour)) {
		t.Errorf("failure facts wrong: %+v", s)
	}
}

func TestAfterSuccess(t *testing.T) {
	t.Parallel()
	s := SiteState{ConsecutiveFailures: 4, FailureWarned: true, ETag: "old", LastOutcome: OutcomeFailure}
	got := s.AfterSuccess(CheckFacts{At: t0, HTTPStatus: 200, ETag: "e"}, time.Hour, ChangeBaseline, "h1")
	if got.Light() != LightGreen || got.BaselineHash != "h1" || got.ConsecutiveFailures != 0 || got.FailureWarned ||
		got.ETag != "e" || !got.LastSuccessAt.Equal(t0) || !got.LastChangeAt.IsZero() {
		t.Errorf("baseline state wrong: %+v", got)
	}
	same := got.AfterSuccess(CheckFacts{At: t0.Add(time.Hour), HTTPStatus: 304}, time.Hour, ChangeUnchanged, "")
	if same.BaselineHash != "h1" || same.ETag != "e" {
		t.Errorf("unchanged check altered the baseline or validators: %+v", same)
	}
	changed := got.AfterSuccess(CheckFacts{At: t0.Add(2 * time.Hour)}, time.Hour, ChangeChanged, "h2")
	if changed.BaselineHash != "h2" || !changed.LastChangeAt.Equal(t0.Add(2*time.Hour)) {
		t.Errorf("change state wrong: %+v", changed)
	}
}

func TestShouldWarnFailures(t *testing.T) {
	t.Parallel()
	if (SiteState{ConsecutiveFailures: 2}).ShouldWarnFailures(3) {
		t.Error("warned below threshold")
	}
	if !(SiteState{ConsecutiveFailures: 3}).ShouldWarnFailures(3) {
		t.Error("did not warn at threshold")
	}
	if (SiteState{ConsecutiveFailures: 5, FailureWarned: true}).ShouldWarnFailures(3) {
		t.Error("warned twice")
	}
}

func TestWithoutBaseline(t *testing.T) {
	t.Parallel()
	s := SiteState{BaselineHash: "h", BaselineSnapshotID: 2, PendingChangeID: 1, ETag: "e",
		LastOutcome: OutcomeFailure, ConsecutiveFailures: 2, LastChangeAt: t0}
	got := s.WithoutBaseline(t0)
	if got.HasBaseline() || got.HasPendingAlert() || got.ETag != "" || got.Light() != LightGrey ||
		got.ConsecutiveFailures != 0 || !got.NextDueAt.Equal(t0) || !got.LastChangeAt.Equal(t0) {
		t.Errorf("reset wrong: %+v", got)
	}
}
