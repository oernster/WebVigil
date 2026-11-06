// Command installer is the WebVigil setup program: a second Wails application that carries the
// built app as an embedded zip and installs, updates, repairs or removes it, all per user with no
// administrator rights. Its install policy lives in internal/infrastructure/setup.
package main

import (
	"embed"
	"log"
	"os"
	"slices"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// payload is the built application; build.ps1 replaces the committed empty placeholder.
//
//go:embed payload.zip
var payload []byte

// appVersion is stamped from VERSION by build.ps1 through -ldflags -X; it must stay a var.
var appVersion = "0.0.0-dev"

const (
	windowTitle = "WebVigil Setup"
	windowW     = 700
	windowH     = 640
	// The window background before the page paints: the dark surface token.
	bgRed, bgGreen, bgBlue, bgAlpha = 6, 18, 42, 255
	uninstallFlag                   = "-uninstall"
)

func main() {
	app := newApp(payload, appVersion, slices.Contains(os.Args[1:], uninstallFlag))
	err := wails.Run(&options.App{
		Title:            windowTitle,
		Width:            windowW,
		Height:           windowH,
		DisableResize:    true,
		BackgroundColour: &options.RGBA{R: bgRed, G: bgGreen, B: bgBlue, A: bgAlpha},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		Bind:             []any{app},
	})
	if err != nil {
		log.Printf("setup: %v", err)
	}
}
