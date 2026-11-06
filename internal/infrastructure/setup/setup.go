// Package setup is the install policy of the WebVigil setup program: where things go, how the
// payload is unpacked, which conversation a run is having; in the Windows half, the registry,
// shortcut and process work. The setup program's window is a thin facade over it.
package setup

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// AppName names the install folder, the uninstall record and the shortcuts.
	AppName = "WebVigil"
	// ExeName is the installed application.
	ExeName = "WebVigil.exe"
	// MaintenanceExe is the copy of the setup program kept beside the app, which the Apps list
	// opens for Modify, Repair and Uninstall.
	MaintenanceExe = "WebVigilSetup.exe"
	// Publisher is recorded in the uninstall entry.
	Publisher = "Oliver Ernster"

	programsSubdir = "Programs"
	dirPerm        = 0o755
	bytesPerKB     = 1024
)

func localAppData() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", fmt.Errorf("LOCALAPPDATA is not set")
	}
	return base, nil
}

// InstallDir is %LOCALAPPDATA%\Programs\WebVigil: per user, so no administrator rights are needed.
func InstallDir() (string, error) {
	base, err := localAppData()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, programsSubdir, AppName), nil
}

// DataDir is %LOCALAPPDATA%\WebVigil, where the application keeps its database and run log.
func DataDir() (string, error) {
	base, err := localAppData()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppName), nil
}

// ExtractZip unpacks the payload into dest. Every entry is checked against dest BEFORE any file
// is written, so a crafted archive that would climb out of the install folder writes nothing.
func ExtractZip(data []byte, dest string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open payload: %w", err)
	}
	for _, f := range r.File {
		if _, err := fenced(dest, f.Name); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(dest, dirPerm); err != nil {
		return fmt.Errorf("create %q: %w", dest, err)
	}
	for _, f := range r.File {
		if err := extractEntry(f, dest); err != nil {
			return err
		}
	}
	return nil
}

// fenced returns the target of name inside dest, refusing one that would land outside it.
func fenced(dest, name string) (string, error) {
	target := filepath.Join(dest, name)
	inside := filepath.Clean(dest) + string(os.PathSeparator)
	if filepath.IsAbs(name) || !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), inside) {
		return "", fmt.Errorf("the payload holds an unsafe path: %q", name)
	}
	return target, nil
}

func extractEntry(f *zip.File, dest string) error {
	target, _ := fenced(dest, f.Name) // every entry was checked by ExtractZip before this runs
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, dirPerm)
	}
	if err := os.MkdirAll(filepath.Dir(target), dirPerm); err != nil {
		return fmt.Errorf("create folder for %q: %w", target, err)
	}
	src, err := f.Open()
	if err != nil {
		return fmt.Errorf("open %q: %w", f.Name, err)
	}
	defer src.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, dirPerm)
	if err != nil {
		return fmt.Errorf("create %q: %w", target, err)
	}
	if _, err := io.Copy(out, src); err != nil {
		out.Close()
		return fmt.Errorf("write %q: %w", target, err)
	}
	return out.Close()
}

// DirSizeKB is the size of a folder tree in KB, for the uninstall entry's EstimatedSize.
func DirSizeKB(dir string) uint32 {
	var total int64
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return uint32(total / bytesPerKB)
}

// CopyFile copies src to dst, used to keep the maintenance copy of the setup program.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, dirPerm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// CompareVersions compares major.minor.patch numerically, ignoring a leading v and a
// pre-release suffix: -1 when a is older than b, 0 when equal, 1 when newer.
func CompareVersions(a, b string) int {
	pa, pb := parseVersion(a), parseVersion(b)
	for i := range pa {
		switch {
		case pa[i] < pb[i]:
			return -1
		case pa[i] > pb[i]:
			return 1
		}
	}
	return 0
}

func parseVersion(v string) [3]int {
	core := strings.SplitN(strings.TrimPrefix(strings.TrimSpace(v), "v"), "-", 2)[0]
	var out [3]int
	for i, part := range strings.SplitN(core, ".", 3) {
		out[i], _ = strconv.Atoi(strings.TrimSpace(part))
	}
	return out
}
