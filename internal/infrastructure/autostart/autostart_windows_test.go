//go:build windows

package autostart

import (
	"errors"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// The test writes under a scratch key of its own beneath HKCU\Software, never the real Run key.
const scratchKey = `Software\WebVigilTest\Run`

func TestSetWritesAndRemovesEntry(t *testing.T) {
	r := Registry{keyPath: scratchKey, exe: func() (string, error) { return `C:\Apps\WebVigil\WebVigil.exe`, nil }}
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, scratchKey)
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\WebVigilTest`)
	})
	if on, err := r.Enabled(); err != nil || on {
		t.Fatalf("before any entry: Enabled = %v, %v", on, err)
	}
	if err := r.Set(true); err != nil {
		t.Fatal(err)
	}
	if on, err := r.Enabled(); err != nil || !on {
		t.Errorf("after Set(true): Enabled = %v, %v", on, err)
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, scratchKey, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	v, _, err := key.GetStringValue(ValueName)
	key.Close()
	if err != nil || v != `"C:\Apps\WebVigil\WebVigil.exe" -hidden` {
		t.Errorf("entry = %q, %v", v, err)
	}
	if err := r.Set(false); err != nil {
		t.Fatal(err)
	}
	if err := r.Set(false); err != nil {
		t.Errorf("removing an absent entry: %v", err)
	}
	if on, err := r.Enabled(); err != nil || on {
		t.Errorf("key without the value: Enabled = %v, %v", on, err)
	}
	key, _ = registry.OpenKey(registry.CURRENT_USER, scratchKey, registry.QUERY_VALUE)
	if _, _, err := key.GetStringValue(ValueName); !errors.Is(err, registry.ErrNotExist) {
		t.Errorf("entry still present: %v", err)
	}
	key.Close()
	failing := Registry{keyPath: scratchKey, exe: func() (string, error) { return "", errors.New("no exe") }}
	if err := failing.Set(true); err == nil {
		t.Error("missing executable not reported")
	}
}
