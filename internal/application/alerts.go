package application

import (
	"context"
	"sync"
	"time"

	"webvigil/internal/domain"
)

// RepeatAlarmEvery is the gap between repeats while repeat-until-acknowledged is on (FR-092).
const RepeatAlarmEvery = 30 * time.Second

// Timer is a pending call that can be stopped.
type Timer interface {
	Stop() bool
}

// AfterFunc schedules f after d; production passes a wrapper over time.AfterFunc.
type AfterFunc func(d time.Duration, f func()) Timer

// settingsReader gives the current settings.
type settingsReader interface {
	Current() domain.Settings
}

// Alerts announces changes and repeated failures, plays one alarm at a time and acknowledges
// pending changes (FR-090 to FR-095).
type Alerts struct {
	history  HistoryStore
	sounder  Sounder
	notifier Notifier
	settings settingsReader
	clock    Clock
	events   Events
	log      Logger
	after    AfterFunc

	mu      sync.Mutex
	playing bool
	repeat  Timer
}

// AlertsDeps groups the collaborators of Alerts.
type AlertsDeps struct {
	History  HistoryStore
	Sounder  Sounder
	Notifier Notifier
	Settings settingsReader
	Clock    Clock
	Events   Events
	Log      Logger
	After    AfterFunc
}

// NewAlerts wires the alert service.
func NewAlerts(d AlertsDeps) *Alerts {
	return &Alerts{history: d.History, sounder: d.Sounder, notifier: d.Notifier, settings: d.Settings,
		clock: d.Clock, events: d.Events, log: d.Log, after: d.After}
}

// ChangeDetected shows and sounds a change according to the site's preferences.
func (a *Alerts) ChangeDetected(site domain.Site, change domain.ChangeEvent) {
	if site.Notify.Visual {
		a.notifier.NotifyChange(ChangeNotice{SiteID: site.ID, ChangeID: change.ID, Name: site.Name,
			URL: site.URL, At: change.DetectedAt, Summary: change.Summary})
	}
	if site.Notify.Sound {
		a.Ring()
	}
}

// FailuresRepeated shows a warning once a run of failures reaches the site's threshold. A failure
// never sounds the change alarm (FR-094).
func (a *Alerts) FailuresRepeated(site domain.Site, state domain.SiteState) {
	if site.Notify.Visual {
		a.notifier.NotifyFailures(FailureNotice{SiteID: site.ID, Name: site.Name,
			Failures: state.ConsecutiveFailures, Detail: state.LastDetail})
	}
}

// Ring starts the alarm unless one is already sounding, so simultaneous changes never overlap.
func (a *Alerts) Ring() {
	a.mu.Lock()
	if a.playing {
		a.mu.Unlock()
		return
	}
	a.playing = true
	a.mu.Unlock()
	go a.play()
}

func (a *Alerts) play() {
	defer func() {
		if p := recover(); p != nil {
			a.log.Printf("alarm panicked: %v", p)
		}
		a.finish()
	}()
	path := a.settings.Current().SoundPath
	if err := a.sounder.Play(path); err != nil {
		a.log.Printf("alarm sound %q could not be played (%v); playing the built-in chime", path, err)
		if path != "" {
			_ = a.sounder.Play("")
		}
	}
}

func (a *Alerts) finish() {
	pending, err := a.history.PendingCount(context.Background())
	if err != nil {
		a.log.Printf("count pending alerts: %v", err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.playing = false
	if a.settings.Current().RepeatAlarm && pending > 0 {
		a.repeat = a.after(RepeatAlarmEvery, a.Ring)
	}
}

// Acknowledge stamps every pending change of the site, keeping history and baseline; it stops a
// repeating alarm once nothing is pending (FR-093).
func (a *Alerts) Acknowledge(ctx context.Context, id domain.SiteID) error {
	if _, err := a.history.Acknowledge(ctx, id, a.clock.Now()); err != nil {
		return err
	}
	pending, err := a.history.PendingCount(ctx)
	if err != nil {
		return err
	}
	if pending == 0 {
		a.StopRepeat()
	}
	a.events.Emit(EventSitesChanged, id)
	return nil
}

// StopRepeat cancels a scheduled repeat.
func (a *Alerts) StopRepeat() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.repeat != nil {
		a.repeat.Stop()
		a.repeat = nil
	}
}

// TestSound plays path (empty for the built-in chime) and reports whether it could.
func (a *Alerts) TestSound(path string) error {
	return a.sounder.Play(path)
}
