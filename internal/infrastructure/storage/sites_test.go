package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"webvigil/internal/application"
	"webvigil/internal/domain"
)

var (
	bg = context.Background()
	t0 = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
)

func openTemp(t *testing.T, r Retention) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "webvigil.db")
	s, err := Open(bg, path, r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}

func richSite() domain.Site {
	s := domain.NewSite("Shop", "https://example.com/item")
	s.Mode = domain.ModeFields
	s.Extraction.Fields = []domain.FieldRule{{Name: "Cover", Selector: "#img", Attribute: "src", Optional: true}}
	s.Extraction.Selectors = []string{"main"}
	s.Ignore = domain.IgnoreRules{ExcludeSelectors: []string{".ad"}, ExcludePatterns: []string{`\d+`}, CaseInsensitive: true}
	s.Notify = domain.NotificationPrefs{Sound: false, Visual: true, RequireAck: true}
	s.Interval, s.Timeout, s.FailureThreshold = 2*time.Hour, 45*time.Second, 5
	return s
}

func TestSiteRoundTrip(t *testing.T) {
	t.Parallel()
	s, _ := openTemp(t, DefaultRetention)
	want := richSite()
	id, err := s.AddSite(bg, want, domain.SiteState{NextDueAt: t0})
	if err != nil {
		t.Fatal(err)
	}
	want.ID = id
	got, err := s.GetSite(bg, id)
	if err != nil || got.BaselineKey() != want.BaselineKey() || got.Name != want.Name || got.Notify != want.Notify ||
		got.Interval != want.Interval || got.Timeout != want.Timeout || got.FailureThreshold != 5 || !got.Enabled {
		t.Errorf("got %+v, %v", got, err)
	}
	st, err := s.GetState(bg, id)
	if err != nil || !st.NextDueAt.Equal(t0) || st.SiteID != id {
		t.Errorf("state %+v, %v", st, err)
	}
	got.Name = "Renamed"
	st.BaselineHash = "h"
	if err := s.UpdateSite(bg, got, st); err != nil {
		t.Fatal(err)
	}
	list, _ := s.ListSites(bg)
	states, _ := s.ListStates(bg)
	if len(list) != 1 || list[0].Name != "Renamed" || len(states) != 1 || states[0].BaselineHash != "h" {
		t.Errorf("after update: %+v %+v", list, states)
	}
}

func TestNotFound(t *testing.T) {
	t.Parallel()
	s, _ := openTemp(t, DefaultRetention)
	checks := map[string]error{}
	_, checks["GetSite"] = s.GetSite(bg, 9)
	_, checks["GetState"] = s.GetState(bg, 9)
	_, checks["GetChange"] = s.GetChange(bg, 9)
	_, checks["GetSnapshot"] = s.GetSnapshot(bg, 9)
	checks["DeleteSite"] = s.DeleteSite(bg, 9)
	checks["UpdateSite"] = s.UpdateSite(bg, domain.Site{ID: 9, Mode: domain.ModeText}, domain.SiteState{})
	for name, err := range checks {
		if !errors.Is(err, application.ErrNotFound) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestDeleteSiteRemovesEverything(t *testing.T) {
	t.Parallel()
	s, _ := openTemp(t, DefaultRetention)
	id, _ := s.AddSite(bg, richSite(), domain.SiteState{})
	other, _ := s.AddSite(bg, domain.NewSite("Other", "https://other.example/"), domain.SiteState{})
	record(t, s, id, "a", nil)
	record(t, s, id, "b", &domain.ChangeEvent{SiteID: id})
	record(t, s, other, "x", nil)
	if err := s.DeleteSite(bg, id); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"site_state", "check_history", "snapshots", "change_events"} {
		var n int
		_ = s.db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE site_id = ?", id).Scan(&n)
		if n != 0 {
			t.Errorf("%s still holds %d rows", table, n)
		}
	}
	if h, _ := s.History(bg, other, 10); len(h) != 1 {
		t.Error("deleting one site touched another")
	}
}

func TestImportIsAtomic(t *testing.T) {
	t.Parallel()
	s, _ := openTemp(t, DefaultRetention)
	keep, _ := s.AddSite(bg, domain.NewSite("Keep", "https://keep.example/"), domain.SiteState{})
	bad := []application.ImportWrite{
		{Site: domain.NewSite("New", "https://new.example/")},
		{Site: domain.Site{ID: 999, Name: "Ghost", Mode: domain.ModeText}},
	}
	if err := s.ApplyImport(bg, bad); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("failing import = %v", err)
	}
	if list, _ := s.ListSites(bg); len(list) != 1 {
		t.Errorf("a failed import left %d sites", len(list))
	}
	upd := domain.NewSite("Kept and updated", "https://keep.example/")
	upd.ID = keep
	good := []application.ImportWrite{{Site: domain.NewSite("New", "https://new.example/")}, {Site: upd}}
	if err := s.ApplyImport(bg, good); err != nil {
		t.Fatal(err)
	}
	list, _ := s.ListSites(bg)
	if len(list) != 2 || list[0].Name != "Kept and updated" {
		t.Errorf("import result: %+v", list)
	}
}

func TestReopenKeepsState(t *testing.T) {
	t.Parallel()
	s, path := openTemp(t, DefaultRetention)
	id, _ := s.AddSite(bg, richSite(), domain.SiteState{})
	record(t, s, id, "a", nil)
	ids := record(t, s, id, "b", &domain.ChangeEvent{SiteID: id, Summary: "sum"})
	set := domain.Settings{Theme: domain.ThemeDark, Paused: true, SoundPath: `C:\a.wav`, RepeatAlarm: true}
	if err := s.SaveSettings(bg, set); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	again, err := Open(bg, path, DefaultRetention)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	st, _ := again.GetState(bg, id)
	if st.PendingChangeID != ids.ChangeID || st.BaselineSnapshotID != ids.SnapshotID || st.BaselineHash != "hash-b" {
		t.Errorf("state after reopen: %+v", st)
	}
	if got, _ := again.LoadSettings(bg); got != set {
		t.Errorf("settings after reopen: %+v", got)
	}
}

func TestSettingsDefaultsAndBadTheme(t *testing.T) {
	t.Parallel()
	s, _ := openTemp(t, DefaultRetention)
	if got, err := s.LoadSettings(bg); err != nil || got != domain.DefaultSettings() {
		t.Errorf("first run settings: %+v %v", got, err)
	}
	_, _ = s.db.Exec("INSERT OR REPLACE INTO application_settings (key, value) VALUES ('settings', '{\"theme\":\"neon\"}')")
	if got, _ := s.LoadSettings(bg); got.Theme != domain.ThemeSystem {
		t.Errorf("unknown stored theme = %q", got.Theme)
	}
	_, _ = s.db.Exec("UPDATE application_settings SET value = 'not json'")
	if _, err := s.LoadSettings(bg); err == nil {
		t.Error("corrupt settings not reported")
	}
}

func TestRefusesNewerSchema(t *testing.T) {
	t.Parallel()
	s, path := openTemp(t, DefaultRetention)
	_, _ = s.db.Exec("PRAGMA user_version = 99")
	_ = s.Close()
	if _, err := Open(bg, path, DefaultRetention); err == nil {
		t.Error("a newer database was opened")
	}
}

func TestCorruptRulesAreReported(t *testing.T) {
	t.Parallel()
	s, _ := openTemp(t, DefaultRetention)
	id, _ := s.AddSite(bg, richSite(), domain.SiteState{})
	_, _ = s.db.Exec("UPDATE sites SET extraction_json = 'x' WHERE id = ?", id)
	if _, err := s.GetSite(bg, id); err == nil {
		t.Error("corrupt extraction rules not reported")
	}
	_, _ = s.db.Exec("UPDATE sites SET extraction_json = '{}', ignore_json = 'x' WHERE id = ?", id)
	if _, err := s.ListSites(bg); err == nil {
		t.Error("corrupt ignore rules not reported")
	}
}
