// Package tray shows WebVigil in the Windows notification area: an icon read from the executable,
// a menu of Open, Refresh All, Pause or Resume and Exit, a tooltip carrying the monitoring state and
// a balloon used when toast notifications are unavailable. Ported from PigeonPost's taskbar tray.
package tray

// Actions are the callbacks the tray invokes. The composition root supplies them, so this package
// has no dependency on the window framework.
type Actions struct {
	Open           func()
	RefreshAll     func()
	TogglePause    func()
	Exit           func()
	BalloonClicked func()
}

// Menu labels.
const (
	labelOpen    = "Open WebVigil"
	labelRefresh = "Refresh All"
	labelPause   = "Pause Monitoring"
	labelResume  = "Resume Monitoring"
	labelExit    = "Exit"
)

// pauseLabel names the action the pause item will perform.
func pauseLabel(paused bool) string {
	if paused {
		return labelResume
	}
	return labelPause
}

func invoke(f func()) {
	if f != nil {
		f()
	}
}
