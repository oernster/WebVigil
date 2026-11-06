//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

// desktopFileName is the autostart entry's file name.
const desktopFileName = "webvigil.desktop"

// New returns the XDG autostart entry for the running WebVigil. FLATPAK_ID is set by flatpak inside
// its sandbox and names the installed app.
func New() Entry {
	home, err := os.UserHomeDir()
	if err != nil {
		return Entry{err: fmt.Errorf("find the home folder for the sign-in entry: %w", err)}
	}
	flatpakID := os.Getenv("FLATPAK_ID")
	dir := AutostartDir(home, os.Getenv("XDG_CONFIG_HOME"), flatpakID != "")
	return Entry{
		path: filepath.Join(dir, desktopFileName),
		content: func() (string, error) {
			exe, err := os.Executable()
			if err != nil {
				return "", fmt.Errorf("find WebVigil: %w", err)
			}
			return DesktopEntry(DesktopExec(exe, flatpakID)), nil
		},
	}
}
