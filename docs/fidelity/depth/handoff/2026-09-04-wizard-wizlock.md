# Wizard `wizlock` audit — 2026-09-04

## Scope

This slice closes the source-order wizard surface at `src/act.wizard.c`
`do_wizlock` (1769–1801), after the completed VNUM/STAT/VSTAT and switch
slices. The command-table entry is `interpreter.c:834` and the registered Go
handler is `pkg/session/wiz_system.go:211`. The neighboring `do_date` path is
not re-counted here.

The C branch families covered are the current-state query, valid numeric
restrictions for open/closed/level-qualified modes, C `atoi` parsing, negative
and above-owner invalid values, and the `one_argument` trailing-input
boundary. Numeric `game_restrict` state and exact `do_wizlock` output are part
of the player-visible/state contract (R1/R2/R3/R5e). Login gating remains a
separate shared surface because its C call sites must be verified and wired
as a class (R4/R5b/R5c/R5e).

## C authority

`src/act.wizard.c:1769-1801` calls `one_argument`, parses the first token with
C `atoi`, rejects values below zero or above `GET_LEVEL(ch)`, stores the
integer in `game_restrict`, and renders separate open, closed-to-new-players,
and level-qualified status bodies. `src/interpreter.c:834` registers the
command at `LVL_IMPL-1`, so the handler must not add a stricter internal gate.

The login call sites are `src/interpreter.c:1832-1847` for new-character
creation and `src/interpreter.c:1906-1910` for known-player password login.
They use the integer restriction threshold differently; this slice records
that shared integration as blocked rather than collapsing it to the old
boolean-only behavior (R4/R5b/R5c/R5e).

## Pre-fix result — 2026-09-04

The C-first vehicle `cmd/dp-oracle-diff/scenarios/wizard-wizlock-depth.txt`
was red on the fresh `origin/main`-derived port at `DP_SEED=1` and `DP_SEED=2`.
C distinguished restriction levels and C `atoi`'s nonnumeric-prefix behavior;
Go collapsed all nonzero values to a boolean, emitted invented mutation
messages, rejected nonnumeric input, and used a different above-level error.
The no-argument query and negative invalid message happened to match, but the
complete state-transition vehicle diverged (R1/R2/R3/R5e).

## Implementation and proof — 2026-09-04

Commit `ef418e60c` adds an integer `wizlockLevel` alongside the compatibility
boolean, uses the existing C-compatible `cAtoi`, removes the extra Go-only
`LVL_IMPL` gate, preserves invalid state, and renders the exact C status
families. The vehicle is green with `--show-oracle` at both seeds 1 and 2:

```text
result: no normalized divergence
```

The oracle blocks cover all 12 annotated command cases: open, level-one, and
level-two queries and mutations; reset to open; nonnumeric, negative, and
above-owner values; one-argument trailing input; and the final level-three
query. The manifest records 13 `do_wizlock` rows: 12 proven/delegated and one
blocked login-threshold row.

Verification completed on this slice:

- `make fidelity-depth`: 4231 total, 4130 proven/delegated, 50 blocked, 51 excluded.
- `go build ./...` and `go vet ./...`: pass.
- `go test ./...`: pass.
- `golangci-lint run ./...`: pass with 0 issues.
- `gofumpt -l .`: clean.
- high-severity `gosec`: pass.

No C or oracle-tree files were changed. The untracked
`website/static/images/` directory remains outside this slice.
