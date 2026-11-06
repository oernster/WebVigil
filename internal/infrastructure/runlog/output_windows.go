//go:build windows

package runlog

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// hasErrorOutput reports whether the run was given an error output. A windowed program started
// without one reads a handle of 0, so everything it writes there is lost.
func hasErrorOutput() bool {
	handle, err := windows.GetStdHandle(windows.STD_ERROR_HANDLE)
	return err == nil && handle != 0 && handle != windows.InvalidHandle
}

// sendAll points the run's error output at file: the handle the runtime looks up for each report,
// then os.Stderr, which was fixed from that handle at start.
func sendAll(file *os.File) error {
	if err := windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(file.Fd())); err != nil {
		return fmt.Errorf("runlog: send error output to %s: %w", file.Name(), err)
	}
	os.Stderr = file
	return nil
}
