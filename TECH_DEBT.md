# WebVigil: Technical Debt

Open debt only. Each item names where it is in the code; delete it once fixed.

## 1. Setup's removal weights are estimates

`internal/infrastructure/setup/route.go` weights the install steps by their
median times in `setup.log`. No uninstall has been timed, so the removal steps
still carry starting estimates. Replace them from the step log of a real
uninstall.

## Not debt (do not "fix" these)

- **No script execution, no challenge bypass:** decisions; see
  [DECISIONS-TRADEOFFS.md](DECISIONS-TRADEOFFS.md).
