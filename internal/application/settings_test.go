package application

import (
	"errors"
	"testing"
	"time"

	"webvigil/internal/domain"
)

type fakeAutoStart struct {
	set     []bool
	err     error
	on      bool  // what Enabled reports
	readErr error // why Enabled cannot tell
}

func (f *fakeAutoStart) Enabled() (bool, error) { return f.on, f.readErr }

func (f *fakeAutoStart) Set(enabled bool) error {
	if f.err != nil {
		return f.err
	}
	f.set = append(f.set, enabled)
	return nil
}

// TestSettingsTakeTheSignInChoiceFromTheEntry: setup writes the sign-in entry without touching the
// stored settings, so the entry decides what Settings shows; an unreadable one leaves the stored
// choice.
func TestSettingsTakeTheSignInChoiceFromTheEntry(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name          string
		stored, entry bool
		readErr       error
		want          bool
	}{
		{"setup wrote the entry", false, true, nil, true},
		{"the entry was removed", true, false, nil, false},
		{"the entry cannot be read", true, false, errBoom, true},
	} {
		store := newMemStore()
		store.settings.StartAtSignIn = c.stored
		s, err := NewSettings(bg, store, &fakeAutoStart{on: c.entry, readErr: c.readErr}, &eventLog{})
		if err != nil {
			t.Fatal(err)
		}
		if got := s.Current().StartAtSignIn; got != c.want {
			t.Errorf("%s: StartAtSignIn = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestThemePersists(t *testing.T) {
	t.Parallel()
	store := newMemStore()
	s, err := NewSettings(bg, store, &fakeAutoStart{}, &eventLog{})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetTheme(bg, domain.ThemeDark); err != nil {
		t.Fatal(err)
	}
	again, _ := NewSettings(bg, store, &fakeAutoStart{}, &eventLog{})
	if again.Current().Theme != domain.ThemeDark {
		t.Errorf("theme after reload = %q", again.Current().Theme)
	}
	if err := s.SetTheme(bg, "neon"); !errors.Is(err, domain.ErrInvalidSettings) {
		t.Errorf("bad theme: %v", err)
	}
}

// rereadingEvents reads the settings while being told they changed, as the tray tooltip does.
type rereadingEvents struct{ s *Settings }

func (e *rereadingEvents) Emit(EventKind, domain.SiteID) { _ = e.s.Current() }

func TestSaveDoesNotDeadlockWhenListenersRead(t *testing.T) {
	t.Parallel()
	ev := &rereadingEvents{}
	s, _ := NewSettings(bg, newMemStore(), &fakeAutoStart{}, ev)
	ev.s = s
	done := make(chan error, 2)
	go func() {
		done <- s.SetTheme(bg, domain.ThemeLight)
		done <- s.SetTheme(bg, domain.ThemeDark)
	}()
	for range 2 {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Save deadlocked: a listener that reads the settings never returned")
		}
	}
	if s.Current().Theme != domain.ThemeDark {
		t.Errorf("theme = %q", s.Current().Theme)
	}
}

func TestSettingsSave(t *testing.T) {
	t.Parallel()
	store := newMemStore()
	auto := &fakeAutoStart{}
	s, _ := NewSettings(bg, store, auto, &eventLog{})
	if err := s.SetPaused(bg, true); err != nil || !store.settings.Paused {
		t.Errorf("pause: %v", err)
	}
	next := s.Current()
	next.StartAtSignIn = true
	if err := s.Save(bg, next); err != nil || len(auto.set) != 1 || !auto.set[0] {
		t.Errorf("sign-in start: %v %v", err, auto.set)
	}
	if err := s.Save(bg, next); err != nil || len(auto.set) != 1 {
		t.Error("unchanged sign-in choice rewrote the entry")
	}
	auto.err = errBoom
	next.StartAtSignIn = false
	if err := s.Save(bg, next); !errors.Is(err, errBoom) || !s.Current().StartAtSignIn {
		t.Errorf("autostart failure: %v", err)
	}
	auto.err = nil
	store.fail["SaveSettings"] = true
	if err := s.Save(bg, next); !errors.Is(err, errBoom) {
		t.Errorf("store failure: %v", err)
	}
	store.fail["LoadSettings"] = true
	if _, err := NewSettings(bg, store, auto, &eventLog{}); !errors.Is(err, errBoom) {
		t.Errorf("load failure: %v", err)
	}
}
