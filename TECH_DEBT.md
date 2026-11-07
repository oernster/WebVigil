# WebVigil: Technical Debt

Open debt only. Each item names where it is in the code; delete it once fixed.

## 1. Settings cannot see the sign-in entry setup wrote

The `AutoStart` port can only write the entry (`Set`), so the Settings tick
shows the stored choice. When setup has written the Run key, Settings can show
"Start at sign-in" unticked. Fix: give the port a read and show what is there.

## 2. The user-data scan skips the setup page

The structural walk skips every folder named `dist`, so the committed
`installer/frontend/dist/index.html` is outside `TestNoUserSpecificData`.

## 3. Setup's progress weights are estimates

`internal/infrastructure/setup/route.go` weights each step by a starting
estimate. `setup.log` now records real timings (unpacking about 75 ms,
shortcuts 1.7 to 3.1 s); the weights should come from them.

## 4. Uninstalling with "delete data" leaves the browser profile

`Uninstall` in `installer/app.go` removes only the data folder, so the
window's WebView2 profile in `%APPDATA%\WebVigil.exe` survives it.

## Not debt (do not "fix" these)

- **No script execution, no challenge bypass:** decisions; see
  [DECISIONS-TRADEOFFS.md](DECISIONS-TRADEOFFS.md).
