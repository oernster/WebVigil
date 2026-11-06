package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"webvigil/internal/application"
	"webvigil/internal/domain"
	"webvigil/internal/infrastructure/files"
	"webvigil/internal/infrastructure/storage"
	"webvigil/internal/infrastructure/tray"
)

// services are the application services built by the composition root. They are absent only
// when storage could not be opened, which every bound method reports through ready().
type services struct {
	store     *storage.Store
	settings  *application.Settings
	alerts    *application.Alerts
	checker   *application.Checker
	engine    *application.Engine
	sites     *application.Sites
	exchange  *application.Exchange
	history   *application.History
	scheduler *application.Scheduler
	files     files.Files
}

// App is the facade bound to the page. It translates between the wire and the services and owns
// nothing else: every rule lives in the application layer.
type App struct {
	version string
	logPath string
	problem string
	svc     *services
	tray    *tray.Tray
	// updates is built apart from svc: the update check needs no storage, so it still answers when
	// WebVigil's data could not be opened.
	updates *application.UpdateService

	ctx    context.Context
	stop   context.CancelFunc
	toasts bool

	mu      sync.Mutex
	pending map[string]application.ImportPreview // import previews awaiting confirmation, by path
}

func newApp(version, logPath string) *App {
	return &App{version: version, logPath: logPath, pending: map[string]application.ImportPreview{}}
}

// errUnavailable is returned by every action while storage could not be opened.
var errUnavailable = errors.New("WebVigil's data could not be opened; see the message at the top of the window")

func (a *App) ready() (*services, error) {
	if a.svc == nil {
		return nil, errUnavailable
	}
	return a.svc, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.svc == nil {
		return
	}
	a.toasts = initToasts(a)
	a.tray.Start(tray.Actions{
		Open:           a.showWindow,
		RefreshAll:     func() { _, _ = a.RefreshAll() },
		TogglePause:    func() { _ = a.SetPaused(!a.svc.settings.Current().Paused) },
		Exit:           func() { runtime.Quit(a.ctx) },
		BalloonClicked: a.showWindow,
	})
	runCtx, stop := context.WithCancel(context.Background())
	a.stop = stop
	go func() {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("scheduler panicked: %v", p)
				a.emitProblem(fmt.Sprintf("Automatic monitoring stopped after an internal error: %v", p))
			}
		}()
		a.svc.scheduler.Run(runCtx, application.SchedulerTick)
	}()
	a.refreshTray()
}

// shutdown stops the scheduler, cancels running checks, waits for them and closes storage, in
// that order, so no check writes after the database is closed (FR-088).
func (a *App) shutdown(context.Context) {
	if a.svc == nil {
		return
	}
	if a.stop != nil {
		a.stop()
	}
	a.svc.engine.Stop()
	a.svc.alerts.StopRepeat()
	a.tray.Stop()
	if err := a.svc.store.Close(); err != nil {
		log.Printf("close storage: %v", err)
	}
}

func (a *App) showWindow() {
	if a.ctx == nil {
		return
	}
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
}

// State returns what the page needs first, including any startup problem.
func (a *App) State() AppStateDTO {
	st := AppStateDTO{Version: a.version, Problem: a.problem, LogPath: a.logPath, Toasts: a.toasts}
	if a.svc != nil {
		st.Settings = settingsDTO(a.svc.settings.Current())
	} else {
		st.Settings = settingsDTO(domain.DefaultSettings())
	}
	return st
}

// ListSites returns every row of the main table.
func (a *App) ListSites() ([]SiteDTO, error) {
	svc, err := a.ready()
	if err != nil {
		return nil, err
	}
	views, err := svc.sites.List(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]SiteDTO, 0, len(views))
	for _, v := range views {
		out = append(out, siteDTO(v))
	}
	return out, nil
}

// NewSiteForm returns a blank definition carrying every default.
func (a *App) NewSiteForm() SiteFormDTO { return formDTO(domain.NewSite("", "")) }

// GetSite returns one definition for the Edit dialog.
func (a *App) GetSite(id int64) (SiteFormDTO, error) {
	svc, err := a.ready()
	if err != nil {
		return SiteFormDTO{}, err
	}
	s, err := svc.sites.Get(context.Background(), domain.SiteID(id))
	return formDTO(s), err
}

// AddSite stores a new definition; it is checked at once.
func (a *App) AddSite(form SiteFormDTO, allowDuplicate bool) (int64, error) {
	svc, err := a.ready()
	if err != nil {
		return 0, err
	}
	id, err := svc.sites.Add(context.Background(), form.site(), allowDuplicate)
	if err != nil {
		return 0, err
	}
	_ = svc.engine.Start(id, false)
	return int64(id), nil
}

// UpdateSite replaces a definition; see application.Sites.Update for the baseline rule.
func (a *App) UpdateSite(form SiteFormDTO, rebaseline, allowDuplicate bool) error {
	svc, err := a.ready()
	if err != nil {
		return err
	}
	return svc.sites.Update(context.Background(), form.site(), rebaseline, allowDuplicate)
}

// DeleteSite removes a site and its history.
func (a *App) DeleteSite(id int64) error {
	svc, err := a.ready()
	if err != nil {
		return err
	}
	return svc.sites.Delete(context.Background(), domain.SiteID(id))
}

// SetEnabled turns scheduled monitoring of one site on or off.
func (a *App) SetEnabled(id int64, enabled bool) error {
	svc, err := a.ready()
	if err != nil {
		return err
	}
	return svc.sites.SetEnabled(context.Background(), domain.SiteID(id), enabled)
}

// Refresh starts a manual check of one site.
func (a *App) Refresh(id int64) error {
	svc, err := a.ready()
	if err != nil {
		return err
	}
	return svc.engine.Start(domain.SiteID(id), true)
}

// RefreshAll starts a manual check of every enabled site.
func (a *App) RefreshAll() (int, error) {
	svc, err := a.ready()
	if err != nil {
		return 0, err
	}
	return svc.scheduler.RefreshAll(context.Background())
}

// Acknowledge clears a site's pending change, keeping its history.
func (a *App) Acknowledge(id int64) error {
	svc, err := a.ready()
	if err != nil {
		return err
	}
	return svc.alerts.Acknowledge(context.Background(), domain.SiteID(id))
}

// OpenWebsite opens a site in the default browser.
func (a *App) OpenWebsite(id int64) error {
	svc, err := a.ready()
	if err != nil {
		return err
	}
	s, err := svc.sites.Get(context.Background(), domain.SiteID(id))
	if err != nil {
		return err
	}
	return a.OpenExternal(s.URL)
}

// OpenExternal hands an http(s) address to the default browser. WebVigil never fetches such a page
// itself; the browser does the asking, so the donate link and Open Website add no traffic of
// WebVigil's own. Any other scheme is refused.
func (a *App) OpenExternal(address string) error {
	u := strings.TrimSpace(address)
	if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
		return fmt.Errorf("only web addresses can be opened, not %q", u)
	}
	runtime.BrowserOpenURL(a.ctx, u)
	return nil
}

// Preview extracts an unsaved definition once and stores nothing.
func (a *App) Preview(form SiteFormDTO) (PreviewDTO, error) {
	svc, err := a.ready()
	if err != nil {
		return PreviewDTO{}, err
	}
	p, err := svc.checker.Preview(context.Background(), form.site())
	if err != nil {
		return PreviewDTO{}, err
	}
	out := PreviewDTO{HTTPStatus: p.HTTPStatus, Category: string(p.Category), Detail: p.Detail, Text: p.Content.Text,
		Fields: []FieldValueDTO{}}
	for _, f := range p.Content.Fields {
		out.Fields = append(out.Fields, FieldValueDTO(f))
	}
	return out, nil
}

// refreshTray puts the monitoring state into the tray tooltip.
func (a *App) refreshTray() {
	if a.svc == nil {
		return
	}
	pending, _ := a.svc.store.PendingCount(context.Background())
	paused := a.svc.settings.Current().Paused
	tip := productName
	switch {
	case pending == 1:
		tip += ": 1 change to acknowledge"
	case pending > 1:
		tip += fmt.Sprintf(": %d changes to acknowledge", pending)
	}
	if paused {
		tip += " (paused)"
	}
	a.tray.SetState(tip, paused)
}
