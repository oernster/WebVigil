package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"webvigil/internal/domain"
)

type fakeStarter struct {
	mu      sync.Mutex
	started []domain.SiteID
	manual  []bool
	running map[domain.SiteID]bool
	err     error
}

func (f *fakeStarter) Start(id domain.SiteID, manual bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.started = append(f.started, id)
	f.manual = append(f.manual, manual)
	return nil
}

func (f *fakeStarter) Running(id domain.SiteID) bool { return f.running[id] }

type schedRig struct {
	store *memStore
	eng   *fakeStarter
	set   *fixedSettings
	clock *fakeClock
	s     *Scheduler
}

func newSchedRig() *schedRig {
	r := &schedRig{store: newMemStore(), eng: &fakeStarter{running: map[domain.SiteID]bool{}},
		set: &fixedSettings{s: domain.DefaultSettings()}, clock: &fakeClock{now: t0}}
	r.s = NewScheduler(r.store, r.eng, r.set, r.clock, &logSink{})
	return r
}

func (r *schedRig) add(url string, due time.Time, enabled bool) domain.SiteID {
	s, _ := domain.NewSite("", url).Normalised()
	s.Enabled = enabled
	id, _ := r.store.AddSite(bg, s, domain.SiteState{NextDueAt: due})
	return id
}

func TestDisabledSiteIsNotScheduled(t *testing.T) {
	t.Parallel()
	r := newSchedRig()
	r.add("https://a.example/", t0, false)
	on := r.add("https://b.example/", t0, true)
	r.add("https://c.example/", t0.Add(time.Minute), true)
	if n, err := r.s.Tick(bg); n != 1 || err != nil || r.eng.started[0] != on || r.eng.manual[0] {
		t.Errorf("tick = %d %v %v", n, err, r.eng.started)
	}
}

func TestHostSpacing(t *testing.T) {
	t.Parallel()
	r := newSchedRig()
	r.add("https://shop.example/a", t0, true)
	r.add("https://shop.example/b", t0, true)
	if n, _ := r.s.Tick(bg); n != 1 {
		t.Fatalf("first tick started %d", n)
	}
	r.clock.advance(HostSpacing - time.Second)
	r.eng.running[1] = true
	if n, _ := r.s.Tick(bg); n != 0 {
		t.Errorf("second request to the host came too soon: %d", n)
	}
	r.clock.advance(time.Second)
	if n, _ := r.s.Tick(bg); n != 1 || r.eng.started[1] != 2 {
		t.Errorf("after spacing started %d", n)
	}
}

func TestResumeRunsOverdueOnce(t *testing.T) {
	t.Parallel()
	r := newSchedRig()
	for _, u := range []string{"https://a.example/", "https://b.example/", "https://c.example/"} {
		r.add(u, t0.Add(-72*time.Hour), true)
	}
	if n, _ := r.s.Tick(bg); n != 3 {
		t.Fatalf("overdue started %d", n)
	}
	for id, st := range r.store.states { // each check completes and moves its due time
		st.NextDueAt = domain.NextDue(t0, time.Hour, 0, 0)
		r.store.states[id] = st
	}
	r.clock.advance(time.Minute)
	if n, _ := r.s.Tick(bg); n != 0 {
		t.Errorf("backlog replayed: %d", n)
	}
}

func TestPauseStopsScheduling(t *testing.T) {
	t.Parallel()
	r := newSchedRig()
	r.add("https://a.example/", t0, true)
	r.add("https://b.example/", t0, false)
	r.set.s.Paused = true
	if n, _ := r.s.Tick(bg); n != 0 {
		t.Errorf("paused tick started %d", n)
	}
	if n, err := r.s.RefreshAll(bg); n != 1 || err != nil || !r.eng.manual[0] {
		t.Errorf("refresh all while paused = %d %v", n, err)
	}
}

func TestSchedulerErrors(t *testing.T) {
	t.Parallel()
	for _, m := range []string{"ListSites", "ListStates"} {
		r := newSchedRig()
		r.store.fail[m] = true
		if _, err := r.s.Tick(bg); !errors.Is(err, errBoom) {
			t.Errorf("%s: %v", m, err)
		}
	}
	r := newSchedRig()
	r.store.fail["ListSites"] = true
	if _, err := r.s.RefreshAll(bg); !errors.Is(err, errBoom) {
		t.Errorf("refresh all: %v", err)
	}
	r = newSchedRig()
	r.add("https://a.example/", t0, true)
	r.eng.err = ErrStopped
	if n, err := r.s.Tick(bg); n != 0 || err != nil {
		t.Errorf("refused start counted: %d %v", n, err)
	}
}

func TestSchedulerRunTicksUntilCancelled(t *testing.T) {
	t.Parallel()
	r := newSchedRig()
	r.store.fail["ListSites"] = true // every tick fails and is logged
	log := &logSink{}
	r.s.log = log
	ctx, cancel := context.WithCancel(bg)
	done := make(chan struct{})
	go func() { r.s.Run(ctx, time.Millisecond); close(done) }()
	for log.count() < 2 {
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return")
	}
}
