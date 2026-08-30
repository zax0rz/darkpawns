# Depth handoff — dig command

Date: 2026-08-30
Queue slice: `src/interpreter.c:413`, `dig` / `do_dig`
Starting main: `67e83cb86`
Working branch: `glm/depth-dig`

## Queue decision

The special-procedure inventory and registration-table queue remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was attempted once
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After `dc`, the interpreter table's next un-manifested command
family was `dig`.

## C path and proof

The command table registers `dig` at `src/interpreter.c:413` with
`POS_RESTING` and `LVL_BUILDER` (31), dispatching to `do_dig` in
`src/new_cmds2.c:818-881`. The handler uses `two_arguments`, whose
`one_argument` calls lowercase tokens and skips the C fill list (`in`, `from`,
`with`, `the`, `on`, `at`, `to`). It then applies C `atoi` and `real_room`,
checks missing and unknown rooms, applies the saved `GET_OLC_ZONE`/target-zone
builder gate, accepts both cases of the six direction initials, emits the
invalid-direction warning while continuing with the default north/south pair,
creates bare reciprocal exits, and confirms with the lowercased direction
buffer.

The initial pre-fix `dig-red-depth` vehicle on main reached `Huh?!?` for every
probe while the oracle reached the C format, invalid-direction-plus-success,
missing-room, success, and trailing-argument branches. Additional RED probes
then exposed uppercase confirmation (`N` versus `n`) and fill-word parsing
(`the east the 1205`). The minimal parser fix made all three vehicles GREEN:
`dig-depth`, `dig-permission-depth`, and `dig-current-zone-depth`, each with
`--seed 1 --show-oracle` and no normalized divergence. The permission vehicle
covered same-zone builder success, target-zone refusal, and current-zone
refusal. A focused unit test proves `CREATE`-style bare-exit replacement
clears prior door metadata. No `src/` or `darkpawns-c-oracle/` file was edited.

## Evidence and gates

Added:

- `cmd/dp-oracle-diff/scenarios/dig-depth.txt`
- `cmd/dp-oracle-diff/scenarios/dig-permission-depth.txt`
- `cmd/dp-oracle-diff/scenarios/dig-current-zone-depth.txt`
- `docs/fidelity/depth/dig.tsv`
- `pkg/game/world_write_test.go`
- focused parser tests in `pkg/session/wiz_movement_test.go`

The complete local gates passed:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
git diff --check
```

The frontier after eleven `dig` rows is 1546 total, 1492 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1492/1506 (99.1%). A
same-day earlier handoff recorded a validator failure caused by an accidental
replacement of the existing `at` test file while adding focused tests; that
test was restored before this successful gate run.

This handoff follows R1/R2/R3/R4 and R5/R5e: exact player bytes, command
surface, deterministic proof, no invention, and verification of the actual C
dispatch and call path. R5b/R5c keep the shared C argument parser behavior
explicit rather than treating the first happy path as completion.

## Next queue item

The next un-manifested interpreter-table family after `dig` is `disarm` at
`src/interpreter.c:414`, dispatching to `do_disarm` in the source-order
sweep. Return to clean `main`, pull, run `make fidelity-depth`, reread the
testing guide and newest handoff, map `disarm`'s C path, and start
`glm/depth-disarm`.
