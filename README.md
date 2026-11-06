# <img width="128" height="128" alt="application-icon" src="https://github.com/user-attachments/assets/9dce21f0-8c11-4278-8928-b492c2ea19b2" /> WebVigil

Watch the web. Know what changes.

> **Commercial licences available.** WebVigil is free and open source under
> GPL-3.0. If those terms do not suit what you are building, such as a
> closed-source product, a commercial licence can be bought from me
> separately. It covers my own code; third-party libraries keep their own
> licences. See
> [commercial licensing](https://ernster.dev/commercial-licensing.html).

---

WebVigil is a Windows desktop application that checks the web pages you choose
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
- Monitoring while nobody is signed in to Windows. There is no Windows service;
  checks run while WebVigil is running.
- macOS and Linux. WebVigil is Windows only.

## What it does

- **Reads a page four ways.** Visible text (the default, with scripts, styles
  and hidden elements dropped), the text of chosen CSS selectors, a set of named
  fields taken from selectors or attributes and the raw response. Preview
  extraction shows exactly what will be compared before you save.
- **Ignores what does not matter.** Excluded selectors and text patterns are
  removed before comparison; whitespace can be collapsed and case folded.
- **Compares deterministically.** The content is normalised and hashed; a
  different hash is a change. The first successful check is the baseline and
  raises nothing. Seeing the same changed content again raises nothing more,
  while a return to an earlier version counts as a change.
- **Raises an alarm you choose.** A sound (a built-in chime or your own WAV
  file) and a Windows notification naming the site, either or both. The alarm
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
- **Checks politely.** Each site has its own interval (five minutes at the
  shortest), at most four checks run at once, one host is never asked twice
  within ten seconds, conditional requests save a download when nothing
  changed and repeated failures back off up to a day.
- **Shows the history.** Every check with its outcome, time and status, plus
  every change as a line diff (or a list of changed fields), acknowledged or not.
- **Imports and exports definitions** as a JSON file. An import is previewed
  first: a site you already watch is skipped unless you choose to update it or
  import it as a separate site; nothing is written until you confirm, then all
  of it is written at once.
- **Lives in the tray.** Closing the window hides it and monitoring carries on;
  the tray menu offers Open, Refresh All, Pause or Resume and Exit. It can start
  when you sign in to Windows, waiting in the tray.
- **Explains itself.** Help holds a Guide to every control and rule, About with
  the open-source credits, the full licence and Check for Updates.
- **Tells you when there is a new version.** A few seconds after it starts,
  then once a day, WebVigil asks GitHub for its latest release and offers
  Download, Skip This Version or Later. Check for Updates asks at once and
  always answers.
- **Light, dark or following Windows.**

## What it does not do

- **Two kinds of connection, nothing about you in either.** WebVigil fetches
  the pages you configured, identifying itself as `WebVigil/<version>`, with no
  cookies. The update check is the only other request: an anonymous read of
  WebVigil's public releases list on GitHub, carrying no identifier and nothing
  about your sites. There is no telemetry and there are no accounts. The donate
  button and Download hand a web address to your browser, which is the program
  that connects.
- **No AI.** Changes are found by comparison, never by interpretation.
- **No administrator rights.** Setup installs for your account only.
- **No backup of history.** An export carries site definitions only, never
  snapshots or check history.

## Install

1. Download `WebVigilSetup.exe` from the
   [Releases page](https://github.com/oernster/WebVigil/releases).
2. Run it and choose Install.

Setup puts WebVigil in `%LOCALAPPDATA%\Programs\WebVigil` and lists it under
Settings > Apps. It offers a Start menu entry (ticked), a desktop shortcut
(unticked), starting WebVigil when you sign in, waiting in the tray (ticked) and
opening WebVigil when setup finishes (ticked).

Running setup again offers Update, Go back or Repair, depending on the version
already installed, with Uninstall beside each. Modify in the Apps list opens
the same program. Uninstall removes the program, its shortcuts, its sign-in
entry and its notification registration; your websites and history are kept
unless you tick the box to delete them too.

WebVigil needs the Microsoft WebView2 runtime.

## Where it keeps things

| What | Where |
|---|---|
| The program | `%LOCALAPPDATA%\Programs\WebVigil` |
| Your websites, history and settings | `%LOCALAPPDATA%\WebVigil\webvigil.db` |
| The run log | `%LOCALAPPDATA%\WebVigil\run.log` |
| The setup log | `%LOCALAPPDATA%\WebVigil\setup.log` |
| The window's browser profile, including a skipped update version | `%APPDATA%\WebVigil.exe` |

## Stack

| Part | What |
|---|---|
| Language | Go, with no C compiler needed |
| Window | Wails v2 hosting a React and TypeScript page in WebView2 |
| Storage | SQLite through `modernc.org/sqlite` |
| Page reading | `golang.org/x/net/html` and `andybalholm/cascadia` |
| Windows | `golang.org/x/sys/windows` for the tray, the sound and the registry; Wails for the notifications, with tray balloons when toasts are unavailable |
| The setup program | A second Wails application with a hand-written page |
| Tests | Go's `testing` with gofmt, go vet and staticcheck; Vitest, tsc and ESLint for the page |
| Build | PowerShell: `build.ps1` and `test.ps1` |

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

[REQUIREMENTS.md](REQUIREMENTS.md) is the specification: every requirement with
the test or check that verifies it.

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
