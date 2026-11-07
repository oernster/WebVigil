# Decisions and trade-offs

The deliberate choices WebVigil rests on: what was chosen, what was given up
and why. The detail and the tests that hold each one are in
[ARCHITECTURE.md](ARCHITECTURE.md), [TESTING.md](TESTING.md) and
[REQUIREMENTS.md](REQUIREMENTS.md); [TECH_DEBT.md](TECH_DEBT.md) holds what was
weighed and left.

## The product

### A desktop program, not a service

WebVigil runs on the user's own computer and checks while it is running.

- **Rather than:** a hosted monitor or a background service that runs before
  sign-in.
- **Gains:** no account, no server and nothing about the user's sites leaves
  the machine.
- **Costs:** nothing is checked while the computer is off or signed out.

### Go, Wails and React

A Go core in a Wails window hosting a React page in the system's web engine.

- **Rather than:** shipping a browser engine of its own.
- **Gains:** one small native program per platform; the same core everywhere.
- **Costs:** three web engines to allow for, plus a C compiler for the macOS and
  Linux builds.

## Reading pages

### No script is ever run

Pages are parsed as delivered. A page that builds its content in script is
reported as an extraction failure.

- **Rather than:** a headless browser.
- **Gains:** fast, small and safe: a fetched page cannot act on the machine.
- **Costs:** pages that render only through script cannot be watched.

### Comparison, never interpretation

Content is extracted, normalised and hashed; a different hash is a change.

- **Rather than:** similarity thresholds or AI judgement of what matters.
- **Gains:** a change is a fact that can be shown line by line, never a guess.
- **Costs:** any noise a page adds counts until the user excludes it, with
  ignore rules or by watching named fields only.

### Ready-made readings fill the dialog in

For a page WebVigil knows, such as an Amazon product, the dialog fills in the
fields worth comparing.

- **Rather than:** applying hidden rules to that kind of page silently.
- **Gains:** what the dialog shows is what is compared; every rule can be
  changed.
- **Costs:** an existing site changes only once it is opened and saved.

### The baseline follows the latest content

After a change the new content becomes the baseline, so a page returning to an
earlier version is a change again.

- **Rather than:** comparing against the first version for ever.
- **Gains:** each alarm means "different from what you last accepted".
- **Costs:** a page flipping between two versions alarms on every flip.

### An edit that changes the meaning starts a new baseline

Changing the address, mode, fields or ignore rules asks before saving, then
the next check sets a new baseline without an alarm.

- **Rather than:** comparing content read two different ways.
- **Gains:** no false alarm from the edit itself; history is kept.
- **Costs:** a change made between the edit and the next check goes unreported.

## Being a good visitor

### Honest and polite

WebVigil names itself in every request, sends no cookies, runs few checks at
once, spaces requests to one host, asks conditionally when it can and backs off
after failures or when a server asks it to wait.

- **Rather than:** posing as a browser to get past sites that refuse robots.
- **Gains:** site owners see exactly what is calling and how often.
- **Costs:** some sites refuse it; that is reported rather than evaded.

### Challenges are failures, never puzzles

A CAPTCHA, a robot check or a bot wall is reported as such.

- **Rather than:** solving, bypassing or disguising against it.
- **Gains:** WebVigil never works against a site's own defences.
- **Costs:** a site behind one cannot be watched.

### Failures say why

Every failed check records its category and detail; a run of failures warns
the user once it reaches the site's threshold.

- **Rather than:** retrying quietly and showing only the last good result.
- **Gains:** a site that has stopped answering is never mistaken for one that
  has stopped changing.
- **Costs:** a flaky site produces warnings the user must read.

## Telling the user

### One alarm, until acknowledged

A change shows a notification and sounds a chime made in code; several changes
at once ring once. By default a change waits for acknowledgement.

- **Rather than:** a shipped sound file or overlapping alarms.
- **Gains:** nothing to lose or license; a burst of changes stays one sound.
- **Costs:** the built-in chime cannot be restyled, only replaced by the user's
  own WAV.

### Closing the window

On Windows the close button asks: minimise to the tray or quit. On macOS and
Linux it quits.

- **Rather than:** one behaviour everywhere.
- **Gains:** Windows keeps watching from the tray; elsewhere nothing runs on
  unseen.
- **Costs:** on macOS and Linux the window must stay open for checks to run.

## Data and installation

### Everything local, history bounded

One local database holds the sites, the checks and the snapshots; each site
keeps a bounded history. An export carries the definitions only.

- **Rather than:** unbounded history or a full backup format.
- **Gains:** the database stays small and an export is safe to share.
- **Costs:** old history is pruned; moving machines loses it.

### Installs for one user

The Windows setup and the Flatpak install for the current user only.

- **Rather than:** a machine-wide install.
- **Gains:** no administrator rights at any point.
- **Costs:** each user of a shared computer installs their own copy.

### One anonymous update check

The only request besides the user's own checks reads WebVigil's public release
list, carrying nothing about the user or their sites.

- **Rather than:** an update service or telemetry.
- **Gains:** the user learns of releases; nothing is reported back.
- **Costs:** updating is a download the user runs themselves.
