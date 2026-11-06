//go:build !windows && !darwin && !linux

package autostart

import "errors"

// New returns an entry that reports sign-in start is unavailable: WebVigil writes one for Windows,
// macOS and Linux only.
func New() Entry {
	return Entry{err: errors.New("start at sign-in is available on Windows, macOS and Linux only")}
}
