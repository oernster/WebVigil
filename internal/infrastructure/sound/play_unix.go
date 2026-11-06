//go:build darwin || linux

package sound

import "os/exec"

// Player implements application.Sounder through the platform's own command-line player.
type Player struct{}

// Play sounds path (the built-in alarm when path is empty) and returns when it has finished.
func (Player) Play(path string) error {
	return external{players: players, lookPath: exec.LookPath, run: func(name string, args ...string) error {
		return exec.Command(name, args...).Run()
	}}.play(path)
}
