package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"webvigil/internal/infrastructure/setup"
)

const (
	progressEvent  = "progress"
	stepLogName    = "setup.log"
	licenceEntry   = "LICENSE"
	stepTimeLayout = "15:04:05.000"
)

// App is the setup program's facade over the setup package. It owns no install policy: it reads
// the machine once, runs the steps the page asks for and reports progress.
type App struct {
	ctx       context.Context
	payload   []byte
	version   string
	uninstall bool
	machine   setup.Machine
}

func newApp(payload []byte, version string, uninstall bool) *App {
	return &App{payload: payload, version: version, uninstall: uninstall}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.machine = setup.ReadMachine()
}

// StateDTO is the one reading of the machine the page decides everything from.
type StateDTO struct {
	Route         setup.Route   `json:"route"`
	Words         setup.Wording `json:"words"`
	ThisVersion   string        `json:"thisVersion"`
	Machine       setup.Machine `json:"machine"`
	UninstallOnly bool          `json:"uninstallOnly"`
	LogPath       string        `json:"logPath"`
}

// Options are the choices on the route screen.
type Options struct {
	SignIn    bool `json:"signIn"`
	StartMenu bool `json:"startMenu"`
	Desktop   bool `json:"desktop"`
}

// State returns the route, its wording and what the machine holds.
func (a *App) State() StateDTO {
	m := a.machine
	r := setup.DecideRoute(m.Installed, m.InstalledVersion, a.version)
	return StateDTO{Route: r, Words: setup.Words(r, m.InstalledVersion, a.version), ThisVersion: a.version,
		Machine: m, UninstallOnly: a.uninstall, LogPath: logPath()}
}

// AppRunning reports whether WebVigil is open; the page asks before touching any file.
func (a *App) AppRunning() bool { return setup.AppRunning() }

// CloseApp ends a running WebVigil.
func (a *App) CloseApp() error { return setup.CloseApp() }

// Install installs, updates, downgrades or repairs with the chosen options.
func (a *App) Install(o Options) error {
	dir, err := setup.InstallDir()
	if err != nil {
		return err
	}
	log := newStepLog()
	defer log.close()
	steps := []func() error{
		func() error { return setup.ExtractZip(a.payload, dir) },
		func() error {
			self, err := os.Executable()
			if err != nil {
				return err
			}
			if err := setup.CopyFile(self, filepath.Join(dir, setup.MaintenanceExe)); err != nil {
				return err
			}
			return setup.WriteUninstallEntry(a.version, dir)
		},
		func() error { return setup.SetShortcuts(dir, o.StartMenu, o.Desktop) },
		func() error { return setup.SetSignIn(dir, o.SignIn) },
	}
	return a.run(log, setup.InstallSteps, steps)
}

// ApplyOption applies one option at once, for the Manage route where there is nothing to install.
func (a *App) ApplyOption(o Options) error {
	dir, err := setup.InstallDir()
	if err != nil {
		return err
	}
	if err := setup.SetShortcuts(dir, o.StartMenu, o.Desktop); err != nil {
		return err
	}
	return setup.SetSignIn(dir, o.SignIn)
}

// Uninstall removes WebVigil; its websites and history go too only when removeData is set.
func (a *App) Uninstall(removeData bool) error {
	dir, err := setup.InstallDir()
	if err != nil {
		return err
	}
	log := newStepLog()
	defer log.close()
	steps := []func() error{
		func() error { return setup.SetShortcuts(dir, false, false) },
		func() error { setup.RemoveRegistration(); return nil },
		func() error {
			if !removeData {
				return nil
			}
			return setup.RemoveData()
		},
		func() error { return setup.ScheduleRemoval(dir) },
	}
	return a.run(log, setup.RemoveSteps, steps)
}

// run performs each step, logging and reporting it; the first failure ends the run.
func (a *App) run(log *stepLog, names []setup.Step, steps []func() error) error {
	for i, step := range steps {
		runtime.EventsEmit(a.ctx, progressEvent, map[string]any{"pct": setup.Progress(names, i), "step": names[i].Name})
		started := time.Now()
		err := step()
		log.write(names[i].Name, time.Since(started), err)
		if err != nil {
			return fmt.Errorf("%s: %w", names[i].Name, err)
		}
	}
	runtime.EventsEmit(a.ctx, progressEvent, map[string]any{"pct": setup.Progress(names, len(steps)), "step": "Done"})
	return nil
}

// Launch starts the installed WebVigil, so the page can bring it forward and only then close.
func (a *App) Launch() error {
	dir, err := setup.InstallDir()
	if err != nil {
		return err
	}
	return setup.LaunchApp(dir)
}

// Licence returns the licence carried in the payload beside WebVigil.exe.
func (a *App) Licence() (string, error) {
	r, err := zip.NewReader(bytes.NewReader(a.payload), int64(len(a.payload)))
	if err != nil {
		return "", err
	}
	for _, f := range r.File {
		if f.Name == licenceEntry {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			b, err := io.ReadAll(rc)
			return string(b), err
		}
	}
	return "", fmt.Errorf("this setup carries no licence file")
}

// Quit closes setup.
func (a *App) Quit() { runtime.Quit(a.ctx) }

// stepLog records every step with its duration, flushed as it goes, because the worst setup
// failures are the ones that never raise. The timings are what the bar's weights come from.
type stepLog struct{ f *os.File }

func logPath() string {
	dir, err := setup.DataDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, stepLogName)
}

func newStepLog() *stepLog {
	p := logPath()
	if p == "" {
		return &stepLog{}
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	f, _ := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	return &stepLog{f: f}
}

func (l *stepLog) write(step string, took time.Duration, err error) {
	if l.f == nil {
		return
	}
	result := "ok"
	if err != nil {
		result = err.Error()
	}
	fmt.Fprintf(l.f, "%s %-28s %6d ms  %s\n", time.Now().Format(stepTimeLayout), step, took.Milliseconds(), result)
	_ = l.f.Sync()
}

func (l *stepLog) close() {
	if l.f != nil {
		_ = l.f.Close()
	}
}
