package autostart

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Entry implements application.AutoStart as one file the desktop reads at sign-in: a LaunchAgent on
// macOS, an XDG autostart entry on Linux. It is built per platform by New; the file handling is the
// same for both, so it lives here, untagged, where every platform's tests reach it.
type Entry struct {
	path    string
	content func() (string, error)
	err     error // why no entry can be written at all (no home folder)
}

// Enabled reports whether the entry file is present.
func (e Entry) Enabled() (bool, error) {
	if e.err != nil {
		return false, e.err
	}
	_, err := os.Stat(e.path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read the sign-in entry: %w", err)
	}
	return true, nil
}

// Set writes the entry or removes it; removing an entry that is not there is not an error.
func (e Entry) Set(enabled bool) error {
	if e.err != nil {
		return e.err
	}
	if !enabled {
		if err := os.Remove(e.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove the sign-in entry: %w", err)
		}
		return nil
	}
	text, err := e.content()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(e.path), 0o755); err != nil {
		return fmt.Errorf("create the sign-in folder: %w", err)
	}
	if err := os.WriteFile(e.path, []byte(text), 0o644); err != nil {
		return fmt.Errorf("write the sign-in entry: %w", err)
	}
	return nil
}

// launchAgentLabel names the LaunchAgent, which launchd requires to be unique per user.
const launchAgentLabel = ValueName

// LaunchAgentPlist is the macOS LaunchAgent that runs exe once when the user logs in.
func LaunchAgentPlist(exe string) string {
	return strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`,
		`<plist version="1.0">`,
		`<dict>`,
		`	<key>Label</key>`,
		`	<string>` + xmlText(launchAgentLabel) + `</string>`,
		`	<key>ProgramArguments</key>`,
		`	<array>`,
		`		<string>` + xmlText(exe) + `</string>`,
		`	</array>`,
		`	<key>RunAtLoad</key>`,
		`	<true/>`,
		`</dict>`,
		`</plist>`,
		``,
	}, "\n")
}

func xmlText(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// DesktopEntry is the XDG autostart entry that runs exec when the user logs in.
func DesktopEntry(exec string) string {
	return strings.Join([]string{
		"[Desktop Entry]",
		"Type=Application",
		"Name=" + ValueName,
		"Exec=" + exec,
		"Terminal=false",
		"X-GNOME-Autostart-enabled=true",
		"",
	}, "\n")
}

// DesktopExec is the Exec line for the running WebVigil. Inside a flatpak the executable's path
// exists only in the sandbox, so the desktop is asked to run the installed app by its id instead;
// otherwise the path is quoted by the Desktop Entry rules ("\", "\"", "`" and "$" escaped).
func DesktopExec(exe, flatpakID string) string {
	if flatpakID != "" {
		return "flatpak run " + flatpakID
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`)
	return `"` + r.Replace(exe) + `"`
}

// AutostartDir is the folder the desktop reads autostart entries from. A flatpak sees its own
// XDG_CONFIG_HOME inside the sandbox, which the desktop never reads, so there the host's
// ~/.config is used (the manifest grants xdg-config/autostart).
func AutostartDir(home, xdgConfigHome string, inFlatpak bool) string {
	if xdgConfigHome == "" || inFlatpak {
		xdgConfigHome = filepath.Join(home, ".config")
	}
	return filepath.Join(xdgConfigHome, "autostart")
}

// launchAgentFile is the LaunchAgent's file name in ~/Library/LaunchAgents.
const launchAgentFile = launchAgentLabel + ".plist"
