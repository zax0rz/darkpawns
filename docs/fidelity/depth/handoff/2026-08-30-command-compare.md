# Depth handoff — compare command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:393`, `compare` / `do_compare`  
Starting main: `50431be27`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`,
`src/spec_procs2.c`, and `src/spec_procs3.c`, including the active
registration tables, remains exhausted. The blocked
`objmagic.sleep-entry-gates` row was already attempted through the cast-sleep
outlaw/reagent vehicle and remains blocked; it was not repicked. After the
merged `commands` slice, the interpreter sweep selected `compare` at line 393
in table order. The `do_action` social entries before this row remain owned by
the existing social family manifest and were not repicked individually.

## C path and proof

The command table registers `compare` at `src/interpreter.c:393` for
`POS_STANDING` and routes it to `do_compare` in `src/new_cmds.c:1952-2070`.
The handler computes APPRAISE-based probability without gating entry on the
skill, rejects blindness first, resolves both names from carrying only, then
checks active fighting, same object, object type, armor wear-slot identity,
and the weapons/armor type class. The terminal comparison path consumes the
C percent draw, computes weapon or armor value differences, adds C's failure
noise or calls `improve_skill`, selects one of seven threshold messages, and
capitalizes the result.

The existing rejection vehicle was extended with deterministic disposable
objects: a second weapon, two same-slot body armors, a different-slot helmet,
and a second container. The final `compare --seed 1 --show-oracle` vehicle is
GREEN for missing objects, same object, type mismatch, weapon result, armor
result, armor-slot mismatch, and non-comparable same-type objects. Seeds 2, 3,
5, and 8 are also GREEN, establishing the weapon and armor random paths. The
existing `TestCmdCompare_Fighting` and new `TestCmdCompare_Blind` cover the two
early gates that are not useful to stack into the object vehicle. No confirmed
Go divergence remained in this slice, and neither `src/` nor
`darkpawns-c-oracle/` was edited.

The earlier implementation fixes, already present on `main`, removed a
non-C skill gate and stub output; this slice adds the missing durable depth
manifest and completes the live proof matrix without changing that confirmed
behavior.

## Changes

- Extend `cmd/dp-oracle-diff/scenarios/compare.txt` with guaranteed-load weapon,
  armor, helmet, and container fixtures plus live result/gate probes.
- Remove the stale oracle-unverified note from `pkg/game/skill_advanced.go`.
- Add `TestCmdCompare_Blind` to the command wrapper tests.
- Add the nine rows in `docs/fidelity/depth/compare.tsv` and this dated
  handoff.

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

The refreshed frontier is 1,462 total cases: 1,408 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,408/1,422 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the command dispatch, early-gate
ordering, object audience/state, value arithmetic, and random path follow
R5b/R5c.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table family after
`compare`.
