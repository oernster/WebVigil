//go:build windows

package setup

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"webvigil/internal/infrastructure/autostart"
)

//lint:ignore ST1005 shown verbatim to the user
var errStillRunning = errors.New("WebVigil could not be closed. Please exit it from its tray icon and try again.")

const (
	uninstallKeyPath = `Software\Microsoft\Windows\CurrentVersion\Uninstall\WebVigil`
	shortcutName     = "WebVigil.lnk"
	closeWait        = 5 * time.Second
	closePoll        = 100 * time.Millisecond
	forcedExitCode   = 1
	// Wails registers the toast activator under the executable's file name (measured in its
	// notifications.go): an AppUserModelId entry naming a COM class, the class itself and a
	// categories key. Uninstall removes all three.
	toastAppIDKey      = `Software\Classes\AppUserModelId\` + ExeName
	toastActivatorName = "CustomActivator"
	toastCLSIDPrefix   = `Software\Classes\CLSID\`
	toastCategoriesKey = `SOFTWARE\` + ExeName
)

// Machine is what the machine already holds, read once so every option opens on what is true.
type Machine struct {
	Installed        bool   `json:"installed"`
	InstalledVersion string `json:"installedVersion"`
	InstallDir       string `json:"installDir"`
	SignIn           bool   `json:"signIn"`
	StartMenu        bool   `json:"startMenu"`
	Desktop          bool   `json:"desktop"`
}

// ReadMachine reads the install record, the sign-in entry and the shortcuts.
func ReadMachine() Machine {
	m := Machine{}
	m.InstallDir, _ = InstallDir()
	if key, err := registry.OpenKey(registry.CURRENT_USER, uninstallKeyPath, registry.QUERY_VALUE); err == nil {
		m.InstalledVersion, _, err = key.GetStringValue("DisplayVersion")
		m.Installed = err == nil
		key.Close()
	}
	if key, err := registry.OpenKey(registry.CURRENT_USER, autostart.RunKeyPath, registry.QUERY_VALUE); err == nil {
		_, _, err = key.GetStringValue(autostart.ValueName)
		m.SignIn = err == nil
		key.Close()
	}
	for _, s := range shortcutPaths() {
		if _, err := os.Stat(s.path); err == nil {
			if s.desktop {
				m.Desktop = true
			} else {
				m.StartMenu = true
			}
		}
	}
	return m
}

func hidden() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}

// processIDs finds every process by image name. Never by process tree: recorded parent ids churn
// and setup could find itself recorded as a descendant and end itself.
func processIDs() []uint32 {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snap)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	var ids []uint32
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		if strings.EqualFold(windows.UTF16ToString(e.ExeFile[:]), ExeName) {
			ids = append(ids, e.ProcessID)
		}
	}
	return ids
}

// AppRunning reports whether WebVigil is running.
func AppRunning() bool { return len(processIDs()) > 0 }

// CloseApp ends every WebVigil process and waits for the executable to be released.
func CloseApp() error {
	for _, pid := range processIDs() {
		if h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid); err == nil {
			_ = windows.TerminateProcess(h, forcedExitCode)
			windows.CloseHandle(h)
		}
	}
	deadline := time.Now().Add(closeWait)
	for AppRunning() {
		if time.Now().After(deadline) {
			return errStillRunning
		}
		time.Sleep(closePoll)
	}
	return nil
}

// LaunchApp starts the installed WebVigil detached, so it outlives setup.
func LaunchApp(dir string) error {
	cmd := exec.Command(filepath.Join(dir, ExeName))
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start WebVigil: %w", err)
	}
	return cmd.Process.Release()
}

// WriteUninstallEntry registers WebVigil in the Apps list with Modify and Repair offered, both of
// which reopen setup rather than sending the user back to the download.
func WriteUninstallEntry(version, dir string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, uninstallKeyPath, registry.WRITE)
	if err != nil {
		return fmt.Errorf("create the install record: %w", err)
	}
	defer key.Close()
	setup := filepath.Join(dir, MaintenanceExe)
	values := map[string]string{
		"DisplayName": AppName, "DisplayVersion": version, "Publisher": Publisher, "InstallLocation": dir,
		"DisplayIcon": filepath.Join(dir, ExeName), "UninstallString": `"` + setup + `" -uninstall`,
		"ModifyPath": `"` + setup + `"`,
	}
	for k, v := range values {
		if err := key.SetStringValue(k, v); err != nil {
			return fmt.Errorf("write %s: %w", k, err)
		}
	}
	for k, v := range map[string]uint32{"NoModify": 0, "NoRepair": 0, "EstimatedSize": DirSizeKB(dir)} {
		if err := key.SetDWordValue(k, v); err != nil {
			return fmt.Errorf("write %s: %w", k, err)
		}
	}
	return nil
}

// SetSignIn writes or removes the sign-in entry: the same single value WebVigil's Settings writes.
func SetSignIn(dir string, enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, autostart.RunKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open the sign-in key: %w", err)
	}
	defer key.Close()
	if !enabled {
		if err := key.DeleteValue(autostart.ValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}
	return key.SetStringValue(autostart.ValueName, autostart.Command(filepath.Join(dir, ExeName)))
}

type shortcut struct {
	path    string
	desktop bool
}

func shortcutPaths() []shortcut {
	var out []shortcut
	if appData := os.Getenv("APPDATA"); appData != "" {
		out = append(out, shortcut{filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", shortcutName), false})
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, shortcut{filepath.Join(home, "Desktop", shortcutName), true})
	}
	return out
}

// SetShortcuts creates or removes the Start Menu and Desktop shortcuts as chosen.
func SetShortcuts(dir string, startMenu, desktop bool) error {
	exe := filepath.Join(dir, ExeName)
	for _, s := range shortcutPaths() {
		want := (s.desktop && desktop) || (!s.desktop && startMenu)
		if !want {
			_ = os.Remove(s.path)
			continue
		}
		script := fmt.Sprintf(`$s=(New-Object -ComObject WScript.Shell).CreateShortcut(%q);$s.TargetPath=%q;`+
			`$s.IconLocation=%q;$s.WorkingDirectory=%q;$s.Save()`, s.path, exe, exe, dir)
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
		cmd.SysProcAttr = hidden()
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("create %s: %w: %s", s.path, err, out)
		}
	}
	return nil
}

// RemoveRegistration deletes the install record, the sign-in entry and the toast registration.
func RemoveRegistration() {
	_ = registry.DeleteKey(registry.CURRENT_USER, uninstallKeyPath)
	_ = SetSignIn("", false)
	if key, err := registry.OpenKey(registry.CURRENT_USER, toastAppIDKey, registry.QUERY_VALUE); err == nil {
		if guid, _, err := key.GetStringValue(toastActivatorName); err == nil && guid != "" {
			_ = registry.DeleteKey(registry.CURRENT_USER, toastCLSIDPrefix+guid+`\LocalServer32`)
			_ = registry.DeleteKey(registry.CURRENT_USER, toastCLSIDPrefix+guid)
		}
		key.Close()
	}
	_ = registry.DeleteKey(registry.CURRENT_USER, toastAppIDKey)
	_ = registry.DeleteKey(registry.CURRENT_USER, toastCategoriesKey+`\NotificationCategories`)
	_ = registry.DeleteKey(registry.CURRENT_USER, toastCategoriesKey)
}

// ScheduleRemoval removes the program folder after setup exits, since setup's own maintenance copy
// lives in it and cannot delete itself while running.
func ScheduleRemoval(dir string) error {
	cmd := exec.Command("cmd", "/C", fmt.Sprintf(`ping 127.0.0.1 -n 3 >nul & rmdir /s /q "%s"`, dir))
	cmd.SysProcAttr = hidden()
	return cmd.Start()
}
