package main

import "testing"

// TestCloseAsksOnlyWhereTheTrayCanBringTheWindowBack: the close button offers the choice on
// Windows with WebVigil running; it quits off Windows, while storage is unavailable (no tray was
// started) and once an explicit Quit is under way.
func TestCloseAsksOnlyWhereTheTrayCanBringTheWindowBack(t *testing.T) {
	cases := []struct {
		name                           string
		quitting, hidesToTray, running bool
		want                           bool
	}{
		{"windows, running", false, true, true, true},
		{"no tray (macOS, Linux)", false, false, true, false},
		{"storage unavailable, no tray started", false, true, false, false},
		{"quit already under way", true, true, true, false},
	}
	for _, c := range cases {
		if got := closeAsks(c.quitting, c.hidesToTray, c.running); got != c.want {
			t.Errorf("%s: closeAsks = %v, want %v", c.name, got, c.want)
		}
	}
}
