package application

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"webvigil/internal/domain"
)

// gateRunner blocks every check until released, counting how many run at once.
type gateRunner struct {
	release chan struct{}
	started chan domain.SiteID
	now     atomic.Int32
	peak    atomic.Int32
	manual  sync.Map
	panicID domain.SiteID
	err     error
}

func newGateRunner() *gateRunner {
	return &gateRunner{release: make(chan struct{}), started: make(chan domain.SiteID, 64)}
}

func (g *gateRunner) Check(ctx context.Context, id domain.SiteID, manual bool) (CheckResult, error) {
	n := g.now.Add(1)
	defer g.now.Add(-1)
	for {
		p := g.peak.Load()
		if n <= p || g.peak.CompareAndSwap(p, n) {
			break
		}
	}
	g.manual.Store(id, manual)
	g.started <- id
	if id == g.panicID {
		panic("kaboom")
	}
	select {
	case <-g.release:
	case <-ctx.Done():
		return CheckResult{}, ErrCancelled
	}
	return CheckResult{}, g.err
}

func waitStarted(t *testing.T, g *gateRunner, n int) {
	t.Helper()
	for range n {
		select {
		case <-g.started:
		case <-time.After(5 * time.Second):
			t.Fatal("check did not start")
		}
	}
}

// waitIdle waits until the engine has finished every check of id.
func waitIdle(t *testing.T, e *Engine, id domain.SiteID) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for e.Running(id) {
		if time.Now().After(deadline) {
			t.Fatal("check did not finish")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestNoOverlappingChecks(t *testing.T) {
	t.Parallel()
	g := newGateRunner()
	e := NewEngine(g, &eventLog{}, &logSink{})
	defer e.Stop()
	if err := e.Start(1, true); err != nil {
		t.Fatal(err)
	}
	if err := e.Start(1, false); !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("second start = %v", err)
	}
	if !e.Running(1) || e.Running(2) {
		t.Error("Running wrong")
	}
	waitStarted(t, g, 1)
	close(g.release)
}

func TestConcurrencyLimit(t *testing.T) {
	t.Parallel()
	g := newGateRunner()
	e := NewEngine(g, &eventLog{}, &logSink{})
	for id := domain.SiteID(1); id <= MaxConcurrentChecks+3; id++ {
		if err := e.Start(id, false); err != nil {
			t.Fatal(err)
		}
	}
	waitStarted(t, g, MaxConcurrentChecks)
	select {
	case <-g.started:
		t.Fatal("more than the limit started")
	case <-time.After(50 * time.Millisecond):
	}
	close(g.release)
	waitStarted(t, g, 3)
	e.Stop()
	if g.peak.Load() != MaxConcurrentChecks {
		t.Errorf("peak = %d", g.peak.Load())
	}
}

func TestDeleteCancelsRunningCheck(t *testing.T) {
	t.Parallel()
	g := newGateRunner()
	ev := &eventLog{}
	e := NewEngine(g, ev, &logSink{})
	defer e.Stop()
	_ = e.Start(7, false)
	waitStarted(t, g, 1)
	e.Cancel(7)
	if e.Running(7) {
		t.Error("still running after Cancel")
	}
	e.Cancel(8) // nothing running: returns at once
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if len(ev.events) != 2 || ev.events[0] != EventCheckStarted || ev.events[1] != EventCheckFinished {
		t.Errorf("events = %v", ev.events)
	}
}

func TestStopCancelsAndWaits(t *testing.T) {
	t.Parallel()
	g := newGateRunner()
	e := NewEngine(g, &eventLog{}, &logSink{})
	_ = e.Start(1, false)
	waitStarted(t, g, 1)
	for id := domain.SiteID(2); id <= MaxConcurrentChecks+2; id++ {
		_ = e.Start(id, false) // some wait for a slot and are cancelled while waiting
	}
	e.Stop()
	if g.now.Load() != 0 || e.Running(1) {
		t.Error("checks still running after Stop")
	}
	if err := e.Start(9, false); !errors.Is(err, ErrStopped) {
		t.Errorf("start after stop = %v", err)
	}
}

func TestCancelWhileWaitingForSlot(t *testing.T) {
	t.Parallel()
	g := newGateRunner()
	ev := &eventLog{}
	e := NewEngine(g, ev, &logSink{})
	for id := domain.SiteID(1); id <= MaxConcurrentChecks; id++ {
		_ = e.Start(id, false)
	}
	waitStarted(t, g, MaxConcurrentChecks)
	queued := domain.SiteID(MaxConcurrentChecks + 1)
	_ = e.Start(queued, false)
	for {
		ev.mu.Lock()
		n := len(ev.events)
		ev.mu.Unlock()
		if n == MaxConcurrentChecks+1 { // the queued check has announced itself and is waiting
			break
		}
		time.Sleep(time.Millisecond)
	}
	e.Cancel(queued)
	if e.Running(queued) || g.now.Load() != MaxConcurrentChecks {
		t.Error("queued check was not cancelled cleanly")
	}
	close(g.release)
	e.Stop()
}

func TestEnginePanicIsReported(t *testing.T) {
	t.Parallel()
	g := newGateRunner()
	g.panicID = 3
	log := &logSink{}
	e := NewEngine(g, &eventLog{}, log)
	_ = e.Start(3, false)
	waitStarted(t, g, 1)
	e.Stop()
	if log.count() != 1 || e.Running(3) {
		t.Errorf("panic not reported: %v", log.lines)
	}
}

func TestEngineErrorIsLogged(t *testing.T) {
	t.Parallel()
	g := newGateRunner()
	g.err = errBoom
	close(g.release)
	log := &logSink{}
	e := NewEngine(g, &eventLog{}, log)
	_ = e.Start(1, false)
	waitStarted(t, g, 1)
	waitIdle(t, e, 1)
	e.Stop()
	if log.count() != 1 {
		t.Errorf("error not logged: %v", log.lines)
	}
}

func TestManualAndScheduledShareEngine(t *testing.T) {
	t.Parallel()
	r := newCheckRig(t)
	r.fetch.set(siteURL, "a")
	e := NewEngine(r.checker, &eventLog{}, &logSink{})
	sched := NewScheduler(r.store, e, &fixedSettings{}, r.clock, &logSink{})
	if n, err := sched.Tick(context.Background()); n != 1 || err != nil {
		t.Fatalf("tick: %d %v", n, err)
	}
	waitIdle(t, e, r.id)
	if err := e.Start(r.id, true); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, e, r.id)
	e.Stop()
	if len(r.store.checks) != 2 || r.store.checks[0].Manual || !r.store.checks[1].Manual {
		t.Errorf("checks = %+v", r.store.checks)
	}
}
