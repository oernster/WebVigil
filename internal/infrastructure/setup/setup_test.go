package setup

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.Write([]byte(body))
	}
	_ = w.Close()
	return buf.Bytes()
}

func TestExtractZip(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "WebVigil")
	if err := ExtractZip(zipOf(t, map[string]string{"WebVigil.exe": "exe", "sub/readme.txt": "hi"}), dest); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "sub", "readme.txt")); string(b) != "hi" {
		t.Errorf("extracted %q", b)
	}
	if DirSizeKB(dest) != 0 {
		t.Error("two tiny files are under a kilobyte")
	}
}

func TestExtractZipFenceWritesNothing(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "WebVigil")
	payload := zipOf(t, map[string]string{"a.txt": "first", "../escape.txt": "out"})
	if err := ExtractZip(payload, dest); err == nil {
		t.Fatal("an escaping entry was accepted")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("files were written before the fence refused the archive")
	}
	if err := ExtractZip([]byte("not a zip"), dest); err == nil {
		t.Error("a corrupt payload was accepted")
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a")
	_ = os.WriteFile(src, []byte("setup"), 0o644)
	if err := CopyFile(src, filepath.Join(dir, "b")); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(filepath.Join(dir, "missing"), filepath.Join(dir, "c")); err == nil {
		t.Error("copying a missing file succeeded")
	}
}

func TestDirsFollowLocalAppData(t *testing.T) {
	t.Setenv("LOCALAPPDATA", `C:\Users\someone\AppData\Local`)
	inst, _ := InstallDir()
	data, _ := DataDir()
	if inst != filepath.Join(`C:\Users\someone\AppData\Local`, "Programs", "WebVigil") ||
		data != filepath.Join(`C:\Users\someone\AppData\Local`, "WebVigil") {
		t.Errorf("dirs = %s, %s", inst, data)
	}
	t.Setenv("LOCALAPPDATA", "")
	if _, err := InstallDir(); err == nil {
		t.Error("no LOCALAPPDATA was accepted")
	}
}

// TestRemoveDataRemovesTheProfileToo: deleting the user's data takes the window's WebView2 profile
// with the data folder; both environment folders point at scratch space, never the real ones.
func TestRemoveDataRemovesTheProfileToo(t *testing.T) {
	local, roaming := t.TempDir(), t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	t.Setenv("APPDATA", roaming)
	data, profile := filepath.Join(local, AppName), filepath.Join(roaming, ExeName)
	for _, d := range []string{data, profile} {
		if err := os.MkdirAll(filepath.Join(d, "inner"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := ProfileDir(); got != profile {
		t.Errorf("ProfileDir = %s, want %s", got, profile)
	}
	if err := RemoveData(); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{data, profile} {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Errorf("%s survived: %v", d, err)
		}
	}
	if err := RemoveData(); err != nil {
		t.Errorf("removing folders already gone: %v", err)
	}
	t.Setenv("APPDATA", "")
	if err := RemoveData(); err == nil {
		t.Error("no APPDATA was accepted")
	}
	t.Setenv("LOCALAPPDATA", "")
	if err := RemoveData(); err == nil {
		t.Error("no LOCALAPPDATA was accepted")
	}
}

// TestRemovalWaitsThenRemovesEveryFolder: the command pauses before deleting, so setup has exited
// and let go of its folders, then removes each one, quoted for paths with spaces.
func TestRemovalWaitsThenRemovesEveryFolder(t *testing.T) {
	got := RemovalCommand([]string{`C:\Program Dir\WebVigil`, `C:\Roaming\WebVigilSetup.exe`})
	want := removalWait + ` & rmdir /s /q "C:\Program Dir\WebVigil" & rmdir /s /q "C:\Roaming\WebVigilSetup.exe"`
	if got != want {
		t.Errorf("RemovalCommand = %s\nwant           %s", got, want)
	}
}

// TestSetupProfileFollowsTheRunningExecutable: WebView2 names the profile after the executable, so
// a setup saved under another name still finds its own.
func TestSetupProfileFollowsTheRunningExecutable(t *testing.T) {
	roaming := t.TempDir()
	t.Setenv("APPDATA", roaming)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := SetupProfileDir(); got != filepath.Join(roaming, filepath.Base(exe)) {
		t.Errorf("SetupProfileDir = %s", got)
	}
	t.Setenv("APPDATA", "")
	if _, err := SetupProfileDir(); err == nil {
		t.Error("no APPDATA was accepted")
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{{"0.1.0", "0.2.0", -1}, {"1.10.0", "1.9.9", 1}, {"v1.2.3", "1.2.3", 0}, {"1.2.3-dev", "1.2.3", 0}, {"2", "1.9", 1}}
	for _, c := range cases {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%s, %s) = %d", c.a, c.b, got)
		}
	}
}

func TestDecideRouteAndWords(t *testing.T) {
	cases := []struct {
		installed bool
		have      string
		want      Route
		action    string
		flow      bool
	}{
		{false, "", RouteInstall, "Install", false},
		{true, "0.0.9", RouteUpdate, "Update", true},
		{true, "0.2.0", RouteDowngrade, "Go back", true},
		{true, "0.1.0", RouteManage, "Repair", false},
	}
	for _, c := range cases {
		r := DecideRoute(c.installed, c.have, "0.1.0")
		w := Words(r, c.have, "0.1.0")
		if r != c.want || w.Action != c.action || w.Flow != c.flow || w.Heading == "" || w.Lead == "" {
			t.Errorf("%v %s: route %s, words %+v", c.installed, c.have, r, w)
		}
	}
}

func TestProgress(t *testing.T) {
	if Progress(InstallSteps, 0) != 0 || Progress(InstallSteps, len(InstallSteps)) != 100 || Progress(nil, 0) != 0 {
		t.Error("progress ends wrong")
	}
	// Weighted by measured time: the bar barely moves until the shortcuts, which take nearly all of it.
	if p := Progress(InstallSteps, 2); p >= 10 {
		t.Errorf("progress before the shortcuts = %d, want under 10", p)
	}
	if p := Progress(InstallSteps, 3); p != 100 {
		t.Errorf("progress after the shortcuts = %d, want 100", p)
	}
	// Removal: scheduling the program files' removal is the longest step, so it moves the bar most.
	if p := Progress(RemoveSteps, 3); p >= 50 {
		t.Errorf("removal progress before the program files = %d, want under 50", p)
	}
}
