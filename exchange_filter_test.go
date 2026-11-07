package main

import (
	"strings"
	"testing"
)

// TestExchangeFiltersGiveMacOSSingleSegmentExtensions: every pattern handed to the macOS dialog
// names one extension segment, since a two-part one crashes the Wails Cocoa dialog; other
// platforms keep the full WebVigil extension.
func TestExchangeFiltersGiveMacOSSingleSegmentExtensions(t *testing.T) {
	t.Parallel()
	for _, f := range exchangeFilters(darwinOS) {
		ext := strings.TrimPrefix(f.Pattern, "*.")
		if strings.Contains(ext, ".") {
			t.Errorf("macOS pattern %q has a multi-part extension", f.Pattern)
		}
	}
	for _, goos := range []string{"windows", "linux"} {
		if got := exchangeFilters(goos)[0].Pattern; got != "*.webvigil.json" {
			t.Errorf("%s: first pattern = %q, want *.webvigil.json", goos, got)
		}
	}
}
