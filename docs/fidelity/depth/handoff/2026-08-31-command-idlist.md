# Depth-fidelity handoff: `idlist`

Date: 2026-08-31

## Queue position and frontier

This session continued the source-order `src/interpreter.c` command-family sweep
after the special-procedure inventory and the one-time `objmagic.sleep-entry-gates`
attempt. `idlist` is the command-table row at `src/interpreter.c:512`, immediately
after the already-covered `inventory`, `idea`, and `ident` rows. Before this slice,
the frontier was 2,260 total cases: 2,200 proven/delegated, 16 blocked, and 44
excluded. The five new manifest rows bring it to 2,265 total: 2,205 proven/delegated,
16 blocked, and 44 excluded (2,205/2,221 actionable, 99.3%). The howl handoff PR
(#953) remains open and not green because its hosted checks did not fire after the
single permitted retry; it remains unmerged.

The next source-order unclaimed command is `imotd` at `src/interpreter.c:513`.

## C call path and branch inventory

The C command table registers `idlist` as `POS_DEAD`, handler `do_idlist`, with the
`LVL_GRGOD` (38) level gate. `src/act.wizard.c:3575-3693` ignores the command
argument, opens the fixed relative filename `object_idlist`, walks every object
prototype in C load order, and writes the object short description, item type,
affected-bit and extra-flag lists, weight/cost, type-specific spell/charge/damage/
armor details, object affects, and a trailing blank line. Staff and wand entries use
`CASTING_EQ`; scroll/potion spell names are concatenated exactly as authored. The
only ordinary player-visible success byte sequence is `Ok. Id list complete.`;
open failure emits `Could not open id list file, cannot complete operation!`.
Arguments must not select a different path or add a player-facing object count.

The Go call path previously added an implementor-only handler check, accepted an
optional filename, wrote a different compact report under `data/`, and emitted a
counted completion line. The level-38 vehicle proved the gate divergence; the
implementor vehicle proved the argument/output divergence. The report body is not
visible through the telnet oracle, so its exact C branch and formatting are covered
by a focused unit golden.

## RED/GREEN result

The two oracle vehicles were RED on main, then GREEN after the confirmed fixes:

- `idlist-gate-depth` lowers the first player to level 38 and proves the registered
  C authority gate.
- `idlist-depth` proves implementor success and that trailing arguments are ignored.
- Seeds 1, 2, 3, 5, and 8 are green for both vehicles.
- `TestWriteObjectIDListMatchesCReportShape` pins the report order, table names,
  spell concatenation, charge grammar, arithmetic, affect branches, flag rendering,
  and blank-line shape; `TestIDListBitArrayUsesUndefinedForOutOfTableBits` covers
  the C bit-table fallback.

No `src/` or `darkpawns-c-oracle/` files were edited. The implementation follows
R1/R2/R3/R4 and the C-first call-path audit follows R5e/R5c.

## Verification

The complete local gates passed after the final change: `make fidelity-depth`,
`go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`,
and a clean `gofumpt -l .` check. Feature PR #962 was merged only after the
post-fix hosted test, lint, and security jobs were green; build/deploy were skipped
by the workflow. The first hosted run's security failure was G115 in the helper's
signed-int conversion; the explicit 32-position scan fixed it and the rerun passed.

## Next session

Start from clean `main`, pull, rerun `make fidelity-depth`, reread the depth-testing
guide and newest handoff, then take `imotd` in source-table order. Keep the open,
not-green howl PR unmerged.
