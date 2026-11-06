package main

import (
	_ "embed"
)

// The Help menu's backend, ported from PigeonPost: About, the licence text and the update check.

//go:embed LICENSE
var licenceText string

// releaseRepo is the "owner/name" of the repository that publishes WebVigil's releases. build.ps1
// stamps it through -ldflags -X from the build's own git remote, so no account is written into the
// source (CODE RULES 14); it must stay a var. A build without it reports the check as unreachable.
var releaseRepo = ""

const (
	appTagline   = "Watch the web. Know what changes."
	appAuthor    = "Oliver Ernster"
	appCopyright = "Copyright (c) Oliver Ernster"
	appLicence   = "GPL-3.0"
)

// appCredits are the open-source components WebVigil ships, each with the licence read from its own
// LICENSE file. Build-only tools (the icon generator's image library) are not shipped and not listed.
var appCredits = []CreditDTO{
	{Name: "Go", Licence: "BSD-3-Clause"},
	{Name: "Wails", Licence: "MIT"},
	{Name: "React", Licence: "MIT"},
	{Name: "Vite", Licence: "MIT"},
	{Name: "andybalholm/cascadia", Licence: "BSD-2-Clause"},
	{Name: "golang.org/x/net", Licence: "BSD-3-Clause"},
	{Name: "golang.org/x/sys", Licence: "BSD-3-Clause"},
	{Name: "modernc.org/sqlite", Licence: "BSD-3-Clause"},
}

// About returns the application metadata and open-source credits for Help > About.
func (a *App) About() AboutDTO {
	return AboutDTO{
		Name:      productName,
		Tagline:   appTagline,
		Version:   a.version,
		Author:    appAuthor,
		Copyright: appCopyright,
		Licence:   appLicence,
		LogPath:   a.logPath,
		Credits:   appCredits,
	}
}

// LicenceText returns the full licence text bundled with the application.
func (a *App) LicenceText() string {
	return licenceText
}

// CheckForUpdates runs one update check against the latest published release. skippedVersion is
// the release the user chose to skip, passed as empty for a manual check so an explicit request
// still reports it. Wails runs bound calls off the main thread, so the short network wait never
// blocks the window; failure is reported as no update, never as an error.
func (a *App) CheckForUpdates(skippedVersion string) UpdateStatusDTO {
	status := a.updates.Check(a.ctx, skippedVersion)
	return UpdateStatusDTO{
		Current:         status.Current,
		Latest:          status.Latest,
		UpdateAvailable: status.UpdateAvailable,
		DownloadURL:     status.DownloadURL,
		PageURL:         status.PageURL,
	}
}
