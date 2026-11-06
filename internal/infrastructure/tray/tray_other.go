//go:build !windows

package tray

// Tray is a no-op off Windows; WebVigil ships for Windows only.
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
