// Package runlog keeps the log a run leaves: a line naming when the run started, then everything
// the run writes to its error output. Ported from PigeonPost, which ported it from Bridge Talk.
//
// A windowed Windows program is started with no error output, so without this every log line and
// the Go runtime's own crash report would go nowhere.
package runlog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

const (
	// FileName names the log inside the application's data folder.
	FileName = "run.log"
	// MaxBytes is the size past which a run starts the log afresh, so it never grows without end.
	MaxBytes = 1 << 20

	startedLayout = "2006-01-02 15:04:05"
	folderPerm    = 0o755
	filePerm      = 0o644
)

// Open opens the log at path and writes the run's start line, starting afresh past MaxBytes.
func Open(path, app string, started time.Time) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), folderPerm); err != nil {
		return nil, fmt.Errorf("runlog: make the folder for %s: %w", path, err)
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if info, err := os.Stat(path); err == nil && info.Size() > MaxBytes {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, filePerm)
	if err != nil {
		return nil, fmt.Errorf("runlog: open %s: %w", path, err)
	}
	if _, err := fmt.Fprintf(file, "%s started %s\n", app, started.Format(startedLayout)); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("runlog: write to %s: %w", path, err)
	}
	return file, nil
}

// Keep sends what the run reports to file for the rest of the run, including the runtime's crash
// report, whether or not the run was given an error output of its own.
func Keep(file *os.File) error {
	log.SetOutput(file)
	if !hasErrorOutput() {
		return sendAll(file)
	}
	if err := debug.SetCrashOutput(file, debug.CrashOptions{}); err != nil {
		return fmt.Errorf("runlog: copy crash reports to %s: %w", file.Name(), err)
	}
	return nil
}

// Logger adapts the standard log package to application.Logger.
type Logger struct{}

// Printf writes one line to the run log.
func (Logger) Printf(format string, args ...any) { log.Printf(format, args...) }
