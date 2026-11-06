package sound

import (
	"fmt"
	"os"
	"strings"
)

// command is one external sound player: the program and the arguments that come before the file.
type command struct {
	name string
	args []string
}

// external plays through the first of its players found on the PATH: macOS and Linux each ship
// one, so WebVigil carries no audio library of its own there. Each player returns when the sound
// has finished, which is how the alert service keeps two alarms from overlapping.
type external struct {
	players  []command
	lookPath func(string) (string, error)
	run      func(name string, args ...string) error
}

// play sounds path, the built-in alarm when it is empty, written to a temporary WAV for the player.
func (x external) play(path string) error {
	if path == "" {
		tmp, err := os.CreateTemp("", "webvigil-alarm-*.wav")
		if err != nil {
			return fmt.Errorf("the built-in alarm could not be prepared: %w", err)
		}
		defer os.Remove(tmp.Name())
		_, werr := tmp.Write(AlarmWAV())
		if cerr := tmp.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return fmt.Errorf("the built-in alarm could not be prepared: %w", werr)
		}
		path = tmp.Name()
	} else if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("sound file %q: %w", path, err)
	}
	names := make([]string, 0, len(x.players))
	for _, p := range x.players {
		bin, err := x.lookPath(p.name)
		if err != nil {
			names = append(names, p.name)
			continue
		}
		if err := x.run(bin, append(append([]string{}, p.args...), path)...); err != nil {
			return fmt.Errorf("%s could not play %q; it may not be a WAV file: %w", p.name, path, err)
		}
		return nil
	}
	return fmt.Errorf("no sound player was found (looked for %s)", strings.Join(names, ", "))
}
