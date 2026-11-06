//go:build !windows

package sound

import "errors"

// Player implements application.Sounder; WebVigil plays sound on Windows only.
type Player struct{}

// Play reports that sound is unavailable on this platform.
func (Player) Play(string) error { return errors.New("sound is available on Windows only") }
