package application

import (
	"context"
	"sync"
	"time"

	"webvigil/internal/domain"
)

// Scheduling constants (REQUIREMENTS.md FR-083, FR-084).
const (
	SchedulerTick = 5 * time.Second
	HostSpacing   = 10 * time.Second
)

// starter is the part of Engine the scheduler drives.
type starter interface {
	Start(id domain.SiteID, manual bool) error
	Running(id domain.SiteID) bool
}

// Scheduler starts due checks. Each site's next due time is measured from its last check, so
// after a sleep or a long gap every overdue site is checked once, never as a backlog (FR-086).
type Scheduler struct {
	store    SiteStore
	engine   starter
	settings settingsReader
	clock    Clock
	log      Logger
	mu       sync.Mutex
	lastHost map[string]time.Time
}

// NewScheduler wires the scheduler.
func NewScheduler(store SiteStore, engine starter, settings settingsReader, clock Clock, log Logger) *Scheduler {
	return &Scheduler{store: store, engine: engine, settings: settings, clock: clock, log: log,
		lastHost: map[string]time.Time{}}
}

// Tick starts every enabled site that is due, unless monitoring is paused, the site is already
// running or its host was asked within HostSpacing. It returns how many it started.
func (s *Scheduler) Tick(ctx context.Context) (int, error) {
	if s.settings.Current().Paused {
		return 0, nil
	}
	sites, err := s.store.ListSites(ctx)
	if err != nil {
		return 0, err
	}
	states, err := s.store.ListStates(ctx)
	if err != nil {
		return 0, err
	}
	due := make(map[domain.SiteID]time.Time, len(states))
	for _, st := range states {
		due[st.SiteID] = st.NextDueAt
	}
	now := s.clock.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	started := 0
	for _, site := range sites {
		if !site.Enabled || due[site.ID].After(now) || s.engine.Running(site.ID) {
			continue
		}
		host := domain.HostOf(site.URL)
		if last, ok := s.lastHost[host]; ok && now.Sub(last) < HostSpacing {
			continue
		}
		if err := s.engine.Start(site.ID, false); err != nil {
			continue
		}
		s.lastHost[host] = now
		started++
	}
	return started, nil
}

// Run ticks every interval until ctx ends, starting with an immediate tick.
func (s *Scheduler) Run(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		if _, err := s.Tick(ctx); err != nil {
			s.log.Printf("scheduler: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RefreshAll starts a manual check of every enabled site, skipping any already running. It works
// while monitoring is paused (FR-087).
func (s *Scheduler) RefreshAll(ctx context.Context) (int, error) {
	sites, err := s.store.ListSites(ctx)
	if err != nil {
		return 0, err
	}
	started := 0
	for _, site := range sites {
		if site.Enabled && s.engine.Start(site.ID, true) == nil {
			started++
		}
	}
	return started, nil
}
