# Depth handoff — charge command

Date: 2026-08-30
Queue slice: `src/interpreter.c` first un-manifested command family, `charge`
Starting main: `f556f5487`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`, `src/spec_procs2.c`,
and `src/spec_procs3.c`, including active registration-table claims in
`src/spec_assign.c`, is exhausted. Existing handoffs claim the blocked
`objmagic.sleep-entry-gates` row via the cast-sleep outlaw/reagent vehicle; it
remains blocked because the C `SPELL_SLEEP` entry is `TAR_NOT_SELF`. Per the
no-repick rule, this session did not repeat that vehicle. The interpreter sweep
therefore selected `charge` at `src/interpreter.c:377`, handler `do_charge`,
immediately after the manifested `cast` family.

## C path and proof

`src/new_cmds.c:880-955` performs `one_argument`, the skill gate, target lookup
with FIGHTING fallback, self gate, wielded sword/lance gates, the shared
`number(1,101)` chance calculation, `damage(..., SKILL_CHARGE)`, the unmounted
POS_SITTING transition on failure, conditional improvement on success, and
unconditional `WAIT_STATE(ch, PULSE_VIOLENCE*2)`. `damage()` then enrolls both
combatants, routes non-weapon attack type 147 through `skill_message`, and may
consume the post-damage pain/scream `number(0,2)` draw.

The first RED vehicle found three confirmed divergences on main: Go joined all
target words instead of C's first `one_argument` token, Go emitted invented
charge literals instead of C set 147, and Go did not enroll the pair for the
subsequent combat pulse. After those were fixed, draw logging exposed one more
seed-2 mismatch: C's high-damage survival branch burned `number(0,2)` before
`improve_skill`, while Go omitted it. The focused bridge and regression test
removed that RNG offset. The C skill-message `$p` token was also wired to the
wielded object's short description, yielding `a short sword` exactly.

Live proof is green for `charge-depth` and `charge-failure-depth` at seeds
`1,2,3,5,8`; the latter forces the failure arm with skill 1. Focused command,
combat, and damage tests cover target fallback, weapon gates, mount/NOBASH
modifiers, result contracts, and the survival draw.

## Changes

- `pkg/command/skill_commands.go`: C `one_argument` target parsing.
- `pkg/combat/{callbacks.go,skill_messages.go}` and
  `pkg/game/combat_wire.go`: faithful `$p` weapon-description bridge.
- `pkg/game/skill_combat.go`: set-147 skill-message routing, combat enrollment,
  damage skill identity, deferred improvement, and two-pulse result contract.
- `pkg/game/damage_stubs.go`: confirmed charge survival draw after damage.
- `cmd/dp-oracle-diff/scenarios/{charge-depth,charge-failure-depth}.txt` and
  `docs/fidelity/depth/charge.tsv`: durable evidence.

All changes obey R1/R2/R3/R4 and the R5/R5e call-path rule; the shared draw
audit follows R5b/R5c. Neither `src/` nor `darkpawns-c-oracle/` was edited.

## Next queue item

If this PR merges with all checks green, return to clean `main`, refresh the
frontier, and take the next un-manifested interpreter-table family in order:
`circle` (`src/interpreter.c:378`).
