package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "sites.webvigil.json")
	f := New()
	if err := f.WriteAtomic(path, []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := f.WriteAtomic(path, []byte("two")); err != nil {
		t.Fatal(err)
	}
	got, err := f.ReadLimited(path, 100)
	if err != nil || string(got) != "two" {
		t.Errorf("read = %q, %v", got, err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("temporary files left behind: %d entries", len(entries))
	}
	if err := f.WriteAtomic(filepath.Join(dir, "missing", "x.json"), []byte("x")); err == nil {
		t.Error("write into a missing folder succeeded")
	}
	if err := f.WriteAtomic(dir, []byte("x")); err == nil {
		t.Error("rename over a directory succeeded")
	}
}

func TestReadLimited(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "big")
	_ = os.WriteFile(path, []byte("0123456789"), 0o644)
	got, err := New().ReadLimited(path, 4)
	if err != nil || string(got) != "0123" {
		t.Errorf("limited read = %q, %v", got, err)
	}
	if _, err := New().ReadLimited(path+"x", 4); err == nil {
		t.Error("missing file read")
	}
}
