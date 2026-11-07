# WebVigil Architecture

## Invariant

`UI -> Application -> Domain <- Infrastructure`

Dependencies point inward. Each rule below is held by a test in
`tests/structural/`, not by convention.

| Invariant | Enforcing test |
|---|---|
| The domain imports pure standard-library packages only: no `os`, `log`, `net`, `path/filepath`, `syscall`, `database/sql`, `math/rand` or `runtime` | `TestDomainImportsOnlyStdlib` |
| The application imports the domain and the standard library only | `TestApplicationImportsOnlyDomain` |
| Neither inner layer reads the clock or draws randomness: time is injected | `TestInnerLayersReadNoClock` |
| Infrastructure never imports Wails; the window belongs to the composition root | `TestInfrastructureDoesNotImportTheUI` |
| No Go or TypeScript module exceeds 400 lines, nor sits in the band above 380 | `TestModuleSize` |
| The Go DTOs and the TypeScript wire interfaces name the same fields | `TestWireMatches` |
| Every supplied icon master is present and used | `TestSuppliedIconsAreUsed` |
| No shipped file names a particular person or their test data | `TestNoUserSpecificData` |
| No pane or text view wears a focus ring; the browser's own ring stays off scrollers | `TestNoPaneWearsARing`, `TestTheBrowserRingStaysOffScrollers` |
| Only package `main` at the root imports both application and infrastructure | `TestCompositionRootIsWhitelisted` |
| Only `fetch` and `update` import a package that opens a connection; elsewhere `net/http` may be used for its status names alone | `TestOnlyTheListedPackagesReachTheNetwork`, `TestEveryNetworkExemptionExists` |
| The page makes no request of its own: no `fetch`, `XMLHttpRequest`, `WebSocket`, `EventSource` or `sendBeacon` in `frontend/src` | `TestTheFrontEndMakesNoRequestOfItsOwn` |
| The setup page names the product only from the state Go sends it | `TestTheSetupPageNamesNoProduct` |

## Layers

- **Domain** (`internal/domain`): sites and their validation, URL
  normalisation, extraction content and hashing, normalisation, comparison,
  failure categories, challenge detection, line and field diffs, site state
  and backoff, import planning and the ready-made readings. Pure and at 100%
  coverage.
- **Application** (`internal/application`): the use cases and the ports they
  need (`ports.go`). `Checker` runs one check; `Engine` runs checks in the
  background; `Scheduler` decides what is due; `Alerts` notifies, rings and
  acknowledges; `Sites`, `Exchange`, `History`, `Settings` and
  `UpdateService` serve the rest. At 100% coverage, tested against
  hand-written fakes.
- **Infrastructure** (`internal/infrastructure`): one package per port.
  `storage` (SQLite), `fetch` (HTTP), `extract` (HTML), `files`, `clock`,
  `runlog`, `sound`, `tray`, `autostart`, `update` (GitHub releases) and
  `setup` (the Windows setup program's install logic).
- **UI**: the Wails facade in package `main` (`app.go`, `app_exchange.go`,
  `about.go`, `closing.go`, `exchange_filter.go`, `notify.go` and `dto.go`)
  plus the React page in `frontend/src`. The facade maps DTOs; its only rules
  concern the window itself: when the close button asks (`closing.go`) and the
  file-dialog filters macOS can accept (`exchange_filter.go`). On the page,
  `api.ts` is the only file that reaches Go; `logic.ts`, `platform.ts`,
  `reading.ts` and `autoScroll.ts` are pure.

## Composition root

`main.go` builds every adapter and injects it by constructor: no globals, no
service locator. When the data folder cannot be opened, the window still opens
and carries the problem (FR-115) rather than the process ending unseen.

## One check

`Checker.Check` (`internal/application/check.go`), in order:

1. Fetch, sending the stored `ETag` and `Last-Modified` only when a baseline
   exists, so a 304 can only confirm content already held.
2. Classify: a transport failure, a challenge page, a status outside 2xx, a
   missing selector, a parse failure or empty content each becomes a failure
   category with a detail.
3. Extract by mode, normalise by the ignore rules, hash.
4. Compare with the baseline: the first success is the baseline; a different
   hash is a change; the baseline then moves to the new content.
5. Write the check, any snapshot, any change event, the new state and the
   retention pass in one transaction.
6. Tell `Alerts` about a change or about failures reaching the site's
   threshold.

A cancelled check records nothing. Failures back off up to the longer of 24
hours and the site's interval.

## Scheduling

`Scheduler.Tick` runs every five seconds and starts each enabled site that is
due, unless monitoring is paused. `Engine` runs at most four checks at once,
one per site; no host is asked twice within ten seconds. A site's next
check is measured from its last.

## Storage

One SQLite file, schema in `internal/infrastructure/storage/schema.go`, applied
by numbered migrations that are never edited once released. Each site keeps its
newest 1,000 checks; every unacknowledged change and any acknowledged one among
its newest 50; its newest 50 snapshots, plus the baseline and any snapshot a
kept change refers to. Change events refer to snapshots by id with no foreign key,
so retention chooses what to keep.

## Network

| Package | Connects to |
|---|---|
| `fetch` | the pages the user configured, as `WebVigil/<version>`, with no cookies |
| `update` | the GitHub latest-release endpoint of the repository the build was stamped with |

Opening a site, a download or the donate page hands the address to the browser.
The tests above hold this list over the source; they cannot see a request the
web view makes on its own account.

## Platforms

Windows-only parts sit behind build tags: the tray, the registry sign-in entry
and the `winmm` alarm. On macOS and Linux, closing the window quits, sign-in
start writes a LaunchAgent or an XDG autostart entry and the alarm plays
through the system's own player.

## Quality enforcement

`./test.ps1` is the gate and `build.ps1` cannot skip it. See
[TESTING.md](TESTING.md).
