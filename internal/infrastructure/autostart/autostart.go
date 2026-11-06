// Package autostart writes WebVigil's sign-in entry under HKCU\...\Run (REQUIREMENTS.md FR-112).
// The setup program writes the same single entry, so the two can never disagree.
package autostart

// RunKeyPath is the per-user Run key; ValueName is WebVigil's entry in it.
const (
	RunKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	ValueName  = "WebVigil"
	// HiddenFlag makes a sign-in start wait in the tray rather than open the window.
	HiddenFlag = "-hidden"
)

// Command returns the Run value for exe: the quoted path plus the hidden flag.
func Command(exe string) string {
	return `"` + exe + `" ` + HiddenFlag
}
