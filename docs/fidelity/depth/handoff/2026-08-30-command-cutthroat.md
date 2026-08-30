# Depth handoff — cutthroat command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:403`, `cutthroat` / `do_cutthroat`  
Starting main: `c15f21c30`

## Queue decision

The special-procedure inventory and registration-table queue remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was attempted once
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. The interpreter sweep selected `cutthroat`, the first un-manifested
command after the merged `curtsey` slice.

## C path and proof

The command table registers `cutthroat` at `src/interpreter.c:403` for
`POS_FIGHTING` with minimum level 1 and dispatches to `do_cutthroat` in
`src/new_cmds.c:552-655`. The call path is C-first: `GET_SKILL` precedes
`one_argument`, target lookup precedes the self check, weapon value[3] must be
11, and mount/peaceful/low-level-PC/existing-AFF_CUTTHROAT/fighting gates
follow in source order. Only then does C draw `number(1,101)`; an immortal
attacker forces `prob=102`, while an immortal victim overwrites it with -1.

The success arm calls `affect_join` with duration level*2, modifier -2,
`APPLY_HITROLL`, and `AFF_CUTTHROAT`, emits its literal actor/victim preamble,
improves the skill, then calls `damage(..., level/2, SKILL_CUTTHROAT)`, which
uses message set 143, before `WAIT_STATE` two violence pulses. The failure arm
emits its literal lunge preamble, calls ordinary `hit()` with the wielded
weapon, and applies the same wait. The Go pipeline now preserves these
separate pre-damage-improve, set-143 damage, and synchronous ordinary-hit
paths.

The initial depth vehicle exposed the prior invented Go messages/skill gate,
wrong 1..100 roll, missing equipment and room gates, and missing C damage/hit
path. A seed-2 success then exposed the shared death seam: Go emitted a
generic corpse notice and omitted C's set-143 lethal message ordering, death
cry, and minimum XP line. The confirmed divergences were corrected only in
Go, including the C `group_gain` minimum-XP path for zero-effective-XP mobs.
The final C-first vehicles are GREEN at seeds 1, 2, 3, 5, and 8:

- `cutthroat-depth`: no-skill wrapper control, no-argument, missing target,
  self, no-wield, non-piercing, first-token, immortal-caster, and success.
- `cutthroat-failure`: mortal failure, literal preamble, ordinary dagger hit,
  and wait.
- `cutthroat-peaceful`: peaceful-room gate.
- `cutthroat-mounted`: mounted gate.
- `cutthroat-low-level`: level-10-or-below player protection.

Focused unit tests cover exact hidden affect fields, gate ordering, RNG-free
early exits, and the immortal-target override. `--show-oracle` was used while
developing the vehicles. No `src/` or `darkpawns-c-oracle/` file was edited.

## Evidence and gates

Added `docs/fidelity/depth/cutthroat.tsv`, five oracle scenarios, and
`pkg/game/cutthroat_depth_test.go`. The manifest records 16 cases covering the
reachable command branches and their audiences/state transitions.

Final local gates on the committed slice:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
```

The slice follows R1/R2/R3/R4 and R5/R5e: player bytes, command surface,
draw/order parity, no invention, and actual C call-path verification. The
gate inventory and separate ordinary-hit/damage handling apply R5b/R5c to the
shared combat class.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take the next un-manifested command-table item after `cutthroat`.
