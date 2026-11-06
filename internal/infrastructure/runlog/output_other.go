//go:build !windows

package runlog

import "os"

func hasErrorOutput() bool { return true }

func sendAll(file *os.File) error {
	os.Stderr = file
	return nil
}
