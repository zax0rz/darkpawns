# 2026-08-29 — `elevator` depth slice

## Frontier and queue

- The session boundary returned to `main`, pulled successfully, and confirmed
  the required depth instructions plus the `enter_circle` handoff.
- Before this slice, `make fidelity-depth` reported 905 total cases, 882
  proven/delegated, 6 blocked, and 17 excluded: 882/888 actionable (99.3%).
- The next source-order special after the already-claimed `enter_circle` slice
  was `elevator`.
- This slice adds one explicit excluded row. The resulting frontier is 906
  total, 882 proven/delegated, 6 blocked, and 18 excluded: 882/888
  actionable (99.3%).

## C path and registration census

- `src/spec_procs.c:1981-2019` defines `SPECIAL(elevator)`. Its latent command
  gate accepts only `say` or the apostrophe alias, then requires one of two
  exact rune phrases. On a match it calls `do_say`, sends the actor's portal
  message, broadcasts the origin-room message, moves up to two occupants from
  virtual room 5799 to room 5743, and calls `do_look` for each moved occupant.
- `src/spec_assign.c:573-603` declares `SPECIAL(elevator)`, but the complete
  room assignment list at `src/spec_assign.c:605-635` contains no
  `ASSIGNROOM(..., elevator)` call. The mobile assignment table likewise has
  no `ASSIGNMOB(..., elevator)` call.
- `src/spec_assign.c:68-74` shows that only `ASSIGNROOM` installs a room
  function pointer. `src/interpreter.c:1407-1415` invokes a room special only
  through that pointer, and its later mobile loop invokes only assigned mobile
  pointers. Thus the C dispatch path for `elevator` is empty despite the
  procedure body and the existing room 5799 world data.
- The Go registry contains a latent `specElevator` function, but
  `RoomSpecAssign` has no `elevator` entry and no C assignment authorizes one.
  Adding it would invent a player-facing command surface under R2/R4 and
  violate the actual C path required by R5e.

## Manifest result

Added `mob.elevator-unassigned` to
`docs/fidelity/depth/spec-procs.tsv` as D5 `excluded`, with no proving
scenario. This records the source definition and both assignment-table checks;
it does not claim either rune phrase, relocation, audience message, or
destination look as reachable C behavior. No `src/` or
`darkpawns-c-oracle/` file was edited.

## Verification and next handoff

The manifest validator and full repository gates pass on this documentation-only
slice; no Go behavior changed, so no RED/GREEN oracle vehicle is legitimate.
The prior `enter_circle` PR #741 was green and merged; this slice's PR #742
also passed lint, security, and test after the normal queued checks and merged
as `81e204ea3`. Its handoff, the `enter_circle` handoff, and the pet-shop
handoff are published on `main`.

The next source-order special definition is `elemental_room` in
`src/spec_procs.c:2021`. Repeat the complete assignment census before building
any vehicle, without repicking `elevator` or earlier claimed items. After the
active special inventory is exhausted, attempt the single blocked
`objmagic.sleep-entry-gates` row through the cast-sleep outlaw/reagent vehicle,
then sweep un-manifested interpreter command families in table order.

This handoff applies R2 (registered command surface), R4 (no invented
behavior), and R5e (verify the actual C assignment/call path); R1 remains
unclaimed because no player-facing C branch is dispatchable.
