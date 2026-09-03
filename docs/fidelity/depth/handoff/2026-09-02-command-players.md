# Depth handoff — players

Date: 2026-09-02
Branch: `glm/depth-players`
Feature PR: #1233 (merged green)
Feature commit: `9cbd6dbcf`
Main merge: `3c2c478e3`

## Queue position

The special-procedure inventory is exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, was attempted through the cast-sleep
outlaw/reagent vehicle and remains blocked for the unreachable entry gates.
After refreshing `main`, pulling with `--ff-only`, running `make fidelity-depth`,
reading `DEPTH_TESTING.md`, and reviewing the dated handoffs, the next live
interpreter-table family in source order was `players` at
`src/interpreter.c:606`. Social aliases and previously claimed shared
families were excluded by their explicit handoff claims; the stale `show`
handoff did not override the live manifest/source sweep.

Pre-slice frontier: 3,635 total, 3,534 proven/delegated, 48 blocked, 53
excluded.

Post-slice frontier: 3,638 total, 3,537 proven/delegated, 48 blocked, 53
excluded; actionable completion is 3,537/3,585 = 98.7%.

The next session must refresh `main`, rerun the frontier, reread the guide and
newest handoff, then repeat the interpreter-table sweep while honoring explicit
family claims in existing handoffs. Do not repick stale claims or already
covered shared families.

## C call path and observable contract

The command table registers:

```c
{ "players"  , POS_DEAD, do_gen_ps   , LVL_GRGOD, SCMD_PLAYER_LIST }
```

at `src/interpreter.c:606`; `SCMD_PLAYER_LIST` is 12 in
`src/interpreter.h`. The `do_gen_ps` branch at
`src/act.informative.c:2181-2195` sends `A list of registered players:\r\n`,
formats each player name with `%-20.20s`, emits three names per line, and
terminates with `\r\n`. Trailing command arguments are ignored.

The player table path in `src/db.c:2653-2675` appends new entries in lowercase
and `build_player_index` lowercases loaded names. The Go command already had the
same command gate and line layout, but its no-DB oracle harness path
dereferenced a nil database and returned EOF. The confirmed fix uses the active
in-memory world player list when no database is configured (or when the DB list
fails), lowercases names to match C's table, sorts them deterministically, and
retains the C three-column/20-character formatting. No `src/` or oracle-tree
file was edited.

## Proof artifacts

Scenario: `cmd/dp-oracle-diff/scenarios/players-depth.txt`.

The scenario covers the `LVL_GRGOD`/`POS_DEAD` entry gate, the exact list header
and online name, and ignored trailing arguments using the empty-player-table
fixture.

Manifest: `docs/fidelity/depth/players.tsv` (3 rows).

Focused test: `pkg/session/players_depth_test.go`, covering the registration
gate and the C player-table shape without a database, including three-column
line formatting.

With `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`, seed 1 with
`--show-oracle` and seeds 2, 3, 5, and 8 produced
`result: no normalized divergence`.

## Gates and review

All local gates passed on the feature branch:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` (clean)
- `git diff --check`

PR #1233's hosted lint, security, and test checks were green; build/deploy were
correctly skipped. The PR was self-merged only after the checks were green.

This slice follows R1 (player-facing bytes), R2 (the registered command
surface), R3 (deterministic list ordering), R4 (no invented behavior), R5
(actual C call path), R5e (the C source wins), and R5b/R5c (shared storage and
family claims must be audited at the class level rather than repicked).
