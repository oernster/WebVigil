//go:build !windows

package tray

// HidesToTray reports that there is no tray icon to bring a hidden window back from, so closing the
// window quits WebVigil rather than stranding it out of sight.
const HidesToTray = false

// Tray is a no-op off Windows: macOS and Linux run WebVigil while its window is open.
type Tray struct{}

// New returns a tray that does nothing.
func New(string) *Tray { return &Tray{} }

// Start does nothing.
func (*Tray) Start(Actions) {}

// Stop does nothing.
func (*Tray) Stop() {}

// SetState does nothing.
func (*Tray) SetState(string, bool) {}

// Balloon does nothing.
func (*Tray) Balloon(string, string) {}
