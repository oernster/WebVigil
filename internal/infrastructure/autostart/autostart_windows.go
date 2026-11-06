//go:build windows

package autostart

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

// Registry implements application.AutoStart against a Run key.
type Registry struct {
	keyPath string
	exe     func() (string, error)
}

// New returns the sign-in entry for the running executable.
func New() Registry { return Registry{keyPath: RunKeyPath, exe: os.Executable} }

// Set writes or removes the entry.
func (r Registry) Set(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, r.keyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open the sign-in key: %w", err)
	}
	defer key.Close()
	if !enabled {
		if err := key.DeleteValue(ValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf("remove the sign-in entry: %w", err)
		}
		return nil
	}
	exe, err := r.exe()
	if err != nil {
		return fmt.Errorf("find WebVigil.exe: %w", err)
	}
	if err := key.SetStringValue(ValueName, Command(exe)); err != nil {
		return fmt.Errorf("write the sign-in entry: %w", err)
	}
	return nil
}
