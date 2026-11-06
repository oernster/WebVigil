//go:build !windows

package autostart

import "errors"

// Registry is unavailable off Windows.
type Registry struct{}

// New returns the unavailable entry.
func New() Registry { return Registry{} }

// Set reports that sign-in start is a Windows feature.
func (Registry) Set(bool) error { return errors.New("start at sign-in is available on Windows only") }
