// Command WebVigil watches web pages for meaningful changes. This file is the composition root:
// the only place where infrastructure is constructed and handed to the application services.
package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"webvigil/internal/application"
	"webvigil/internal/domain"
	"webvigil/internal/infrastructure/autostart"
	"webvigil/internal/infrastructure/clock"
	"webvigil/internal/infrastructure/extract"
	"webvigil/internal/infrastructure/fetch"
	"webvigil/internal/infrastructure/files"
	"webvigil/internal/infrastructure/runlog"
	"webvigil/internal/infrastructure/sound"
	"webvigil/internal/infrastructure/storage"
	"webvigil/internal/infrastructure/tray"
	"webvigil/internal/infrastructure/update"
)

//go:embed all:frontend/dist
var assets embed.FS

// appVersion is stamped from VERSION by build.ps1 through -ldflags -X; it must stay a var.
var appVersion = "0.0.0-dev"

const (
	productName    = "WebVigil"
	dataFolderName = "WebVigil"
	databaseName   = "webvigil.db"
	instanceID     = "webvigil-single-instance-6f1c2d3e"
	windowWidth    = 1280
	windowHeight   = 760
	minWidth       = 900
	minHeight      = 520
	// The window background shown before the page paints: the dark surface token.
	bgRed, bgGreen, bgBlue, bgAlpha = 6, 18, 42, 255
)

func main() {
	started := time.Now()
	dataDir := dataDirectory()
	logPath := filepath.Join(dataDir, runlog.FileName)
	if f, err := runlog.Open(logPath, productName+" "+appVersion, started); err == nil {
		if err := runlog.Keep(f); err != nil {
			log.Printf("%v", err)
		}
	}
	hidden := slices.Contains(os.Args[1:], autostart.HiddenFlag)
	app := newApp(appVersion, logPath)
	app.updates = application.NewUpdateService(
		update.NewGitHubReleaseSource(releaseRepo), appVersion, application.PlatformKeyFor(runtime.GOOS))
	if err := compose(app, dataDir); err != nil {
		// Never end the run before the window opens: the problem is carried into it (FR-115).
		app.problem = fmt.Sprintf("WebVigil could not open its data in %s: %v", dataDir, err)
		log.Print(app.problem)
	}
	err := wails.Run(&options.App{
		Title:             productName,
		Width:             windowWidth,
		Height:            windowHeight,
		MinWidth:          minWidth,
		MinHeight:         minHeight,
		StartHidden:       hidden,
		HideWindowOnClose: true,
		BackgroundColour:  &options.RGBA{R: bgRed, G: bgGreen, B: bgBlue, A: bgAlpha},
		AssetServer:       &assetserver.Options{Assets: assets},
		OnStartup:         app.startup,
		OnShutdown:        app.shutdown,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               instanceID,
			OnSecondInstanceLaunch: func(options.SecondInstanceData) { app.showWindow() },
		},
		Windows: &windows.Options{Theme: windows.SystemDefault},
		Bind:    []any{app},
	})
	if err != nil {
		log.Printf("wails: %v", err)
	}
}

// dataDirectory is %LOCALAPPDATA%\WebVigil, where every file WebVigil writes lives (NFR-PRV-001).
func dataDirectory() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		if dir, err := os.UserConfigDir(); err == nil {
			base = dir
		}
	}
	return filepath.Join(base, dataFolderName)
}

// compose builds every service. On failure the app keeps its problem and no services.
func compose(app *App, dataDir string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(dataDir, databaseName), storage.DefaultRetention)
	if err != nil {
		return err
	}
	logger := runlog.Logger{}
	sys := clock.System{}
	events := &wailsEvents{app: app}
	settings, err := application.NewSettings(ctx, store, autostart.New(), events)
	if err != nil {
		_ = store.Close()
		return err
	}
	fetcher := fetch.New(productName+"/"+appVersion, domain.MaxResponseBytes, domain.MaxRedirects)
	extractor := extract.New()
	alerts := application.NewAlerts(application.AlertsDeps{
		History: store, Sounder: sound.Player{}, Notifier: &notifier{app: app}, Settings: settings,
		Clock: sys, Events: events, Log: logger, After: clock.AfterFunc,
	})
	checker := application.NewChecker(store, store, fetcher, extractor, sys, alerts)
	engine := application.NewEngine(checker, events, logger)
	app.svc = &services{
		store:     store,
		settings:  settings,
		alerts:    alerts,
		checker:   checker,
		engine:    engine,
		sites:     application.NewSites(store, extractor, sys, engine, events),
		exchange:  application.NewExchange(store, files.New(), sys, events),
		history:   application.NewHistory(store),
		scheduler: application.NewScheduler(store, engine, settings, sys, logger),
		files:     files.New(),
	}
	app.tray = tray.New(productName)
	return nil
}
