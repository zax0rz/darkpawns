# Depth handoff — dc command

Date: 2026-08-30
Queue slice: `src/interpreter.c:409`, `dc` / `do_dc`
Starting main: `f2fb84862`

## Queue decision

The special-procedure inventory and registration-table queue remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was attempted once
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After the merged `dark` slice, the interpreter table's next
un-manifested command family in source order was `dc`.

## C path and proof

The command table registers `dc` at `src/interpreter.c:409` with `POS_DEAD`
and `LVL_IMMORT+1` (32), dispatching to `do_dc` in
`src/act.wizard.c:1736-1766`. The handler runs `one_argument`, applies C's
`atoi` semantics, finds the descriptor by `desc_num`, emits the exact usage or
missing-connection bytes, refuses an equal-or-higher logged-in target, or sets
`close_me` and emits the numbered confirmation. The descriptor number is
allocated by `src/comm.c:1466,1598-1600`, incrementing from 1 and wrapping at
999.

The pre-fix `dc-red-depth` vehicle confirmed RED: C answered
`Usage: DC <connection number> (type USERS for a list)` for `dc all`, while
the old Go path invented `Disconnected 1 players.` and closed the peer. The
committed `dc-depth` vehicle then proved the no-argument and nonnumeric usage
arms, missing descriptor, exact level-32 entry boundary, equal-level refusal,
and lower-level peer closure at descriptor #2. Seed 1 finished with no
normalized divergence. No `src/` or `darkpawns-c-oracle/` file was edited.

The Go fix assigns C-style connection numbers at transport-session creation
for telnet and WebSocket sessions, uses the C decimal-prefix parser, changes
the handler gate to `LVL_IMMORT+1`, removes the invented name/`all` behavior,
and emits the C bytes. The differential harness now preserves a final
audience block when that final command intentionally closes the audience
connection; earlier unexpected EOFs remain errors.

## Evidence and gates

Added:

- `cmd/dp-oracle-diff/scenarios/dc-depth.txt`
- `docs/fidelity/depth/dc.tsv`
- `pkg/session/dc_depth_test.go`

The local gates for this slice are recorded after running:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
git diff --check
```

The frontier after adding six proven rows is 1535 total cases, 1481
proven/delegated, 14 blocked, and 40 excluded; actionable completion is
1481/1495 (99.1%). This slice follows R1/R2/R3/R4 and R5/R5e: exact player
bytes, command-surface fidelity, deterministic proof, no invention, and
verification of the actual C descriptor and dispatch call paths.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `dc`.
