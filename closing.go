package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"webvigil/internal/infrastructure/tray"
)

// eventCloseRequest asks the page to offer the close choice: keep running in the tray or quit.
const eventCloseRequest = "webvigil:close-request"

// beforeClose runs when the window's close button is pressed (ported from PigeonPost). Where a
// hidden window can be brought back from the tray, it keeps the window open and asks the page to
// offer Minimise to Tray or Quit; the page then calls MinimiseToTray or Quit. Everywhere else the
// close goes ahead and WebVigil quits, as it does whenever an explicit Quit is already under way:
// off Windows there is no tray; while storage could not be opened no tray was started.
func (a *App) beforeClose(ctx context.Context) bool {
	if !closeAsks(a.quitting.Load(), tray.HidesToTray, a.svc != nil) {
		return false
	}
	// A close asked for while the window is minimised (the taskbar's Close window) would otherwise
	// show the choice on a window nobody can see.
	a.showWindow()
	runtime.EventsEmit(ctx, eventCloseRequest)
	return true
}

// closeAsks decides whether pressing the close button asks rather than quits.
func closeAsks(quitting, hidesToTray, running bool) bool {
	return !quitting && hidesToTray && running
}

// MinimiseToTray hides the window; monitoring carries on and the tray icon's Open brings it back.
func (a *App) MinimiseToTray() { runtime.WindowHide(a.ctx) }

// Quit exits WebVigil completely. It records that the quit is intended first, so beforeClose lets
// the close through instead of asking again.
func (a *App) Quit() {
	a.quitting.Store(true)
	runtime.Quit(a.ctx)
}
