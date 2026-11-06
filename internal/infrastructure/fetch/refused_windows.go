//go:build windows

package fetch

import (
	"errors"

	"golang.org/x/sys/windows"
)

// isRefused reports a refused connection. On Windows the dial fails with WSAECONNREFUSED, which
// errors.Is does not equate with syscall.ECONNREFUSED (measured: connectex error 10061).
func isRefused(err error) bool {
	return errors.Is(err, windows.WSAECONNREFUSED)
}
