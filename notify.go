package main

import (
	"fmt"
	"log"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"webvigil/internal/application"
	"webvigil/internal/domain"
)

// Event names and notification identifiers shared with the page and the toast system.
const (
	eventChanged     = "webvigil:changed"
	eventShowChanges = "webvigil:show-changes"
	eventProblem     = "webvigil:problem"
	changeCategory   = "webvigil-change"
	actionView       = "view"
	actionOpen       = "open"
	actionAck        = "ack"
	defaultAction    = "DEFAULT_ACTION"
	dataSiteID       = "siteId"
	noticeTimeLayout = "15:04"
)

// wailsEvents tells the page that something changed and keeps the tray tooltip current.
type wailsEvents struct{ app *App }

func (e *wailsEvents) Emit(kind application.EventKind, id domain.SiteID) {
	if e.app.ctx == nil {
		return
	}
	runtime.EventsEmit(e.app.ctx, eventChanged, map[string]any{"kind": string(kind), "siteId": int64(id)})
	if kind == application.EventSitesChanged || kind == application.EventSettings || kind == application.EventCheckFinished {
		e.app.refreshTray()
	}
}

func (a *App) emitProblem(text string) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, eventProblem, text)
	}
}

// initToasts prepares Windows toast notifications with WebVigil's three actions. When they cannot
// be set up, notifier falls back to the tray balloon, so a change is never announced silently.
func initToasts(a *App) bool {
	if err := runtime.InitializeNotifications(a.ctx); err != nil {
		log.Printf("toast notifications unavailable, using tray balloons: %v", err)
		return false
	}
	if !runtime.IsNotificationAvailable(a.ctx) {
		return false
	}
	err := runtime.RegisterNotificationCategory(a.ctx, runtime.NotificationCategory{
		ID: changeCategory,
		Actions: []runtime.NotificationAction{
			{ID: actionView, Title: "View Changes"},
			{ID: actionOpen, Title: "Open Website"},
			{ID: actionAck, Title: "Acknowledge"},
		},
	})
	if err != nil {
		log.Printf("register notification actions: %v", err)
	}
	runtime.OnNotificationResponse(a.ctx, a.onNotification)
	return true
}

// onNotification acts on a click on a toast or one of its buttons.
func (a *App) onNotification(r runtime.NotificationResult) {
	if r.Error != nil {
		log.Printf("notification response: %v", r.Error)
		return
	}
	idValue, _ := r.Response.UserInfo[dataSiteID].(float64) // JSON numbers decode as float64
	id := int64(idValue)
	switch r.Response.ActionIdentifier {
	case actionOpen:
		if err := a.OpenWebsite(id); err != nil {
			log.Printf("open website %d: %v", id, err)
		}
	case actionAck:
		if err := a.Acknowledge(id); err != nil {
			log.Printf("acknowledge %d: %v", id, err)
		}
	default: // actionView, defaultAction and a click on the body
		a.showChanges(id)
	}
}

func (a *App) showChanges(id int64) {
	a.showWindow()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, eventShowChanges, id)
	}
}

// notifier implements application.Notifier.
type notifier struct{ app *App }

func (n *notifier) NotifyChange(c application.ChangeNotice) {
	title := fmt.Sprintf("%s changed", c.Name)
	body := fmt.Sprintf("%s at %s", c.Summary, c.At.Local().Format(noticeTimeLayout))
	a := n.app
	if a.svc != nil && a.svc.settings.Current().ForegroundDialog {
		a.showChanges(int64(c.SiteID))
	}
	if a.toasts {
		err := runtime.SendNotificationWithActions(a.ctx, runtime.NotificationOptions{
			ID: fmt.Sprintf("change-%d", c.ChangeID), Title: title, Body: body, CategoryID: changeCategory,
			Data: map[string]any{dataSiteID: int64(c.SiteID)},
		})
		if err == nil {
			return
		}
		log.Printf("toast failed, using the tray balloon: %v", err)
	}
	a.tray.Balloon(title, body)
}

func (n *notifier) NotifyFailures(f application.FailureNotice) {
	title := fmt.Sprintf("%s keeps failing", f.Name)
	body := fmt.Sprintf("%d checks in a row failed. Latest: %s", f.Failures, f.Detail)
	a := n.app
	if a.toasts {
		if err := runtime.SendNotification(a.ctx, runtime.NotificationOptions{
			ID: fmt.Sprintf("failures-%d", f.SiteID), Title: title, Body: body,
			Data: map[string]any{dataSiteID: int64(f.SiteID)},
		}); err == nil {
			return
		}
	}
	a.tray.Balloon(title, body)
}

var (
	_ application.Events   = (*wailsEvents)(nil)
	_ application.Notifier = (*notifier)(nil)
)
