package sound

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakePlayers answers lookPath for the names present and records what run was asked to play.
type fakePlayers struct {
	present map[string]bool
	ran     []string
	played  []byte // the file's bytes at the moment it was played
	runErr  error
}

func (f *fakePlayers) external(players []command) external {
	return external{
		players: players,
		lookPath: func(name string) (string, error) {
			if f.present[name] {
				return "/bin/" + name, nil
			}
			return "", errors.New("not found")
		},
		run: func(name string, args ...string) error {
			f.ran = append([]string{name}, args...)
			f.played, _ = os.ReadFile(args[len(args)-1])
			return f.runErr
		},
	}
}

var linuxLike = []command{{name: "paplay"}, {name: "aplay", args: []string{"-q"}}}

func TestBuiltInAlarmIsPlayedFromATemporaryWAV(t *testing.T) {
	f := &fakePlayers{present: map[string]bool{"aplay": true}}
	if err := f.external(linuxLike).play(""); err != nil {
		t.Fatal(err)
	}
	if len(f.ran) != 3 || f.ran[0] != "/bin/aplay" || f.ran[1] != "-q" {
		t.Fatalf("ran %v; want the first player present, with its arguments before the file", f.ran)
	}
	if !bytes.Equal(f.played, AlarmWAV()) {
		t.Fatal("the player was not handed the built-in alarm")
	}
	if _, err := os.Stat(f.ran[2]); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the temporary alarm was left behind: %v", err)
	}
}

func TestChosenFileIsPlayedInPlace(t *testing.T) {
	wav := filepath.Join(t.TempDir(), "mine.wav")
	if err := os.WriteFile(wav, []byte("RIFF"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &fakePlayers{present: map[string]bool{"paplay": true, "aplay": true}}
	if err := f.external(linuxLike).play(wav); err != nil {
		t.Fatal(err)
	}
	if strings.Join(f.ran, " ") != "/bin/paplay "+wav {
		t.Fatalf("ran %v; want paplay, the first in order", f.ran)
	}
	if _, err := os.Stat(wav); err != nil {
		t.Fatal("the user's own file must never be removed")
	}
}

func TestPlayReportsEachFailure(t *testing.T) {
	f := &fakePlayers{}
	if err := f.external(linuxLike).play(filepath.Join(t.TempDir(), "gone.wav")); err == nil {
		t.Error("a missing sound file must be reported")
	}
	err := f.external(linuxLike).play("")
	if err == nil || !strings.Contains(err.Error(), "paplay, aplay") {
		t.Errorf("with no player present: %v; want the names looked for", err)
	}
	f = &fakePlayers{present: map[string]bool{"paplay": true}, runErr: errors.New("exit status 1")}
	if err := f.external(linuxLike).play(""); err == nil || !strings.Contains(err.Error(), "paplay could not play") {
		t.Errorf("a failing player: %v; want it named", err)
	}
}
