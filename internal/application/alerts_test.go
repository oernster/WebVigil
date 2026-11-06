package application

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"webvigil/internal/domain"
)

type blockingSounder struct {
	gate    chan struct{}
	plays   atomic.Int32
	paths   sync.Map
	failFor string
	done    chan struct{}
}

func newBlockingSounder() *blockingSounder {
	return &blockingSounder{gate: make(chan struct{}), done: make(chan struct{}, 16)}
}

func (s *blockingSounder) Play(path string) error {
	n := s.plays.Add(1)
	s.paths.Store(n, path)
	defer func() { s.done <- struct{}{} }()
	<-s.gate
	if path != "" && path == s.failFor {
		return errBoom
	}
	return nil
}

type noteLog struct {
	mu       sync.Mutex
	changes  []ChangeNotice
	failures []FailureNotice
}

func (n *noteLog) NotifyChange(c ChangeNotice) {
	n.mu.Lock()
	n.changes = append(n.changes, c)
	n.mu.Unlock()
}
func (n *noteLog) NotifyFailures(f FailureNotice) {
	n.mu.Lock()
	n.failures = append(n.failures, f)
	n.mu.Unlock()
}

type fakeTimer struct{ stopped bool }

func (f *fakeTimer) Stop() bool { f.stopped = true; return true }

type timerLog struct {
	mu     sync.Mutex
	timers []*fakeTimer
	fns    []func()
}

func (l *timerLog) after(_ time.Duration, f func()) Timer {
	l.mu.Lock()
	defer l.mu.Unlock()
	t := &fakeTimer{}
	l.timers, l.fns = append(l.timers, t), append(l.fns, f)
	return t
}

func (l *timerLog) count() int { l.mu.Lock(); defer l.mu.Unlock(); return len(l.timers) }

type alertsRig struct {
	store  *memStore
	sound  *blockingSounder
	notes  *noteLog
	timers *timerLog
	set    *fixedSettings
	log    *logSink
	alerts *Alerts
}

func newAlertsRig() *alertsRig {
	r := &alertsRig{store: newMemStore(), sound: newBlockingSounder(), notes: &noteLog{}, timers: &timerLog{},
		set: &fixedSettings{s: domain.DefaultSettings()}, log: &logSink{}}
	r.alerts = NewAlerts(AlertsDeps{History: r.store, Sounder: r.sound, Notifier: r.notes, Settings: r.set,
		Clock: &fakeClock{now: t0}, Events: &eventLog{}, Log: r.log, After: r.timers.after})
	return r
}

func (r *alertsRig) waitPlays(t *testing.T, n int) {
	t.Helper()
	for range n {
		select {
		case <-r.sound.done:
		case <-time.After(5 * time.Second):
			t.Fatal("sound did not finish")
		}
	}
}

// settle waits until the alarm has finished its bookkeeping after the last play.
func (r *alertsRig) settle() {
	for range 1000 {
		r.alerts.mu.Lock()
		playing := r.alerts.playing
		r.alerts.mu.Unlock()
		if !playing {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

var site = domain.Site{ID: 1, Name: "Shop", URL: siteURL, Notify: domain.NotificationPrefs{Sound: true, Visual: true}}

func TestAlarmsDoNotOverlap(t *testing.T) {
	t.Parallel()
	r := newAlertsRig()
	r.alerts.ChangeDetected(site, domain.ChangeEvent{ID: 5, Summary: "s", DetectedAt: t0})
	r.alerts.ChangeDetected(site, domain.ChangeEvent{ID: 6})
	r.alerts.Ring()
	close(r.sound.gate)
	r.waitPlays(t, 1)
	r.settle()
	if r.sound.plays.Load() != 1 || len(r.notes.changes) != 2 || r.notes.changes[0].ChangeID != 5 || r.notes.changes[0].Summary != "s" {
		t.Errorf("plays = %d, notes = %+v", r.sound.plays.Load(), r.notes.changes)
	}
}

func TestPreferencesAreHonoured(t *testing.T) {
	t.Parallel()
	r := newAlertsRig()
	quiet := site
	quiet.Notify = domain.NotificationPrefs{}
	r.alerts.ChangeDetected(quiet, domain.ChangeEvent{})
	r.alerts.FailuresRepeated(quiet, domain.SiteState{})
	if r.sound.plays.Load() != 0 || len(r.notes.changes) != 0 || len(r.notes.failures) != 0 {
		t.Error("a silent site alerted")
	}
	r.alerts.FailuresRepeated(site, domain.SiteState{ConsecutiveFailures: 3, LastDetail: "d"})
	if len(r.notes.failures) != 1 || r.notes.failures[0].Failures != 3 || r.sound.plays.Load() != 0 {
		t.Errorf("failure warning = %+v", r.notes.failures)
	}
}

func TestRepeatStopsOnAcknowledge(t *testing.T) {
	t.Parallel()
	r := newAlertsRig()
	r.set.s.RepeatAlarm = true
	r.store.states[1] = domain.SiteState{SiteID: 1, PendingChangeID: 9}
	r.store.changes[9] = domain.ChangeEvent{ID: 9, SiteID: 1}
	close(r.sound.gate)
	r.alerts.Ring()
	r.waitPlays(t, 1)
	r.settle()
	if r.timers.count() != 1 {
		t.Fatalf("no repeat scheduled: %d", r.timers.count())
	}
	r.timers.fns[0]() // the repeat fires
	r.waitPlays(t, 1)
	r.settle()
	if err := r.alerts.Acknowledge(bg, 1); err != nil {
		t.Fatal(err)
	}
	if !r.timers.timers[1].stopped || r.store.states[1].HasPendingAlert() || r.store.changes[9].AcknowledgedAt.IsZero() {
		t.Error("acknowledge did not stop the repeat or keep the record")
	}
}

func TestAcknowledgeKeepsHistory(t *testing.T) {
	t.Parallel()
	r := newAlertsRig()
	r.store.states[1] = domain.SiteState{SiteID: 1, PendingChangeID: 9, BaselineHash: "h", BaselineSnapshotID: 3}
	r.store.states[2] = domain.SiteState{SiteID: 2, PendingChangeID: 10}
	r.store.changes[9] = domain.ChangeEvent{ID: 9, SiteID: 1}
	if err := r.alerts.Acknowledge(bg, 1); err != nil {
		t.Fatal(err)
	}
	st := r.store.states[1]
	if st.HasPendingAlert() || st.BaselineHash != "h" || st.BaselineSnapshotID != 3 || len(r.store.changes) != 1 ||
		r.store.changes[9].AcknowledgedAt != t0 {
		t.Errorf("acknowledge = %+v", st)
	}
	r.alerts.StopRepeat() // nothing scheduled: no effect
	for _, m := range []string{"Acknowledge", "PendingCount"} {
		r.store.fail = map[string]bool{m: true}
		if err := r.alerts.Acknowledge(bg, 1); !errors.Is(err, errBoom) {
			t.Errorf("%s: %v", m, err)
		}
	}
}

func TestCustomSoundFallsBackToChime(t *testing.T) {
	t.Parallel()
	r := newAlertsRig()
	r.set.s.SoundPath = "missing.wav"
	r.sound.failFor = "missing.wav"
	r.store.fail["PendingCount"] = true
	close(r.sound.gate)
	r.alerts.Ring()
	r.waitPlays(t, 2)
	r.settle()
	if p, _ := r.sound.paths.Load(int32(2)); p != "" || r.log.count() != 2 {
		t.Errorf("fallback path = %v, log = %v", p, r.log.lines)
	}
	if err := r.alerts.TestSound("missing.wav"); !errors.Is(err, errBoom) {
		t.Errorf("TestSound = %v", err)
	}
}

type panicSounder struct{}

func (panicSounder) Play(string) error { panic("speaker on fire") }

func TestAlarmPanicIsReported(t *testing.T) {
	t.Parallel()
	r := newAlertsRig()
	r.alerts.sounder = panicSounder{}
	r.alerts.Ring()
	for range 1000 {
		if r.log.count() > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	r.settle()
	if r.log.count() != 1 {
		t.Errorf("panic not logged: %v", r.log.lines)
	}
}
