package application

import (
	"context"
	"sync"

	"webvigil/internal/domain"
)

// MaxConcurrentChecks bounds how many checks run at once (REQUIREMENTS.md FR-082).
const MaxConcurrentChecks = 4

// checkRunner is the part of Checker the engine drives.
type checkRunner interface {
	Check(ctx context.Context, id domain.SiteID, manual bool) (CheckResult, error)
}

type running struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// Engine runs checks in the background: one at a time per site, at most MaxConcurrentChecks in
// all, each cancellable. It owns its goroutines and recovers a panic in any of them, reporting it
// rather than letting it end the application.
type Engine struct {
	checker checkRunner
	events  Events
	log     Logger
	root    context.Context
	stop    context.CancelFunc
	slots   chan struct{}
	mu      sync.Mutex
	active  map[domain.SiteID]*running
	wg      sync.WaitGroup
}

// NewEngine creates an engine whose checks live until Stop.
func NewEngine(checker checkRunner, events Events, log Logger) *Engine {
	root, stop := context.WithCancel(context.Background())
	return &Engine{
		checker: checker, events: events, log: log, root: root, stop: stop,
		slots: make(chan struct{}, MaxConcurrentChecks), active: map[domain.SiteID]*running{},
	}
}

// Start begins a check of id in the background. It refuses a site already being checked.
func (e *Engine) Start(id domain.SiteID, manual bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.root.Err() != nil {
		return ErrStopped
	}
	if _, busy := e.active[id]; busy {
		return ErrAlreadyRunning
	}
	ctx, cancel := context.WithCancel(e.root)
	r := &running{cancel: cancel, done: make(chan struct{})}
	e.active[id] = r
	e.wg.Add(1)
	go e.run(ctx, id, manual, r)
	return nil
}

func (e *Engine) run(ctx context.Context, id domain.SiteID, manual bool, r *running) {
	defer e.wg.Done()
	defer func() {
		r.cancel()
		e.mu.Lock()
		delete(e.active, id)
		e.mu.Unlock()
		e.events.Emit(EventCheckFinished, id)
		close(r.done)
	}()
	defer func() {
		if p := recover(); p != nil {
			e.log.Printf("check of site %d panicked: %v", id, p)
		}
	}()
	e.events.Emit(EventCheckStarted, id)
	select {
	case e.slots <- struct{}{}:
	case <-ctx.Done():
		return
	}
	defer func() { <-e.slots }()
	if _, err := e.checker.Check(ctx, id, manual); err != nil && err != ErrCancelled {
		e.log.Printf("check of site %d: %v", id, err)
	}
}

// Running reports whether id is being checked (or waiting for a slot).
func (e *Engine) Running(id domain.SiteID) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := e.active[id]
	return ok
}

// Cancel stops any check of id and waits for it to finish, so the caller may then remove the
// site without a check writing behind it.
func (e *Engine) Cancel(id domain.SiteID) {
	e.mu.Lock()
	r, ok := e.active[id]
	e.mu.Unlock()
	if !ok {
		return
	}
	r.cancel()
	<-r.done
}

// Stop cancels every check and waits for all of them (FR-088).
func (e *Engine) Stop() {
	e.mu.Lock()
	e.stop()
	e.mu.Unlock()
	e.wg.Wait()
}
