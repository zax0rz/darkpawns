# Modernization Phase 4 — post-merge census and item 4.5 redo

Date: 2026-09-06
Base: PR #1395 merge `7198cb05c0e923a7ea3d5dbac34f1f77d4c0e43f`
Branch: `codex/fidelity-census-fixes`

## Census result

The full corpus was run with the real C oracle after the census fixes and the
item-4.5 redo:

```text
TMPDIR=/home/zach/dp-census-final-tmp \
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
ORACLE_REGRESSION_JOBS=4 ORACLE_REGRESSION_SEED=1 \
ORACLE_REGRESSION_TIMEOUT=240s make oracle-regression
```

Authoritative result (934 scenarios, 2026-09-06 06:20–08:19 EDT):

```text
passed=922 expected=9 unpinnable=1 stale=0 failed=0 infra=2 timed_out=0
```

The preserved worker evidence is under
`/home/zach/dp-census-final-evidence/`; the run summary is
`/home/zach/dp-census-final-evidence/census-output.log`.

The two infrastructure classifications are unchanged, run-varying cases:

- `last-depth`: the C oracle exposes a live saved-player row while the Go
  process reports no such player in the same seeded run.
- `room-desc-exits`: the only mismatch is the Go-generated object label suffix
  on the south-room board; the C/Go shape is unstable across attempts.

`accuse-noarg-depth` remains the single human-clearance `UNPINNABLE` case.
The nine `EXPECTED` cases remain pinned by the existing divergence ledger.

## Item 4.5 redo

The position handlers were table-driven in a fresh patch covering
`DoStand`, `DoSit`, `DoRest`, and `DoSleep`. The table retains the C-specific
mounted gate/dismount paths, per-arm room hide flags, exact text, and the
default-arm state transitions. A first draft incorrectly used `POS_SITTING`
for normal `DoRest` arms; the focused oracle matrix caught that regression.
The corrected table carries the next position per arm, including C's distinct
`POS_RESTING` normal arms and `POS_SITTING` default arm.

Focused item-4.5 matrix:

```text
position-basic, position-fighting, position-mounted-gates, position-room,
position-gates, position-wake-target, sleep-spell-depth
7 passed, 0 failed, 0 infra, 0 timed out
```

The standard repository gates passed: `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and
`git diff --check`.

## Ledger reconciliation

The seven dangling proof labels were removed rather than minting uncheckable
scenario evidence: six `show-depth-red` rows in `show.tsv` and
`shop-live-inventory-lane-b` in `shop.tsv` now use proof `-`, retain their
direct C citations, and explicitly remain `blocked`. The generators report
zero unresolved proof labels; the existing pinned divergence file remains
intact and passes `--check-pins`.

Rules applied: R1/R3 for byte and ordering parity, R4 for not inventing proof
vehicles, and R5e for tracing the reachable C call path.

## Phase-4 audit state

PRs #1390, #1391, #1392, and #1393 are merged. Superseded PR #1394 was
closed after this item-4.5 redo; the corrected implementation is committed as
`63e5bb6d0` on `codex/fidelity-census-fixes`.
