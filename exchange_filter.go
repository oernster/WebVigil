package main

import (
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"webvigil/internal/domain"
)

// darwinOS is the runtime.GOOS value for macOS.
const darwinOS = "darwin"

// exchangeFilters offers the WebVigil sites format in the file dialogs on goos.
//
// macOS takes the final extension segment only. Wails turns each pattern into a UTType and adds
// it to an array without a nil check; macOS knows no type for the two-part ".webvigil.json", so
// the lookup answers nil and the insert throws, ending the process as the dialog opens.
func exchangeFilters(goos string) []runtime.FileFilter {
	ext := domain.ExchangeExtension
	if goos == darwinOS {
		ext = filepath.Ext(ext)
	}
	return []runtime.FileFilter{
		{DisplayName: "WebVigil sites (*" + ext + ")", Pattern: "*" + ext},
		{DisplayName: "JSON files (*.json)", Pattern: "*.json"},
	}
}
