//go:build windows

package sound

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// PlaySound flags. Playback is synchronous on purpose: Play returns when the sound has finished,
// which is how the alert service keeps two alarms from overlapping. sndNoDefault stops Windows
// substituting its own beep when a sound cannot be played, so a failure is reported, not masked.
const (
	sndSync      = 0x0000
	sndNoDefault = 0x0002
	sndMemory    = 0x0004
	sndFilename  = 0x00020000
)

var (
	modwinmm      = windows.NewLazySystemDLL("winmm.dll")
	procPlaySound = modwinmm.NewProc("PlaySoundW")
)

// Player implements application.Sounder.
type Player struct{}

// Play sounds path (the built-in alarm when path is empty) and returns when it has finished.
func (Player) Play(path string) error {
	if path == "" {
		wav := AlarmWAV()
		if r, _, _ := procPlaySound.Call(uintptr(unsafe.Pointer(&wav[0])), 0, sndMemory|sndSync|sndNoDefault); r == 0 {
			return fmt.Errorf("the built-in alarm could not be played")
		}
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("sound file %q: %w", path, err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("sound file %q: %w", path, err)
	}
	if r, _, _ := procPlaySound.Call(uintptr(unsafe.Pointer(name)), 0, sndFilename|sndSync|sndNoDefault); r == 0 {
		return fmt.Errorf("sound file %q could not be played; it may not be a WAV file", path)
	}
	return nil
}
