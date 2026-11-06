//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

// New returns the LaunchAgent for the running executable; launchd reads ~/Library/LaunchAgents at
// login, so writing the file is the whole of turning it on.
func New() Entry {
	home, err := os.UserHomeDir()
	if err != nil {
		return Entry{err: fmt.Errorf("find the home folder for the sign-in entry: %w", err)}
	}
	return Entry{
		path: filepath.Join(home, "Library", "LaunchAgents", launchAgentFile),
		content: func() (string, error) {
			exe, err := os.Executable()
			if err != nil {
				return "", fmt.Errorf("find WebVigil: %w", err)
			}
			return LaunchAgentPlist(exe), nil
		},
	}
}
