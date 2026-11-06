package storage

import (
	"fmt"
	"testing"
	"time"

	"webvigil/internal/application"
	"webvigil/internal/domain"
)

// record writes one successful check of content, as a change when change is given.
func record(t *testing.T, s *Store, id domain.SiteID, content string, change *domain.ChangeEvent) application.CheckIDs {
	t.Helper()
	st, _ := s.GetState(bg, id)
	w := application.CheckWrite{
		Record:   domain.CheckRecord{SiteID: id, At: t0, Outcome: domain.OutcomeSuccess, Duration: 1500 * time.Millisecond, Manual: true},
		Snapshot: &domain.Snapshot{SiteID: id, Hash: "hash-" + content, Content: domain.Content{Text: content, Fields: []domain.FieldValue{{Name: "t", Value: content}}}, CapturedAt: t0},
	}
	if change != nil {
		change.PrevSnapshotID, change.PrevHash, change.NewHash, change.DetectedAt = st.BaselineSnapshotID, st.BaselineHash, "hash-"+content, t0
		w.Change, w.SetPending = change, true
	}
	st.BaselineHash = "hash-" + content
	w.State = st
	ids, err := s.RecordCheck(bg, w)
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

func TestRecordCheckWritesAll(t *testing.T) {
	t.Parallel()
	s, _ := openTemp(t, DefaultRetention)
	id, _ := s.AddSite(bg, richSite(), domain.SiteState{})
	first := record(t, s, id, "a", nil)
	ids := record(t, s, id, "b", &domain.ChangeEvent{SiteID: id, Summary: "1 line added"})
	ch, err := s.GetChange(bg, ids.ChangeID)
	if err != nil || ch.PrevSnapshotID != first.SnapshotID || ch.NewSnapshotID != ids.SnapshotID || ch.Summary != "1 line added" ||
		!ch.DetectedAt.Equal(t0) || !ch.AcknowledgedAt.IsZero() {
		t.Errorf("change = %+v %v", ch, err)
	}
	snap, err := s.GetSnapshot(bg, ids.SnapshotID)
	if err != nil || snap.Content.Text != "b" || snap.Content.Fields[0].Value != "b" || snap.Hash != "hash-b" {
		t.Errorf("snapshot = %+v %v", snap, err)
	}
	h, err := s.History(bg, id, 10)
	if err != nil || len(h) != 2 || h[0].Duration != 1500*time.Millisecond || !h[0].Manual || h[0].Outcome != domain.OutcomeSuccess {
		t.Errorf("history = %+v %v", h, err)
	}
	if n, _ := s.PendingCount(bg); n != 1 {
		t.Errorf("pending = %d", n)
	}
	changes, _ := s.Changes(bg, id, 10)
	if len(changes) != 1 {
		t.Errorf("changes = %d", len(changes))
	}
	if n, err := s.Acknowledge(bg, id, t0.Add(time.Hour)); n != 1 || err != nil {
		t.Errorf("acknowledge = %d %v", n, err)
	}
	ch, _ = s.GetChange(bg, ids.ChangeID)
	st, _ := s.GetState(bg, id)
	if ch.AcknowledgedAt.IsZero() || st.HasPendingAlert() || st.BaselineHash != "hash-b" {
		t.Errorf("after acknowledge: %+v %+v", ch, st)
	}
	if n, _ := s.PendingCount(bg); n != 0 {
		t.Errorf("pending after acknowledge = %d", n)
	}
}

func TestRetention(t *testing.T) {
	t.Parallel()
	const keep = 3
	s, _ := openTemp(t, Retention{Checks: keep, Snapshots: keep})
	id, _ := s.AddSite(bg, richSite(), domain.SiteState{})
	record(t, s, id, "v0", nil)
	var last application.CheckIDs
	for i := 1; i <= 8; i++ {
		last = record(t, s, id, fmt.Sprint("v", i), &domain.ChangeEvent{SiteID: id})
		if i == 2 { // leave the first two changes pending, then acknowledge everything up to here
			continue
		}
		_, _ = s.Acknowledge(bg, id, t0)
	}
	count := func(table string) int {
		var n int
		_ = s.db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE site_id = ?", id).Scan(&n)
		return n
	}
	if count("check_history") != keep {
		t.Errorf("checks kept = %d", count("check_history"))
	}
	if count("change_events") != keep {
		t.Errorf("changes kept = %d", count("change_events"))
	}
	if _, err := s.GetSnapshot(bg, last.SnapshotID); err != nil {
		t.Error("baseline snapshot pruned")
	}
	changes, _ := s.Changes(bg, id, 10)
	for _, c := range changes {
		for _, sid := range []int64{c.PrevSnapshotID, c.NewSnapshotID} {
			if _, err := s.GetSnapshot(bg, sid); err != nil {
				t.Errorf("snapshot %d of kept change %d pruned", sid, c.ID)
			}
		}
	}
	if n := count("snapshots"); n > keep+1 {
		t.Errorf("snapshots kept = %d", n)
	}
}

func TestPendingChangesSurviveRetention(t *testing.T) {
	t.Parallel()
	s, _ := openTemp(t, Retention{Checks: 2, Snapshots: 2})
	id, _ := s.AddSite(bg, richSite(), domain.SiteState{})
	record(t, s, id, "v0", nil)
	for i := 1; i <= 5; i++ {
		record(t, s, id, fmt.Sprint("v", i), &domain.ChangeEvent{SiteID: id})
	}
	if changes, _ := s.Changes(bg, id, 10); len(changes) != 5 {
		t.Errorf("unacknowledged changes pruned: %d left", len(changes))
	}
}
