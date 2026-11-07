package autostart

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEntryWritesRewritesAndRemoves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "LaunchAgents", launchAgentFile)
	text := "first"
	e := Entry{path: path, content: func() (string, error) { return text, nil }}
	if err := e.Set(false); err != nil {
		t.Fatalf("removing an absent entry: %v", err)
	}
	if on, err := e.Enabled(); err != nil || on {
		t.Fatalf("before any entry: Enabled = %v, %v", on, err)
	}
	if err := e.Set(true); err != nil {
		t.Fatal(err)
	}
	if on, err := e.Enabled(); err != nil || !on {
		t.Fatalf("after Set(true): Enabled = %v, %v", on, err)
	}
	text = "second"
	if err := e.Set(true); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "second" {
		t.Fatalf("entry holds %q, want the rewrite", got)
	}
	if err := e.Set(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("entry still present after removal: %v", err)
	}
}

func TestEntryReportsWhyItCannotBeWritten(t *testing.T) {
	want := errors.New("no home folder")
	if err := (Entry{err: want}).Set(true); !errors.Is(err, want) {
		t.Fatalf("Set = %v, want %v", err, want)
	}
	if _, err := (Entry{err: want}).Enabled(); !errors.Is(err, want) {
		t.Fatalf("Enabled = %v, want %v", err, want)
	}
	failing := Entry{path: filepath.Join(t.TempDir(), "x"), content: func() (string, error) { return "", want }}
	if err := failing.Set(true); !errors.Is(err, want) {
		t.Fatalf("Set = %v, want the content error", err)
	}
}

func TestLaunchAgentPlistRunsTheExecutableAtLoad(t *testing.T) {
	exe := "/Applications/Web & Vigil.app/Contents/MacOS/WebVigil"
	plist := LaunchAgentPlist(exe)
	if err := xml.Unmarshal([]byte(plist), new(struct{})); err != nil {
		t.Fatalf("plist is not well-formed XML: %v", err)
	}
	for _, want := range []string{"<string>" + ValueName + "</string>", "Web &amp; Vigil.app", "<key>RunAtLoad</key>"} {
		if !strings.Contains(plist, want) {
			t.Errorf("plist lacks %q:\n%s", want, plist)
		}
	}
	if strings.Contains(plist, HiddenFlag) {
		t.Error("the LaunchAgent must open the window: off Windows there is no tray to wait in")
	}
}

func TestDesktopEntryAndExec(t *testing.T) {
	if got := DesktopExec(`/opt/web $vigil/"w"`, ""); got != `"/opt/web \$vigil/\"w\""` {
		t.Errorf("quoted Exec = %s", got)
	}
	if got := DesktopExec("/app/bin/webvigil", "app.id.WebVigil"); got != "flatpak run app.id.WebVigil" {
		t.Errorf("flatpak Exec = %s", got)
	}
	entry := DesktopEntry("flatpak run app.id.WebVigil")
	for _, want := range []string{"[Desktop Entry]", "Exec=flatpak run app.id.WebVigil", "Name=" + ValueName} {
		if !strings.Contains(entry, want) {
			t.Errorf("entry lacks %q", want)
		}
	}
}

func TestAutostartDir(t *testing.T) {
	home := filepath.Join("home", "u")
	cases := []struct {
		xdg       string
		inFlatpak bool
		want      string
	}{
		{"", false, filepath.Join(home, ".config", "autostart")},
		{filepath.Join("cfg"), false, filepath.Join("cfg", "autostart")},
		{filepath.Join(home, ".var", "app", "x", "config"), true, filepath.Join(home, ".config", "autostart")},
	}
	for _, c := range cases {
		if got := AutostartDir(home, c.xdg, c.inFlatpak); got != c.want {
			t.Errorf("AutostartDir(%q, %v) = %s, want %s", c.xdg, c.inFlatpak, got, c.want)
		}
	}
}
