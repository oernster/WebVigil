// Package autostart writes WebVigil's single sign-in entry (REQUIREMENTS.md FR-112): on Windows a
// value under HKCU\...\Run, which the setup program writes too so the two can never disagree; on
// macOS a LaunchAgent; on Linux an XDG autostart entry.
package autostart

// RunKeyPath is the per-user Run key; ValueName is WebVigil's entry in it. ValueName names the
// entry on every other platform too.
const (
	RunKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	ValueName  = "WebVigil"
	// HiddenFlag makes a sign-in start wait in the tray rather than open the window. Only Windows
	// has a tray, so only the Run value carries it.
	HiddenFlag = "-hidden"
)

// Command returns the Run value for exe: the quoted path plus the hidden flag.
func Command(exe string) string {
	return `"` + exe + `" ` + HiddenFlag
}
