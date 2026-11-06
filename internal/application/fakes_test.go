package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"webvigil/internal/domain"
)

var errBoom = errors.New("boom")

var t0 = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)

// memStore is an in-memory SiteStore, HistoryStore and SettingsStore. Each fail field makes the
// named method return errBoom.
type memStore struct {
	mu        sync.Mutex
	nextID    int64
	sites     map[domain.SiteID]domain.Site
	order     []domain.SiteID
	states    map[domain.SiteID]domain.SiteState
	checks    []domain.CheckRecord
	snaps     map[int64]domain.Snapshot
	changes   map[int64]domain.ChangeEvent
	settings  domain.Settings
	fail      map[string]bool
	failAfter map[string]int // fail the named method after this many successful calls
	calls     map[string]int
}

func newMemStore() *memStore {
	return &memStore{sites: map[domain.SiteID]domain.Site{}, states: map[domain.SiteID]domain.SiteState{},
		snaps: map[int64]domain.Snapshot{}, changes: map[int64]domain.ChangeEvent{},
		settings: domain.DefaultSettings(), fail: map[string]bool{}, failAfter: map[string]int{}, calls: map[string]int{}}
}

func (m *memStore) err(name string) error {
	m.calls[name]++
	if m.fail[name] {
		return errBoom
	}
	if n, ok := m.failAfter[name]; ok && m.calls[name] > n {
		return errBoom
	}
	return nil
}

func (m *memStore) id() int64 { m.nextID++; return m.nextID }

func (m *memStore) ListSites(context.Context) ([]domain.Site, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("ListSites"); err != nil {
		return nil, err
	}
	out := make([]domain.Site, 0, len(m.order))
	for _, id := range m.order {
		out = append(out, m.sites[id])
	}
	return out, nil
}

func (m *memStore) GetSite(_ context.Context, id domain.SiteID) (domain.Site, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("GetSite"); err != nil {
		return domain.Site{}, err
	}
	s, ok := m.sites[id]
	if !ok {
		return domain.Site{}, ErrNotFound
	}
	return s, nil
}

func (m *memStore) AddSite(_ context.Context, s domain.Site, st domain.SiteState) (domain.SiteID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("AddSite"); err != nil {
		return 0, err
	}
	s.ID = domain.SiteID(m.id())
	st.SiteID = s.ID
	m.sites[s.ID], m.states[s.ID] = s, st
	m.order = append(m.order, s.ID)
	return s.ID, nil
}

func (m *memStore) UpdateSite(_ context.Context, s domain.Site, st domain.SiteState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("UpdateSite"); err != nil {
		return err
	}
	st.SiteID = s.ID
	m.sites[s.ID], m.states[s.ID] = s, st
	return nil
}

func (m *memStore) DeleteSite(_ context.Context, id domain.SiteID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("DeleteSite"); err != nil {
		return err
	}
	delete(m.sites, id)
	delete(m.states, id)
	for i, o := range m.order {
		if o == id {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	return nil
}

func (m *memStore) ListStates(context.Context) ([]domain.SiteState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("ListStates"); err != nil {
		return nil, err
	}
	out := make([]domain.SiteState, 0, len(m.states))
	for _, id := range m.order {
		out = append(out, m.states[id])
	}
	return out, nil
}

func (m *memStore) GetState(_ context.Context, id domain.SiteID) (domain.SiteState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("GetState"); err != nil {
		return domain.SiteState{}, err
	}
	return m.states[id], nil
}

func (m *memStore) ApplyImport(_ context.Context, writes []ImportWrite) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("ApplyImport"); err != nil {
		return err
	}
	for _, w := range writes {
		if w.Site.ID == 0 {
			w.Site.ID = domain.SiteID(m.id())
			m.order = append(m.order, w.Site.ID)
		}
		w.State.SiteID = w.Site.ID
		m.sites[w.Site.ID], m.states[w.Site.ID] = w.Site, w.State
	}
	return nil
}

func (m *memStore) RecordCheck(_ context.Context, w CheckWrite) (CheckIDs, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("RecordCheck"); err != nil {
		return CheckIDs{}, err
	}
	var ids CheckIDs
	if w.Snapshot != nil {
		ids.SnapshotID = m.id()
		snap := *w.Snapshot
		snap.ID = ids.SnapshotID
		m.snaps[snap.ID] = snap
		w.State.BaselineSnapshotID = snap.ID
	}
	if w.Change != nil {
		ids.ChangeID = m.id()
		ch := *w.Change
		ch.ID, ch.NewSnapshotID = ids.ChangeID, ids.SnapshotID
		m.changes[ch.ID] = ch
		if w.SetPending {
			w.State.PendingChangeID = ch.ID
		}
	}
	w.Record.ID = m.id()
	m.checks = append(m.checks, w.Record)
	m.states[w.State.SiteID] = w.State
	return ids, nil
}

func (m *memStore) Acknowledge(_ context.Context, id domain.SiteID, at time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("Acknowledge"); err != nil {
		return 0, err
	}
	n := 0
	for cid, ch := range m.changes {
		if ch.SiteID == id && ch.AcknowledgedAt.IsZero() {
			ch.AcknowledgedAt = at
			m.changes[cid] = ch
			n++
		}
	}
	st := m.states[id]
	st.PendingChangeID = 0
	m.states[id] = st
	return n, nil
}

func (m *memStore) History(_ context.Context, id domain.SiteID, limit int) ([]domain.CheckRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("History"); err != nil {
		return nil, err
	}
	var out []domain.CheckRecord
	for i := len(m.checks) - 1; i >= 0 && len(out) < limit; i-- {
		if m.checks[i].SiteID == id {
			out = append(out, m.checks[i])
		}
	}
	return out, nil
}

func (m *memStore) Changes(_ context.Context, id domain.SiteID, _ int) ([]domain.ChangeEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("Changes"); err != nil {
		return nil, err
	}
	var out []domain.ChangeEvent
	for _, ch := range m.changes {
		if ch.SiteID == id {
			out = append(out, ch)
		}
	}
	return out, nil
}

func (m *memStore) GetChange(_ context.Context, id int64) (domain.ChangeEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("GetChange"); err != nil {
		return domain.ChangeEvent{}, err
	}
	ch, ok := m.changes[id]
	if !ok {
		return domain.ChangeEvent{}, ErrNotFound
	}
	return ch, nil
}

func (m *memStore) GetSnapshot(_ context.Context, id int64) (domain.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("GetSnapshot"); err != nil {
		return domain.Snapshot{}, err
	}
	s, ok := m.snaps[id]
	if !ok {
		return domain.Snapshot{}, ErrNotFound
	}
	return s, nil
}

func (m *memStore) PendingCount(context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.err("PendingCount"); err != nil {
		return 0, err
	}
	n := 0
	for _, st := range m.states {
		if st.HasPendingAlert() {
			n++
		}
	}
	return n, nil
}

func (m *memStore) LoadSettings(context.Context) (domain.Settings, error) {
	if err := m.err("LoadSettings"); err != nil {
		return domain.Settings{}, err
	}
	return m.settings, nil
}

func (m *memStore) SaveSettings(_ context.Context, s domain.Settings) error {
	if err := m.err("SaveSettings"); err != nil {
		return err
	}
	m.settings = s
	return nil
}
