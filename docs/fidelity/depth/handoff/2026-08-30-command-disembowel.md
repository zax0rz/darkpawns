# Depth handoff — disembowel command

Date: 2026-08-30
Queue slice: `src/interpreter.c:415`, `disembowel` / `do_disembowel`
Starting main: `5e69707c3`

## Queue decision

The special-procedure inventory remains exhausted and the one permitted
`objmagic.sleep-entry-gates` cast-sleep attempt remains blocked. The command
table sweep therefore advanced to `disembowel`, the next un-manifested family
in source order. No source or C-oracle file was edited.

## C path and proof

The command table registers `disembowel` at `src/interpreter.c:415` with
`POS_FIGHTING`, dispatching to `do_disembowel` in `src/act.offensive.c:234-293`.
The handler parses one argument with `one_argument`, accepts a visible room
target first, falls back to `FIGHTING(ch)` when the typed target is absent,
rejects self-targeting, requires a wielded piercing weapon, rejects mounted
actors, then draws `number(1,101)` and (for the command's `subcmd == 0`)
`number(50,100)`. A normal awake-target percentage failure calls
`damage(ch,vict,0,SKILL_DISEMBOWEL)`; the other arm calls
`hit(ch,vict,SKILL_DISEMBOWEL)`, which consumes the ordinary d20 and weapon
dice before replacing damage with `level*2+damroll`, then improves the skill
and waits `PULSE_VIOLENCE*2`.

The finalized `disembowel-depth` vehicle was RED on clean main at seed 1:
main emitted the invented `You drive your blade deep...` line where C emitted
the native set-184 message. After the repair it is GREEN at seeds 1, 2, 3, 5,
and 8. The vehicle uses registered disposable mob vnum 5005 in its authored
sleeping state and lowers the crowned actor to level 10, keeping the target
alive while proving the native hit-message branch. Focused tests cover no
target, explicit-target fallback to fighting, no skill delegation, no weapon,
non-piercing weapon, mounted, normal-player percentage failure, fixed damage,
message-path flags, and draw order.

The Go path now uses the C target parser and fighting fallback, preserves the
weapon and mounted gate order, draws the C percentage/probability pair, routes
both zero-damage and positive `hit()` outcomes through a dedicated
`damage()`-equivalent seam, emits skill-message set 184 after position update,
preserves death-cry/XP ordering, and suppresses the generic corpse notice for
this raw-kill path. This follows R1/R2/R3/R4 and R5/R5e: exact player bytes,
command-surface fidelity, deterministic draw/state parity, no invented
behavior, and verification of the actual C dispatch and call path.

## Evidence and gates

Added or changed:

- `cmd/dp-oracle-diff/scenarios/disembowel-depth.txt`
- `docs/fidelity/depth/disembowel.tsv`
- `pkg/command/disembowel_depth_test.go`
- `pkg/game/disembowel_depth_test.go`
- `pkg/command/skill_commands.go`
- `pkg/game/skill_c10_combat.go`
- `pkg/game/damage_stubs.go`
- `pkg/game/death.go`
- `pkg/game/skill_formulas_golden_test.go`

The local gates passed:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
git diff --check
```

The frontier after the manifest addition is 1564 total cases, 1510
proven/delegated, 14 blocked, and 40 excluded; actionable completion is
1510/1524 (99.1%).

## Next queue item

After this slice's PR is handled, return to clean `main`, pull, refresh the
frontier, reread the testing guide and newest handoff, and take the next
un-manifested command-table family after `disembowel`.
