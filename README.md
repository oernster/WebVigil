# <img width="128" height="128" alt="application-icon" src="https://github.com/user-attachments/assets/9dce21f0-8c11-4278-8928-b492c2ea19b2" /> WebVigil

Watch the web. Know what changes.

> **Commercial licences available.** WebVigil is free and open source under
> GPL-3.0. If those terms do not suit what you are building, such as a
> closed-source product, a commercial licence can be bought from me
> separately. It covers my own code; third-party libraries keep their own
> licences. See
> [commercial licensing](https://ernster.dev/commercial-licensing.html).

---

WebVigil is a desktop application for Windows, macOS and Linux that checks the web pages you choose
on a schedule, compares what it reads against the last version it accepted and
tells you when the meaningful content changes. When a check fails it says why,
in a category you can act on, rather than going quiet. Everything it records
stays on your computer.

Site: [ernster.dev/WebVigil](https://ernster.dev/WebVigil/)

## Who it is for

- Anyone who keeps reopening the same pages to see whether something has moved:
  a price, a stock notice, a policy page, a release list, a timetable.
- People who want to know exactly what changed and when, line by line.
- Anyone who would rather a failed check said "the server answered 503" than
  showed an old page as if nothing were wrong.

## Who it is not for

- Pages that build their content in the browser with JavaScript. WebVigil reads
  the HTML the server sends; a page with nothing in it until script runs is
  reported as an extraction failure, never worked around.
- Pages behind a login. WebVigil stores no passwords, cookies or sessions.
- Anyone who needs to get past CAPTCHAs or bot checks. WebVigil recognises a
  challenge page and reports it; it never tries to solve or disguise itself.
- Monitoring while nobody is signed in. There is no background service; checks
  run while WebVigil is running.

## What it does

- **Reads a page four ways.** Visible text (the default, with scripts, styles
  and hidden elements dropped), the text of chosen CSS selectors, a set of named
  fields taken from selectors or attributes and the raw response. Preview
  extraction shows exactly what will be compared before you save.
- **Ignores what does not matter.** Excluded selectors and text patterns are
  removed before comparison; whitespace can be collapsed and case folded.
- **Knows Amazon product pages.** Amazon reshuffles adverts, menus and page data
  on every load, so the whole page never reads the same twice. Enter an Amazon
  product address and the dialog fills in fields for the title, author, formats
  and prices, rating, review count, description, cover and product details
  (without the sales rank, which moves by the hour). They stay visible and
  editable; a robot-check page served in place of the product reads as a failed
  check, not a change.
- **Compares deterministically.** The content is normalised and hashed; a
  different hash is a change. The first successful check is the baseline and
  raises nothing. Seeing the same changed content again raises nothing more,
  while a return to an earlier version counts as a change.
- **Raises an alarm you choose.** A sound (a built-in chime or your own WAV
  file) and a desktop notification naming the site, either or both. The alarm
  can repeat until you acknowledge it; WebVigil can also bring its window
  forward.
- **Keeps a change waiting for you.** A site stays amber until you acknowledge
  the change; a failure on a later check turns it red without hiding the
  change still waiting.
- **Says why a check failed.** DNS, refused or timed-out connections, TLS, each
  HTTP status that matters, redirect loops, unsupported content, oversized
  responses, a missing selector, empty content or a challenge page. Failures
  make no sound; one warning appears once a site has failed as many times in a
  row as its threshold.
- **Checks politely.** Each site has its own interval (from five minutes to
  seven days), at most four checks run at once, one host is never asked twice
  within ten seconds, conditional requests save a download when nothing
  changed and repeated failures back off up to a day (or the site's own
  interval when that is longer).
- **Shows the history.** Every check with its outcome, time and status, plus
  every change as a line diff (or a list of changed fields), acknowledged or not.
- **Imports and exports definitions** as a JSON file. An import is previewed
  first: a site you already watch is skipped unless you choose to update it or
  import it as a separate site; nothing is written until you confirm, then all
  of it is written at once.
- **On Windows, lives in the tray.** Closing the window asks whether to
  minimise WebVigil to the tray (where monitoring carries on) or to quit; the
  tray menu offers Open, Refresh All, Pause or Resume and Exit. It can start
  when you sign in, waiting in the tray.
- **On macOS and Linux, watches while its window is open.** Closing the window
  quits. It can start when you log in, with its window open.
- **Explains itself.** Help holds a Guide to every control and rule, About with
  the open-source credits, the full licence and Check for Updates.
- **Tells you when there is a new version.** A few seconds after it starts,
  then once a day, WebVigil asks GitHub for its latest release and offers
  Download, Skip This Version or Later. Check for Updates asks at once and
  always answers.
- **Light, dark or following the system.**

## What it does not do

- **Two kinds of connection, nothing about you in either.** WebVigil fetches
  the pages you configured, identifying itself as `WebVigil/<version>`, with no
  cookies. The update check is the only other request: an anonymous read of
  WebVigil's public releases list on GitHub, carrying no identifier and nothing
  about your sites. There is no telemetry and there are no accounts. The donate
  button and Download hand a web address to your browser, which is the program
  that connects.
- **No AI.** Changes are found by comparison, never by interpretation.
- **No administrator rights.** The Windows setup and the Flatpak both install
  for your account only.
- **No backup of history.** An export carries site definitions only, never
  snapshots or check history.

## Install

Each release on the [Releases page](https://github.com/oernster/WebVigil/releases)
carries one download per platform.

| Platform | Download | Needs |
|---|---|---|
| Windows | `WebVigilSetup.exe`: run it and choose Install | Windows 10 or 11 on x64 with the Microsoft WebView2 runtime |
| macOS | `WebVigil.dmg`: open it and drag WebVigil into Applications | Apple Silicon |
| Linux | `webvigil.flatpak`: `flatpak install --user webvigil.flatpak` | Flatpak; the GNOME runtime it needs comes from Flathub |

### Windows setup

Setup puts WebVigil in `%LOCALAPPDATA%\Programs\WebVigil` and lists it under
Settings > Apps. It offers a Start menu entry (ticked), a desktop shortcut
(unticked), starting WebVigil when you sign in, waiting in the tray (ticked) and
opening WebVigil when setup finishes (ticked).

Running setup again offers Update, Go back or Repair, depending on the version
already installed, with Uninstall beside each. Modify in the Apps list opens
the same program. Uninstall removes the program, its shortcuts, its sign-in
entry and its notification registration; your websites and history are kept
unless you tick the box to delete them too.

## Where it keeps things

Your websites, history and settings are `webvigil.db`; the run log is `run.log`.
Both sit in WebVigil's data folder:

| Platform | Data folder | Sign-in entry |
|---|---|---|
| Windows | `%LOCALAPPDATA%\WebVigil` | `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\WebVigil` |
| macOS | `~/Library/Application Support/WebVigil` | `~/Library/LaunchAgents/WebVigil.plist` |
| Linux (Flatpak) | `~/.var/app/uk.codecrafter.WebVigil/config/WebVigil` | `~/.config/autostart/webvigil.desktop` |

On Windows the program itself is in `%LOCALAPPDATA%\Programs\WebVigil`, setup
logs to `setup.log` in the data folder and the window's browser profile,
including a skipped update version, is `%APPDATA%\WebVigil.exe`.

## Stack

| Part | What |
|---|---|
| Language | Go; the Windows build needs no C compiler, while macOS and Linux need one for the web view |
| Window | Wails v2 hosting a React and TypeScript page in WebView2 on Windows, WKWebView on macOS and WebKitGTK on Linux |
| Storage | SQLite through `modernc.org/sqlite` |
| Page reading | `golang.org/x/net/html` and `andybalholm/cascadia` |
| Windows | `golang.org/x/sys/windows` for the tray, the sound and the registry; Wails for the notifications, with tray balloons when toasts are unavailable |
| macOS and Linux | The system's own player for the alarm (`afplay`; on Linux the first of `paplay`, `pw-play`, `aplay` and `gst-play-1.0`); a LaunchAgent or an XDG autostart entry for sign-in start; Wails for the notifications |
| The setup program | Windows only: a second Wails application with a hand-written page |
| Tests | Go's `testing` with gofmt, go vet and staticcheck; Vitest, tsc and ESLint for the page |
| Build | PowerShell `build.ps1` and `test.ps1` on Windows; bash `builddmg.sh` on macOS and `build_flatpak.sh` on Linux |

## Build and test

```powershell
./build.ps1
```

That regenerates the icons, runs the whole test gate, then builds
`build/bin/WebVigil.exe` and `dist-installer/WebVigilSetup.exe`.
`./build.ps1 -SkipInstaller` stops after the application. The gate alone is:

```powershell
./test.ps1
```

On an Apple Silicon Mac, `bash builddmg.sh` builds, signs and notarises
`WebVigil.dmg`. On Linux, `bash build_flatpak.sh` builds `webvigil.flatpak` and
installs it for the current user; `bash cleanup_flatpak.sh` uninstalls it and
removes the Flatpak build files. Neither script runs the gate, which is the
Windows build's.

[REQUIREMENTS.md](REQUIREMENTS.md) is the specification: every requirement with
the test or check that verifies it. [ARCHITECTURE.md](ARCHITECTURE.md),
[TESTING.md](TESTING.md), [DEVELOPMENT.md](DEVELOPMENT.md),
[DECISIONS-TRADEOFFS.md](DECISIONS-TRADEOFFS.md) and
[TECH_DEBT.md](TECH_DEBT.md) cover the design, the tests, working on the source,
the choices behind it and the debt still open.

## Supporting the project

The strip along the foot of the window carries a donate button. It hands a
PayPal page to your browser; WebVigil itself sends nothing.

WebVigil is free and stays free. Donations support maintenance and continued
development. Nothing is withheld behind one: there is no paid tier, no licence
key and no feature a donation unlocks.

<a href="https://www.paypal.com/ncp/payment/4JDW9BNNG97CE"><img src="docs/donate.png" alt="Donate to WebVigil" width="120"></a>

## Licence

Distributed under the GNU General Public License v3.0; see [LICENSE](LICENSE).

A commercial licence for my own code is also available, separately from the
open-source licence: see
[commercial licensing](https://ernster.dev/commercial-licensing.html).

### Open source credits

Compiled into WebVigil, as its About dialog states:

- **Go**, **golang.org/x/net** and **golang.org/x/sys**: BSD 3-Clause.
- **Wails**: MIT.
- **React** and **Vite**: MIT.
- **andybalholm/cascadia**: BSD 2-Clause.
- **modernc.org/sqlite**: BSD 3-Clause.
