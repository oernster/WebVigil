// Package files implements application.Files: bounded reads and atomic writes.
package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Files reads and writes exchange files on disk.
type Files struct{}

// New returns the file service.
func New() Files { return Files{} }

// ReadLimited reads at most limit bytes of path. A caller that passes its own cap plus one can
// tell an oversized file from one exactly at the cap without reading the rest of it.
func (Files) ReadLimited(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, limit))
}

// WriteAtomic writes data to a temporary file beside path, flushes it to disk and renames it
// over path, so a reader never sees a half-written file (FR-011).
func (Files) WriteAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	cleanup := func(e error) error {
		_ = tmp.Close()
		_ = os.Remove(name)
		return e
	}
	if _, err := tmp.Write(data); err != nil {
		return cleanup(fmt.Errorf("write %s: %w", name, err))
	}
	if err := tmp.Sync(); err != nil {
		return cleanup(fmt.Errorf("flush %s: %w", name, err))
	}
	if err := tmp.Close(); err != nil {
		return cleanup(err)
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}
