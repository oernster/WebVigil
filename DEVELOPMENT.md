# WebVigil Development Guide

Building and the gate are in [README.md](README.md#build-and-test); this file
covers working on the source.

## Prerequisites

- Go at the version in `go.mod`.
- Node.js with npm, for the page.
- The Wails CLI matching `go.mod` (v2.12):
  `go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`.
- staticcheck needs no install: the gate runs it through `go run`.
- macOS and Linux also need a C compiler for the web view; Linux needs
  WebKitGTK 4.1, which the Flatpak build supplies.

## Running from source

```powershell
wails dev
```

The version reads `0.0.0-dev` and the update check reports every check as
unreachable, since no release repository is stamped in. The data folder is the
same one an installed copy uses (see README), so a dev run shares its sites.
The exception is a run started from inside a packaged (MSIX) Windows app, such as
the Claude desktop app: Windows gives it that app's private copy of AppData, so
it reads and writes there instead.

## Project layout

| Path | Holds |
|---|---|
| `main.go` | the composition root |
| the other root `.go` files | the Wails facade; `dto.go` holds its DTOs |
| `internal/domain` | the rules: pure, 100% covered |
| `internal/application` | the use cases and their ports, 100% covered |
| `internal/infrastructure` | one adapter package per port |
| `frontend/src` | the React page; `api.ts` is its only door to Go |
| `installer/` | the Windows setup program, a second Wails application |
| `tests/structural` | the architecture tests |
| `tools/genicons` | derives every icon from the masters in `assets/` |
| `docs/` | the GitHub Pages site |

## Generated files

- `go run ./tools/genicons` writes every icon from `assets/`; `build.ps1` runs
  it first. Never edit its outputs by hand.
- Wails writes `frontend/wailsjs/` on every build; it is ignored by git.
- `python stamp_version.py` copies `VERSION` into the site's version tokens.
- `installer/payload.zip` is committed as an empty placeholder; `build.ps1`
  fills it for the setup build and restores it afterwards.

## A new binding

Add the method to the facade, its DTO to `dto.go`, the same fields to
`frontend/src/wire.ts` (`TestWireMatches` compares them) and the call to
`frontend/src/api.ts`.

## Line endings

`.gitattributes` keeps `*.sh` and `*.go` LF on every checkout: the scripts run
on macOS and Linux; gofmt rejects CRLF.

## Versioning

`VERSION` is the only home of the version. The builds stamp it into
`main.appVersion` through `-ldflags`; the site carries it through
`stamp_version.py`. No Markdown file names a version.

## Cutting a release

1. Set `VERSION`, then run `./build.ps1` on Windows, `bash builddmg.sh` on
   the Mac and `bash build_flatpak.sh` on Linux.
2. Publish a GitHub release tagged `v<version>` with `WebVigilSetup.exe`,
   `WebVigil.dmg` and `webvigil.flatpak` attached, using `NOTES.md` as its body.
   The update check matches each platform's asset by its file suffix.

## Licence

GPL-3.0; see [LICENSE](LICENSE).
