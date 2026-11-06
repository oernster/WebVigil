package application

import (
	"context"
	"fmt"

	"webvigil/internal/domain"
)

// canceller is the part of Engine the site service needs.
type canceller interface {
	Cancel(id domain.SiteID)
	Running(id domain.SiteID) bool
}

// Sites manages definitions (FR-001 to FR-009).
type Sites struct {
	store     SiteStore
	extractor Extractor
	clock     Clock
	engine    canceller
	events    Events
}

// NewSites wires the site service.
func NewSites(store SiteStore, extractor Extractor, clock Clock, engine canceller, events Events) *Sites {
	return &Sites{store: store, extractor: extractor, clock: clock, engine: engine, events: events}
}

// SiteView is one row of the main table: the definition, its state and whether it is running.
type SiteView struct {
	Site    domain.Site
	State   domain.SiteState
	Light   domain.Light
	Running bool
}

// List returns every site with its state.
func (s *Sites) List(ctx context.Context) ([]SiteView, error) {
	sites, err := s.store.ListSites(ctx)
	if err != nil {
		return nil, err
	}
	states, err := s.store.ListStates(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[domain.SiteID]domain.SiteState, len(states))
	for _, st := range states {
		byID[st.SiteID] = st
	}
	out := make([]SiteView, 0, len(sites))
	for _, site := range sites {
		st := byID[site.ID]
		out = append(out, SiteView{Site: site, State: st, Light: st.Light(), Running: s.engine.Running(site.ID)})
	}
	return out, nil
}

// prepare normalises and validates a definition, then refuses a duplicate URL unless allowed.
func (s *Sites) prepare(ctx context.Context, site domain.Site, allowDuplicate bool) (domain.Site, error) {
	site, err := site.Normalised()
	if err != nil {
		return domain.Site{}, err
	}
	if err := s.extractor.Validate(site); err != nil {
		return domain.Site{}, err
	}
	if allowDuplicate {
		return site, nil
	}
	existing, err := s.store.ListSites(ctx)
	if err != nil {
		return domain.Site{}, err
	}
	for _, e := range existing {
		if e.URL == site.URL && e.ID != site.ID {
			return domain.Site{}, fmt.Errorf("%w: %q", ErrDuplicate, e.Name)
		}
	}
	return site, nil
}

// Add stores a new definition and makes its first check due now (FR-001, FR-004).
func (s *Sites) Add(ctx context.Context, site domain.Site, allowDuplicate bool) (domain.SiteID, error) {
	site.ID = 0
	site, err := s.prepare(ctx, site, allowDuplicate)
	if err != nil {
		return 0, err
	}
	id, err := s.store.AddSite(ctx, site, domain.SiteState{NextDueAt: s.clock.Now()})
	if err != nil {
		return 0, err
	}
	s.events.Emit(EventSitesChanged, id)
	return id, nil
}

// Update replaces a definition. A change to the baseline key is refused with
// ErrRebaselineRequired unless rebaseline is set; with it, the baseline is cleared so the next
// check establishes a new one without an alarm (FR-005, FR-006).
func (s *Sites) Update(ctx context.Context, site domain.Site, rebaseline, allowDuplicate bool) error {
	old, err := s.store.GetSite(ctx, site.ID)
	if err != nil {
		return err
	}
	site, err = s.prepare(ctx, site, allowDuplicate)
	if err != nil {
		return err
	}
	state, err := s.store.GetState(ctx, site.ID)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	if old.BaselineKey() != site.BaselineKey() {
		if !rebaseline {
			return ErrRebaselineRequired
		}
		s.engine.Cancel(site.ID)
		state = state.WithoutBaseline(now)
	} else if site.Interval != old.Interval && !state.LastCheckedAt.IsZero() {
		state.NextDueAt = domain.NextDue(state.LastCheckedAt, site.Interval, state.ConsecutiveFailures, 0)
	}
	if err := s.store.UpdateSite(ctx, site, state); err != nil {
		return err
	}
	s.events.Emit(EventSitesChanged, site.ID)
	return nil
}

// Delete stops any check of the site, then removes it with all its history (FR-007).
func (s *Sites) Delete(ctx context.Context, id domain.SiteID) error {
	s.engine.Cancel(id)
	if err := s.store.DeleteSite(ctx, id); err != nil {
		return err
	}
	s.events.Emit(EventSitesChanged, id)
	return nil
}

// SetEnabled turns scheduled monitoring of one site on or off (FR-009). Turning it on makes it
// due now when it has never been checked.
func (s *Sites) SetEnabled(ctx context.Context, id domain.SiteID, enabled bool) error {
	site, err := s.store.GetSite(ctx, id)
	if err != nil {
		return err
	}
	state, err := s.store.GetState(ctx, id)
	if err != nil {
		return err
	}
	site.Enabled = enabled
	if enabled && state.LastCheckedAt.IsZero() {
		state.NextDueAt = s.clock.Now()
	}
	if err := s.store.UpdateSite(ctx, site, state); err != nil {
		return err
	}
	s.events.Emit(EventSitesChanged, id)
	return nil
}

// Get returns one definition.
func (s *Sites) Get(ctx context.Context, id domain.SiteID) (domain.Site, error) {
	return s.store.GetSite(ctx, id)
}
