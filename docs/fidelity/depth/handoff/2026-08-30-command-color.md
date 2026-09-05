# Depth handoff — color command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:388`, `color` / `do_color`  
Starting main: `686f37df1`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`, `src/spec_procs2.c`,
and `src/spec_procs3.c`, including the active registration tables, remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was already attempted
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After the merged clear-screen family, the interpreter sweep selected
`color` at line 388 in table order.

## C path and proof

The command table registers `color` at `src/interpreter.c:388` for
`POS_DEAD`. `src/act.informative.c:2462-2498` implements `do_color`: NPCs
return silently; `any_one_arg` lowercases the first token; `on` recursively
selects `complete`; and `search_block(..., FALSE)` accepts prefixes in the
ordered `off`, `sparse`, `normal`, `complete` table. Valid input clears and
sets `PRF_COLOR_1/2`, reports the resulting level, and ignores trailing
arguments. Invalid input emits the exact usage line without changing state.
The corresponding Go path is `pkg/session/act_informative.go`.

The RED vehicle on main exposed Go's exact-match parser rejecting the C-valid
one-letter prefixes `s`, `n`, `c`, and `o`, while preserving the C state and
message behavior for exact names, `on`, invalid input, and trailing arguments.
The fix changes only the level lookup to ordered prefix matching. The final
vehicle is GREEN with `--show-oracle`; the normalizer intentionally strips ANSI
transport presentation, so no speculative color-rendering change was made.
Neither `src/` nor `darkpawns-c-oracle/` was edited.

## Changes

- `pkg/session/act_informative.go`: mirror C's case-insensitive ordered prefix
  matching for color levels.
- `cmd/dp-oracle-diff/scenarios/color-depth.txt`: add nine durable cases for
  no-argument reporting, all valid prefixes, the `on` alias, state reports,
  invalid input, and ignored trailing arguments.
- `docs/fidelity/depth/color.tsv`: add the nine manifest rows.
- Add this dated handoff.

`color-depth --seed 1 --show-oracle` is GREEN, and the updated manifest reports
`do_color: 9/9`.

## Gates and frontier

The following all pass on the final tree:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
```

The refreshed frontier is 1,439 total cases: 1,385 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,385/1,399 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the parser/state path and proof
audit follow R5b/R5c. Player-facing text remains grounded in the C oracle.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `color`.
