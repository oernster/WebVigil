# WebVigil Testing

## The gate

```powershell
./test.ps1
```

In order: gofmt, go vet, staticcheck, `go test ./...`, the coverage floor, then
the page's `tsc`, ESLint and Vitest. Any failing step stops the run. Read the
exit code: 0 means every step passed. `build.ps1` runs the gate first and has no
switch to skip it; `builddmg.sh` and `build_flatpak.sh` do not run it.

A bare `go test ./...` checks no formatting, applies no floor and runs only the
analysers `go test` carries.

## Coverage floor

`internal/domain` and `internal/application` are held at 100%: both run with no
network, disk, clock or desktop, so a missed line there is a decision nobody
made. The gate prints the uncovered functions when either falls short.
Infrastructure, the facade and the page are tested but not floored; their
Windows halves need a real desktop.

## What is tested and how

| Layer | How |
|---|---|
| Domain | Table tests over pure functions |
| Application | Use cases against hand-written fakes (`fakes_test.go`, `fakes_io_test.go`); no mock library |
| Storage | A real SQLite file in a temporary folder |
| Fetch | `net/http/httptest` servers on loopback |
| Extract | HTML fixtures in the test files, including an Amazon-shaped page |
| Facade | Plain Go tests of the bindings that need no window (`closing_test.go`, `exchange_filter_test.go`, `reading_test.go`) |
| Page | Vitest under jsdom with Testing Library; dialogs are driven through the real components |

A page test that needs Go puts a stand-in for the one binding it uses on
`window.go.main.App`, where `api.ts` looks (see `SiteDialog.test.tsx`).

## Structural tests

`tests/structural/` reads the source rather than running it: layering, the
clock rule, the composition root, the network list, the page's own requests,
the module-size cap, the wire contract, the icons, user-specific data and the
focus-ring rules. The table is in
[ARCHITECTURE.md](ARCHITECTURE.md). `TestThePaneScanBites` plants a violation to
prove the focus scan fails on one.

## Proving a test bites

A new test counts only once it has failed against a planted defect. Plant the
defect, run the one test, read its exit code, restore the file.
