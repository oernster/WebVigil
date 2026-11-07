package application

import (
	"context"
	"sync"

	"webvigil/internal/domain"
)

// Settings holds the application preferences in memory and persists every change.
type Settings struct {
	storeImpl SettingsStore
	autostart AutoStart
	events    Events
	mu        sync.Mutex
	cur       domain.Settings
}

// NewSettings loads the stored settings. The sign-in choice is taken from the entry itself when it
// can be read, since the setup program writes that entry without touching the stored settings; an
// unreadable entry leaves the stored choice in force (FR-112).
func NewSettings(ctx context.Context, store SettingsStore, autostart AutoStart, events Events) (*Settings, error) {
	cur, err := store.LoadSettings(ctx)
	if err != nil {
		return nil, err
	}
	if on, err := autostart.Enabled(); err == nil {
		cur.StartAtSignIn = on
	}
	return &Settings{storeImpl: store, autostart: autostart, events: events, cur: cur}, nil
}

// Current returns the settings in force.
func (s *Settings) Current() domain.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cur
}

// Save validates and stores new settings, writing the sign-in entry when that choice changed. The
// change is announced only after the lock is released: listeners read the settings back (the tray
// tooltip does); announcing under the lock deadlocked the first save and every call after it.
func (s *Settings) Save(ctx context.Context, next domain.Settings) error {
	if err := next.Validate(); err != nil {
		return err
	}
	if err := s.store(ctx, next); err != nil {
		return err
	}
	s.events.Emit(EventSettings, 0)
	return nil
}

func (s *Settings) store(ctx context.Context, next domain.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if next.StartAtSignIn != s.cur.StartAtSignIn {
		if err := s.autostart.Set(next.StartAtSignIn); err != nil {
			return err
		}
	}
	if err := s.storeImpl.SaveSettings(ctx, next); err != nil {
		return err
	}
	s.cur = next
	return nil
}

// SetPaused pauses or resumes all scheduled monitoring (FR-087).
func (s *Settings) SetPaused(ctx context.Context, paused bool) error {
	next := s.Current()
	next.Paused = paused
	return s.Save(ctx, next)
}

// SetTheme stores the appearance preference (FR-120).
func (s *Settings) SetTheme(ctx context.Context, theme domain.Theme) error {
	next := s.Current()
	next.Theme = theme
	return s.Save(ctx, next)
}
