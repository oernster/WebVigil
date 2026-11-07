# WebVigil requirements

Watch the web. Know what changes.

This document is the baselined specification for WebVigil. Each requirement has
a stable identifier that is never reused or renumbered, an EARS statement, an
acceptance example and the test that verifies it. A requirement that can only
be settled on a real desktop says so; those checks are run by hand against a
real build.

The overriding requirement, from which every other one follows:

> A user shall always be able to tell whether a website was checked
> successfully, whether its content changed and why a check failed.

## 1. Introduction

### 1.1 Purpose

WebVigil is a desktop application for Windows, macOS and Linux that checks user-defined web pages on
a schedule, records what each check found and raises an alarm when the
meaningful content of a page changes. It runs locally with no cloud service, no
AI model and no subscription.

### 1.2 Scope

In scope: website definitions, scheduled and manual checks, change detection by
deterministic comparison, failure classification, notifications, history with a
diff viewer, import and export of definitions, a tray presence on Windows,
sign-in start, light and dark appearance, a per-user setup program on Windows,
a disk image for macOS and a Flatpak for Linux.

Out of scope (Won't this time):

- Monitoring before the user signs in (no background service).
- Authenticated monitoring: no stored credentials, cookies or sessions.
- Running JavaScript from fetched pages; pages that render their content only
  through script are reported as an extraction failure, not worked around.
- Any attempt to solve, bypass or disguise against CAPTCHA or bot challenges.
- AI interpretation of changes.
- A full application-state backup (snapshots and history); export carries
  definitions only.
- Telemetry. Network traffic beyond the checks the user configured and the one
  release query of FR-144 is out of scope.
- A tray presence on macOS or Linux.

### 1.3 Definitions

| Term | Meaning |
|---|---|
| Site | One monitored definition: a name, a URL and how to read it. |
| Check | One attempt to fetch, validate, extract and compare a site. |
| Outcome | The result class of a check: success or failure. |
| Category | The specific reason a check failed, from the table in FR-040. |
| Content | The comparable text produced by extraction and normalisation. |
| Hash | The SHA-256 of the canonical content, hex encoded. |
| Baseline | The last accepted content of a site and its hash. |
| Change event | A record that content differed from the baseline. |
| Pending alert | A change event not yet acknowledged. |
| Baseline key | The settings whose change makes old content incomparable: URL, detection mode, extraction rules and ignore rules. |

### 1.4 Decisions recorded at baseline (2026-11-06, Oliver)

| Decision | Value |
|---|---|
| Response size cap | 8 MiB |
| Redirect cap | 10 |
| Concurrent checks | 4 |
| Minimum spacing between two requests to one host | 10 s |
| Shortest check interval | 5 min |
| Default check interval | 60 min |
| Default request timeout | 30 s |
| Backoff ceiling | 24 h |
| Check history kept per site | 1,000 |
| Snapshots kept per site | 50 |
| Import file cap | 1 MiB |
| Sites per import | 500 |
| Failure warning threshold default | 3 consecutive failures |
| Repeat alarm until acknowledged | off by default |
| Built-in alarm | a chime synthesised in code; no sound file shipped |
| Delivery | Windows: `WebVigil.exe` inside a per-user setup program in the house style; macOS: a signed, notarised `WebVigil.dmg` for Apple Silicon; Linux: a `webvigil.flatpak` bundle |
| Closing the window | Windows: asks Minimise to Tray or Quit; macOS and Linux: quits |

## 2. Overall description

- **User class:** one person on a Windows, macOS or Linux desktop, signed in,
  monitoring a handful to a few hundred pages.
- **Environment:** Windows 10 or 11 on x64 with the WebView2 runtime; macOS on
  Apple Silicon; Linux through Flatpak on the GNOME runtime.
- **Constraints:** Go, Wails v2, React with TypeScript, SQLite through
  `modernc.org/sqlite` (no CGO), everything written per user (NFR-PRV-001).
- **Assumption (owner Oliver):** the target pages serve their meaningful content
  in the initial HTML. Where they do not, FR-044 reports it.

## 3. Functional requirements

Priority is MoSCoW. Every requirement here is Must unless marked otherwise.

### 3.1 Website management

**FR-001 Add a site.** When the user confirms a valid new definition, the site
service shall store it and make its first check due immediately.
Acceptance: given no sites, when `https://example.com/a` is added, then the list
holds one site with status Grey and its next check due at the current time.
Verified by: `internal/application/sites_test.go::TestAddSiteStoresAndIsDueNow`

**FR-002 Reject a bad URL.** If a URL is empty, unparsable, has no host or uses
a scheme other than `http` or `https`, then the site service shall refuse the
definition with a message naming the problem.
Acceptance: `ftp://x`, `javascript:alert(1)`, `example.com` and `https://` are
each refused.
Verified by: `internal/domain/url_test.go::TestNormaliseURLRejects`

**FR-003 Normalise URLs.** The domain shall normalise a URL by lower-casing the
scheme and host, removing a default port, removing the fragment and giving an
empty path the value `/`.
Acceptance: `HTTPS://Example.COM:443#top` normalises to `https://example.com/`.
Verified by: `internal/domain/url_test.go::TestNormaliseURL`

**FR-004 Warn on a duplicate.** When a definition is added whose normalised URL
matches an existing site, the site service shall report the duplicate and store
it only when the user confirms.
Verified by: `internal/application/sites_test.go::TestAddDuplicateNeedsConfirmation`

**FR-005 Edit without losing the baseline.** When only the name, interval,
timeout, enabled state or notification preferences change, the site service
shall keep the baseline and the history.
Verified by: `internal/application/sites_test.go::TestEditKeepsBaselineForCosmeticChange`

**FR-006 Edit that invalidates the baseline.** When an edit changes the
baseline key, the site service shall report that a new baseline is needed and
shall not compare later content against the old baseline. The next successful
check shall establish a new baseline without an alarm.
Verified by: `internal/application/sites_test.go::TestEditBaselineKeyResetsBaseline`

**FR-007 Delete a site.** When the user confirms deletion, the site service shall
remove the site with its state, history, snapshots and change events in one
transaction and stop any scheduled or running check for it.
Verified by: `internal/infrastructure/storage/sites_test.go::TestDeleteSiteRemovesEverything`
and `internal/application/engine_test.go::TestDeleteCancelsRunningCheck`

**FR-008 Delete is confirmed.** The interface shall ask for confirmation naming
the site and stating that its history is removed; Cancel shall change nothing.
Verified by: `frontend/src/components/ConfirmDelete.test.tsx`

**FR-009 Enable and disable.** While a site is disabled, the scheduler shall not
check it; a manual refresh shall still be allowed.
Verified by: `internal/application/scheduler_test.go::TestDisabledSiteIsNotScheduled`

### 3.2 Export and import

**FR-010 Export definitions.** When the user exports, the exchange service shall
write a JSON document with `format` `WebVigilSites`, `schemaVersion` 1, an
`exportedAt` timestamp and every site definition. It shall not write snapshots,
history, hashes, HTTP validators or state.
Verified by: `internal/domain/exchange_test.go::TestBuildExportCarriesDefinitionsOnly`

**FR-011 Export writes atomically.** The file writer shall write to a temporary
file in the destination folder and rename it into place.
Verified by: `internal/infrastructure/files/files_test.go::TestWriteAtomic`

**FR-012 Validate an import.** If the format is not `WebVigilSites`, the schema
version is unsupported, the file exceeds 1 MiB, it holds more than 500 sites or
the JSON is malformed, then the exchange service shall refuse the whole file
with a message naming the reason.
Verified by: `internal/domain/exchange_test.go::TestParseImportRefuses`

**FR-013 Report invalid entries.** If an entry in an otherwise valid file fails
site validation, then the exchange service shall list it as invalid with its
position and reason, then offer the valid entries.
Verified by: `internal/domain/exchange_test.go::TestParseImportListsInvalidEntries`

**FR-014 Preview with duplicate policy.** When an import is previewed, the
exchange service shall mark each entry whose normalised URL exists as a
duplicate, with Skip as the default action; Update and Import as separate are
offered per entry.
Verified by: `internal/domain/exchange_test.go::TestPlanImportDefaultsToSkip`

**FR-015 Import in one transaction.** When the user confirms an import, the
storage layer shall apply every chosen action in one transaction; if any write
fails, then no change shall remain.
Verified by: `internal/infrastructure/storage/sites_test.go::TestImportIsAtomic`

**FR-016 Import summary.** After an import, the interface shall show counts of
imported, updated, skipped and invalid entries.
Verified by: `internal/application/exchange_test.go::TestConfirmImportSummary`

**FR-017 Round trip.** Exporting then importing into an empty store shall
reproduce every exported field exactly.
Verified by: `internal/application/exchange_test.go::TestExportImportRoundTrip`

**FR-018 Imported sites baseline locally.** An imported or updated site shall
establish its own baseline on its first successful check without an alarm.
Verified by: `internal/application/exchange_test.go::TestImportedSiteBaselinesWithoutAlarm`

### 3.3 Fetching

**FR-020 Fetch.** The fetcher shall issue a GET with the configured timeout, the
User-Agent `WebVigil/<version>`, no cookies, decompression, TLS verification and
at most 10 redirects.
Verified by: `internal/infrastructure/fetch/fetch_test.go::TestFetchRecordsResponse`

**FR-021 Record the response.** The fetcher shall record the requested URL,
final URL, status, duration, Content-Type, body size, ETag, Last-Modified,
Retry-After and the time of the check.
Verified by: `internal/infrastructure/fetch/fetch_test.go::TestFetchRecordsResponse`

**FR-022 Size cap.** If a body exceeds 8 MiB, then the fetcher shall stop reading
and report `response_too_large`.
Verified by: `internal/infrastructure/fetch/fetch_test.go::TestFetchSizeCap`

**FR-023 No invented status.** If a check fails before an HTTP response arrives,
then the record shall carry no HTTP status.
Verified by: `internal/infrastructure/fetch/fetch_test.go::TestNetworkFailuresHaveNoStatus`

**FR-024 Conditional requests.** While a site has a baseline and stored
validators, the fetcher shall send `If-None-Match` and `If-Modified-Since`.
When the reply is 304 and a baseline exists, the check service shall record the
check as unchanged.
Verified by: `internal/application/check_test.go::TestNotModifiedWithBaselineIsUnchanged`

**FR-025 304 without baseline.** If a 304 arrives for a site with no baseline,
then the check service shall record a failure with category `unexpected_structure`.
Verified by: `internal/application/check_test.go::TestNotModifiedWithoutBaselineFails`

**FR-026 Content-Type.** If the Content-Type is not textual (HTML, XHTML, XML,
plain text or JSON), then the fetcher shall report `unsupported_content_type`.
Verified by: `internal/infrastructure/fetch/fetch_test.go::TestUnsupportedContentType`

**FR-027 Cancellation.** When a check is cancelled, the fetcher shall abort the
request and the check service shall record nothing as a success.
Verified by: `internal/application/engine_test.go::TestDeleteCancelsRunningCheck`

### 3.4 Failure classification

**FR-040 Categories.** The fetcher and check service shall classify every failure
into exactly one category: `dns`, `connection_refused`, `connection_timeout`,
`tls`, `http_401`, `http_403`, `http_404`, `http_429`, `http_500`, `http_502`,
`http_503`, `http_504`, `http_status` (any other non-2xx), `redirect_loop`,
`unsupported_content_type`, `response_too_large`, `parse_failure`,
`selector_missing`, `challenge`, `unexpected_structure`, `invalid_content`,
`network`, `cancelled`.
Verified by: `internal/domain/failure_test.go::TestCategoryForStatus` and
`internal/infrastructure/fetch/fetch_test.go::TestNetworkFailuresHaveNoStatus`

**FR-041 Challenge pages.** If a response matches a marker in the challenge
table (CAPTCHA forms, bot-check interstitials, login walls), then the check
service shall record `challenge` regardless of the HTTP status.
Verified by: `internal/domain/challenge_test.go::TestDetectChallenge`

**FR-042 Failure keeps the baseline.** When a check fails, the check service
shall keep the baseline and shall not record a change.
Verified by: `internal/application/check_test.go::TestFailureKeepsBaseline`

**FR-043 Missing selector.** If a required selector or field matches nothing,
then the check service shall record `selector_missing` naming it.
Verified by: `internal/infrastructure/extract/extract_test.go::TestMissingSelector`

**FR-044 Empty content.** If extraction yields no text after normalisation,
then the check service shall record `invalid_content`.
Verified by: `internal/application/check_test.go::TestEmptyContentIsInvalid`

### 3.5 Extraction and comparison

**FR-050 Raw mode.** In raw mode the extractor shall use the decoded body.
**FR-051 Text mode (default).** In text mode the extractor shall take the visible
text, dropping `script`, `style`, `noscript`, `template`, `svg`, `head` and
elements marked `hidden` or `aria-hidden="true"`.
**FR-052 Selector mode.** In selector mode the extractor shall take the text of
every element matched by each configured selector, in document order.
**FR-053 Field mode.** In field mode the extractor shall take one value per
field rule from its selector, from an attribute when one is named, else its text.
Verified by: `internal/infrastructure/extract/extract_test.go` (one test per mode)

**FR-054 Ignore rules.** Before comparison the extractor shall remove elements
matching the excluded selectors; the normaliser shall remove text matching
each exclusion pattern, collapse whitespace when set and fold case when set.
Verified by: `internal/domain/normalise_test.go::TestIgnoreRules` and
`internal/infrastructure/extract/extract_test.go::TestExcludedSelectors`

**FR-055 Line endings.** The normaliser shall convert CRLF and CR to LF and trim
trailing whitespace on each line.
Verified by: `internal/domain/normalise_test.go::TestLineEndings`

**FR-056 Hash.** The domain shall hash the canonical content with SHA-256.
Verified by: `internal/domain/content_test.go::TestCanonicalHashIsStable`

**FR-057 Preview.** When the user asks for a preview while editing, the check
service shall fetch and extract once with the unsaved settings and show the
normalised content without storing anything.
Verified by: `internal/application/check_test.go::TestPreviewStoresNothing`

**FR-058 Ready-made readings.** When the address in the Add or Edit dialog is an
Amazon product page (`/dp/<ASIN>` or `/gp/product/<ASIN>` on any Amazon
storefront) and the definition is in text mode, the dialog shall fill in field
mode with the product's title (required), author, formats and prices, rating,
review count, description, cover and details, exclude the sales rank and the
repeated rating from the details and say so beside the detection mode; on an
existing definition the note shall add that the fields are not in use until it
is saved. The
fields shall stay editable; an address shall be filled in once, so a user who
returns it to text mode keeps text mode. Measured on two live book pages: the
visible text differed on each of three fetches and the fields did not.
Verified by: `internal/domain/reading_test.go`,
`internal/infrastructure/extract/reading_test.go`, `reading_test.go`,
`frontend/src/reading.test.ts` and `frontend/src/components/SiteDialog.test.tsx`

### 3.6 Change detection

**FR-060 First success is a baseline.** When a site without a baseline is
checked successfully, the check service shall store the baseline, set the
status Green and raise no alarm.
Verified by: `internal/application/check_test.go::TestFirstSuccessIsBaseline`

**FR-061 Unchanged.** When the hash equals the baseline, the check service shall
record a successful unchanged check and raise no alarm.
Verified by: `internal/application/check_test.go::TestUnchangedRaisesNothing`

**FR-062 Changed.** When the hash differs from the baseline, the check service
shall store a snapshot, record a change event with both hashes, make the new
content the baseline, set a pending alert and raise the configured alarm.
Verified by: `internal/application/check_test.go::TestChangeRaisesAlarm`

**FR-063 No duplicate alarm.** When the same changed content is seen again, the
check service shall raise no further alarm, for scheduled and manual checks.
Verified by: `internal/application/check_test.go::TestRepeatedContentRaisesOnce`

**FR-064 Reversion is a change.** When content returns to an earlier version,
the check service shall record a change event.
Verified by: `internal/application/check_test.go::TestRevertIsAChange`

**FR-065 Change plus failure kept apart.** While a change is pending, when a
later check fails, the site state shall keep the pending alert and report the
failure; the light shall be Red and the change badge shown.
Verified by: `internal/domain/state_test.go::TestFailureDoesNotHidePendingChange`

### 3.7 Status model

**FR-070 Independent fields.** The site state shall hold separately: latest
outcome, latest category, latest HTTP status, baseline presence, pending
alert, enabled and running.
**FR-071 Light.** The domain shall derive the light as Red when the latest check
failed, Grey when there is no baseline, Amber when an alert is pending and
Green otherwise.
Verified by: `internal/domain/state_test.go::TestLight`

**FR-072 Not colour alone.** The interface shall show each light with a text
label and a distinct shape.
Verified by: `frontend/src/components/StatusLight.test.tsx`

### 3.8 Manual refresh and scheduling

**FR-080 One engine.** Manual and scheduled checks shall run through the same
check service.
Verified by: `internal/application/engine_test.go::TestManualAndScheduledShareEngine`

**FR-081 No overlap.** While a site is being checked, the engine shall refuse a
second check of it.
Verified by: `internal/application/engine_test.go::TestNoOverlappingChecks`

**FR-082 Concurrency.** The engine shall run at most 4 checks at once.
Verified by: `internal/application/engine_test.go::TestConcurrencyLimit`

**FR-083 Host spacing.** The scheduler shall not start two checks against one
host less than 10 s apart.
Verified by: `internal/application/scheduler_test.go::TestHostSpacing`

**FR-084 Interval.** The scheduler shall make a site due its interval after its
last check; intervals below 5 min or above 7 days are refused.
Verified by: `internal/domain/schedule_test.go::TestNextDue` and
`internal/domain/site_test.go::TestNormalisedRefuses`

**FR-085 Backoff.** After consecutive failures the scheduler shall double the
wait per failure up to 24 h (up to the site's own interval when that is
longer); after 429 or 503 with Retry-After it shall wait at least that long,
within the same ceiling.
Verified by: `internal/domain/schedule_test.go::TestBackoffAndRetryAfter`

**FR-086 No backlog.** When the application resumes after sleep or a long gap,
each overdue site shall be checked once.
Verified by: `internal/application/scheduler_test.go::TestResumeRunsOverdueOnce`

**FR-087 Pause.** While monitoring is paused, the scheduler shall start no
check; manual refresh shall still work. The paused state survives restart.
Verified by: `internal/application/scheduler_test.go::TestPauseStopsScheduling`

**FR-088 Shutdown.** When the application exits, the scheduler shall cancel
running checks and return before the store is closed.
Verified by: `internal/application/engine_test.go::TestStopCancelsAndWaits`

### 3.9 Notifications

**FR-090 Change alarm.** When a change is recorded on a site with sound on, the
alerter shall play the alarm; with visual on it shall show a desktop
notification naming the site, the time and a summary. On macOS and Linux the
alarm shall play through the first system player present (`afplay`; `paplay`,
`pw-play`, `aplay`, `gst-play-1.0`), which returns when the sound has finished.
Verified by: `internal/application/check_test.go::TestChangeRaisesAlarm` and
`internal/infrastructure/sound/external_test.go`; the notification and the
sound themselves by hand on a real desktop.

**FR-091 No overlapping sound.** When several changes arrive together, the
alerter shall play one alarm at a time.
Verified by: `internal/application/alerts_test.go::TestAlarmsDoNotOverlap`

**FR-092 Repeat until acknowledged (Should).** Where repeat is on, the alerter
shall repeat the alarm until every pending alert is acknowledged.
Verified by: `internal/application/alerts_test.go::TestRepeatStopsOnAcknowledge`

**FR-093 Acknowledge.** When the user acknowledges, the alert service shall
stamp the change event, clear the pending alert, stop any repeat and keep the
baseline and history.
Verified by: `internal/application/alerts_test.go::TestAcknowledgeKeepsHistory`

**FR-094 Failure alarms.** A failed check shall raise no sound. When consecutive
failures reach the site threshold, the alerter shall show one visual warning.
Verified by: `internal/application/check_test.go::TestFailureThresholdWarnsOnce`

**FR-095 Custom sound (Should).** Where a WAV file is chosen, the alerter shall
play it; if it cannot be read, then it shall play the built-in chime and say so.
Verified by: by hand on a real desktop.

### 3.10 History

**FR-100 Check history.** The storage layer shall keep each check's time, status,
duration, outcome, category, detail and hash.
**FR-101 Change history.** The storage layer shall keep each change's previous
and new snapshot, time and acknowledgement time.
Verified by: `internal/infrastructure/storage/history_test.go::TestRecordCheckWritesAll`

**FR-102 Diff.** The domain shall produce a line diff marking added and removed
lines; in field mode it shall list fields whose value changed.
Verified by: `internal/domain/diff_test.go`

**FR-103 Retention.** After each check, the storage layer shall keep the newest
1,000 checks and newest 50 snapshots per site, never deleting the baseline
snapshot or one referenced by a change event still held.
Verified by: `internal/infrastructure/storage/history_test.go::TestRetention`

### 3.11 Application lifecycle

**FR-110 Tray.** On Windows the application shall show a tray icon with Open,
Refresh All, Pause or Resume and Exit. Verified by hand.

**FR-111 Closing the window.** On Windows, when the window's close button is
pressed, the application shall keep the window open and offer Minimise to tray
(which hides it and keeps monitoring) or Quit; Escape or Go back shall leave it
open. While a site, settings, import or delete dialog is open behind the
choice, it shall say that unsaved work is lost by quitting and default to Go
back. Closing the window shall quit on macOS and Linux; on Windows too while
storage could not be opened. Exit and Quit shall stop monitoring.
Verified by: `closing_test.go::TestCloseAsksOnlyWhereTheTrayCanBringTheWindowBack`
and `frontend/src/components/CloseChoiceDialog.test.tsx`; the window itself by
hand; the engine side by `TestStopCancelsAndWaits`.

**FR-112 Sign-in start.** Where sign-in start is on, the application shall
write its one entry: on Windows `HKCU\...\Run\WebVigil` with the quoted path and
`-hidden`, so it waits in the tray; on macOS a LaunchAgent in
`~/Library/LaunchAgents`; on Linux an XDG autostart entry (inside the Flatpak,
one that runs the installed app). Off Windows it opens the window. Turned off,
it shall remove the entry. Settings shall show whether the entry is there,
whoever wrote it: the setup program writes the same entry.
Verified by: `internal/infrastructure/autostart/autostart_windows_test.go`,
`internal/infrastructure/autostart/entry_test.go` and
`internal/application/settings_test.go::TestSettingsTakeTheSignInChoiceFromTheEntry`

**FR-113 Restart.** After a restart the application shall hold the same sites,
baselines, pending alerts and settings.
Verified by: `internal/infrastructure/storage/sites_test.go::TestReopenKeepsState`

**FR-114 Single instance.** When a second copy starts, the first shall come to
the front and the second shall exit. Verified by hand.

**FR-115 No silent death.** If storage cannot be opened, then the application
shall open its window and state the problem with the path of the log.
Verified by hand.

### 3.12 Appearance

**FR-120 Theme.** The application shall offer Light, Dark and System; the choice
survives restart.
Verified by: `internal/application/settings_test.go::TestThemePersists`

**FR-121 Toggle shows its target.** The toolbar toggle shall show the artwork of
the mode it switches to.
Verified by: `frontend/src/components/Toolbar.test.tsx`

**FR-122 Supplied icons.** The toolbar and actions shall use every supplied
image in `assets/` by its file name.
Verified by: `tests/structural/structural_test.go::TestSuppliedIconsAreUsed`

### 3.13 Setup program (Windows)

**FR-130 Per-user install.** The setup program shall install under
`%LOCALAPPDATA%\Programs\WebVigil`, register under `HKCU` with Modify and
Repair; it shall never ask for administrator rights.
**FR-131 Routes.** The setup program shall read the machine once and choose
Install, Update, Downgrade or Manage; Uninstall is reachable from every screen.
**FR-132 Running app.** Before touching any file, the setup program shall ask
the user to close a running WebVigil.
**FR-133 Fenced payload.** The setup program shall refuse an archive entry that
would land outside the install folder.
Verified by: `internal/infrastructure/setup/setup_test.go` (the install paths,
the routes and the payload fence); the registry writes, the running-app check
and the screens by hand.

**FR-134 Uninstall.** Uninstall shall remove the program, its shortcuts and its
registration. With the option to delete the user's data ticked, it shall also
remove the data folder and the window's browser profile
(`%APPDATA%\WebVigil.exe`).
Verified by: `internal/infrastructure/setup/setup_test.go::TestRemoveDataRemovesTheProfileToo`

### 3.14 Help

**FR-140 Help menu.** The toolbar's Help button shall open a menu of Guide, then
About WebVigil, Licence and Check for Updates. It shall open on hover or click
and close on leaving, an outside press, Escape or a choice; Down, Enter or Space
shall open it onto its first item.
Verified by: `frontend/src/components/Menu.test.tsx`,
`frontend/src/components/Toolbar.test.tsx`

**FR-141 Guide.** The Guide shall name every toolbar and row control with the
picture it actually draws, show the status lights from their one definition and
state the rules behind the behaviour as they hold on the platform it runs on.
Verified by: `frontend/src/components/HelpDialogs.test.tsx`,
`frontend/src/platform.test.ts`

**FR-142 About and Licence.** About shall show the name, tagline, version,
author, licence, run log path and open-source credits; Licence shall show the
full bundled licence text titled with the licence's name.
Verified by: `frontend/src/components/HelpDialogs.test.tsx`

**FR-143 Self-reading text.** The Guide, About, Licence and the setup program's
licence shall hold still for 5 s, then read down slowly, hold, rewind and repeat;
any manual input shall suspend the cycle, which resumes where the reader left it.
Verified by: `frontend/src/autoScroll.test.ts`,
`frontend/src/hooks/useAutoScroll.test.tsx`

**FR-144 Update check.** About 3 s after launch, then once a day, the application
shall ask GitHub for the latest published release of the repository the build
names. When it is newer and not skipped, it shall offer Download, Skip This
Version and Later. Help > Check for Updates shall ignore the skip and report
every outcome. A build naming no repository shall report the check as unreachable;
so shall a running version that is not dotted numbers.
Verified by: `internal/application/updates_test.go`,
`internal/infrastructure/update/github_test.go`,
`frontend/src/hooks/useUpdateCheck.test.tsx`

**FR-145 Topmost dialog only.** Escape shall close only the topmost dialog; a
reading surface under another dialog shall freeze rather than read.
Verified by: `frontend/src/components/HelpDialogs.test.tsx`,
`frontend/src/hooks/useAutoScroll.test.tsx`

**FR-146 No pane rings.** No hover or focus rule shall outline a pane or a text
view in the application or the setup program.
Verified by: `tests/structural/focus_test.go`

## 4. Non-functional requirements

**NFR-SEC-001** The application shall send no network request other than the
checks the user configured and the release query of FR-144, which carries no
identifier and nothing about the user's sites. Verified by inspection of every
`http` client use; the release query by
`internal/infrastructure/update/github_test.go::TestTheRequestTargetsTheLatestReleaseEndpointAsJSON`.

**NFR-SEC-002** The interface shall render fetched content as text only, never as
HTML. Verified by `frontend/src/components/DiffView.test.tsx`.

**NFR-MNT-001** Domain and application packages shall hold 100% statement
coverage, gated in `test.ps1`.

**NFR-MNT-002** The layering `UI → Application → Domain ← Infrastructure` shall
be enforced by `tests/structural`.

**NFR-MNT-003** No Go or TypeScript source file shall exceed 400 lines.

**NFR-PRV-001** All data shall stay on the user's machine. WebVigil's own
database and run log live in its per-user data folder: `%LOCALAPPDATA%\WebVigil`
on Windows, else the user configuration folder (`~/Library/Application
Support/WebVigil` on macOS, `~/.var/app/uk.codecrafter.WebVigil/config/WebVigil`
in the Flatpak). On Windows the WebView2 runtime keeps the window's browser
profile, which holds the skipped update version, under `%APPDATA%\WebVigil.exe`.

## 5. Open questions

None at baseline.
