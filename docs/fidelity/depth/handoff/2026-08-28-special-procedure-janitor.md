# Depth-fidelity handoff: `janitor`

Date: 2026-08-28  
Branch: `glm/spec-janitor`  
Starting main: `257747bd4` (`fix: deepen fido special procedure (#701)`)

## Frontier

Before this slice, `make fidelity-depth` reported 574 total cases, 561
proven/delegated, 2 blocked, and 11 excluded: 561/563 actionable, or 99.6%.

This slice adds five manifest rows. The post-slice frontier is 579 total, 565
proven/delegated, 3 blocked, and 11 excluded: 565/568 actionable, or 99.5%.

## Queue position and C inventory

`janitor` is the next unproven special procedure in the definition order after
`fido` at `src/spec_procs.c:750-768`. The active registrations are mob VNum
8061 at `src/spec_assign.c:292` and mob VNum 21229 at `:505`. The next source
procedure is `cityguard` at `src/spec_procs.c:771`; the blocked
`objmagic.sleep-entry-gates` row remains after the special-procedure inventory.

The actual C paths are:

- autonomous mobile activity: `src/mobact.c:68-93` invokes the mob special
  with `cmd == 0` after its awake and fighting filters;
- player command dispatch: `src/interpreter.c:1407-1456` invokes a present
  mob special with the nonzero command;
- procedure: `src/spec_procs.c:750-768` rejects nonzero command, non-awake
  mobs, and negative HP, then scans the room's linked object list;
- object predicate: `CAN_WEAR(i, ITEM_WEAR_TAKE)` plus the source's exact
  reversed `isname(i->name, "corpse")` call;
- success: `act("$n picks up some trash.", ..., TO_ROOM)`, `obj_from_room`,
  `obj_to_char`, and `TRUE`.

The Go change in `pkg/game/spec_procs.go` removes the invented random gates,
uses the typed takeability check and the exact reversed-name behavior, emits
the canonical `Act` room bytes, and transfers through a checked canonical
ObjectLocation path. `MoveObjectToMobInventoryFront` in
`pkg/game/movement.go` preserves C's `obj_to_char` prepend order. No C or
oracle-tree file was edited.

## Proof and blocked branch

Clean main was proven RED with a disposable focused test. A deterministic
stream was selected so the old implementation's extra `number(0, 5)` and
`number(0, 2)` gates passed; it then emitted
`a test mob picks up a steel sword.\r\n` instead of C's exact
`A test mob picks up some trash.\r\n`. The disposable test was removed after
the RED result.

Focused GREEN tests are:

- `TestSpecJanitor_EntryGates` — command, sleeping, and negative-HP returns;
- `TestSpecJanitor_PredicateAndTransfer` — exact takeability/corpse predicate,
  room `Act` bytes, room removal, canonical location, and prepend order;
- `TestSpecJanitor_NoEligibleObject` — silent fallthrough when no object is
  eligible.

The C-first vehicle is
`cmd/dp-oracle-diff/scenarios/spec-proc-janitor.txt`, annotated for all five
manifest rows. Two honest live vehicles were attempted: first registration
8061 and second registration 21229, each scriptless and placed in a temporary
no-exit room with the object setup before the padded autonomous pulse. In both
vehicles the native janitor was absent before the probe, and
`--show-oracle` produced empty pulse blocks. A separate `load mob` attempt
reset the C connection. The normalized empty comparison therefore proves
neither dispatch nor output; it is retained as evidence of the vehicle gap,
not relabeled as oracle-green. `mob.janitor-pulse-dispatch` is marked blocked
after these two honest attempts, with no invented transcript.

## Manifest rows

Added `mob.janitor-entry-gates`, `mob.janitor-trash-predicate`,
`mob.janitor-trash-pickup`, `mob.janitor-inventory-transfer`, and
`mob.janitor-pulse-dispatch` to `docs/fidelity/depth/spec-procs.tsv`.

This slice follows R1 (player-facing bytes), R2 (command and autonomous
dispatch surface), R4 (no invention), R5b/R5c (audit the reachable class and
delegate only shared behavior), and R5e (verify the actual C call path). No
new random draw was retained, so the old non-C janitor RNG behavior is not
carried forward under R3.

## Verification and next handoff

`make fidelity-depth` passes at 579/565/3/11. The normalized janitor vehicle
has no divergence but no executed oracle blocks; the focused janitor tests
pass. Full repository gates and the PR checks must be green before merging
`glm/spec-janitor`; otherwise leave the PR open and advance without fixing
outside this slice.

After this PR is merged only when all GitHub checks are green, refresh `main`,
rerun `make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and this
newest handoff, then take the next unproven special procedure, `cityguard`,
in source definition order. The blocked `objmagic.sleep-entry-gates` row
remains after the special-procedure inventory and before the interpreter
command-family sweep.
