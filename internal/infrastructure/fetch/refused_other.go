//go:build !windows

package fetch

import (
	"errors"
	"syscall"
)

// isRefused reports a refused connection.
func isRefused(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED)
}
