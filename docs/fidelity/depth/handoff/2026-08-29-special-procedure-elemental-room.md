# 2026-08-29 — `elemental_room` depth slice

## Frontier and queue

- The session boundary returned to `main`, pulled successfully, ran the depth
  validator, and reread `docs/fidelity/DEPTH_TESTING.md` plus the prior
  `enter_circle` handoff.
- Before this slice, `make fidelity-depth` reported 906 total cases, 882
  proven/delegated, 6 blocked, and 18 excluded: 882/888 actionable (99.3%).
- The next source-order special after the already-claimed `elevator` slice was
  `elemental_room`.
- This slice adds one explicit excluded row. The resulting frontier is 907
  total, 882 proven/delegated, 6 blocked, and 19 excluded: 882/888
  actionable (99.3%).

## C path and registration census

- `src/spec_procs.c:2021-2068` defines `SPECIAL(elemental_room)`. Its latent
  commandless pulse loops over room occupants, sends sector-specific damage
  text plus the dying line to mortals, subtracts 100 hit points, and can emit a
  death-room message before `raw_kill`; immortals receive the ignore-elements
  line.
- `src/spec_assign.c:573-603` declares `SPECIAL(elemental_room)`, but the
  complete room assignment list at `src/spec_assign.c:605-635` contains no
  `ASSIGNROOM(..., elemental_room)` call. The mobile assignment table likewise
  has no `ASSIGNMOB(..., elemental_room)` call.
- `src/spec_assign.c:68-74` shows that only `ASSIGNROOM` installs a room
  function pointer. `src/mobact.c:68-93` invokes autonomous specials only for
  assigned mobs, while `src/interpreter.c:1407-1456` invokes only registered
  room/mobile pointers from the live command path. Therefore the C dispatch
  path for `elemental_room` is empty despite its commandless body.
- The Go registry contains a latent `specElementalRoom` function, but
  `RoomSpecAssign` has no `elemental_room` entry and no C assignment authorizes
  one. Adding a synthetic room or mob assignment would invent player-facing
  behavior under R2/R4 and contradict the actual C path required by R5e.

## Manifest result

Added `mob.elemental-room-unassigned` to
`docs/fidelity/depth/spec-procs.tsv` as D5 `excluded`, with no proving
scenario. This records the source definition and both assignment-table checks;
it does not claim any sector damage, death, or immortal-ignore output as
reachable C behavior. No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification and next handoff

The manifest validator and full repository gates pass on this documentation-only
slice; no Go behavior changed, so no RED/GREEN oracle vehicle is legitimate.
PR #742 for `elevator` was green and merged as `81e204ea3`, and its handoff is
published on `main`. PR #743 for this slice remains open and unmerged: lint and
security passed, but the test job failed in the pre-existing race
`pkg/telnet/TestListenAcceptNotBlockedBySlowReverseDNS` at
`pkg/telnet/listener_test.go:988`; no retry or unrelated fix was attempted.

The next source-order special definition is `pray_for_items` in
`src/spec_procs.c:2071`. Its room assignment must be treated as active and
audited for a real vehicle; do not repick `elemental_room` or earlier claimed
items. After the active special inventory is exhausted, attempt the single
blocked `objmagic.sleep-entry-gates` row through the cast-sleep outlaw/reagent
vehicle, then sweep un-manifested interpreter command families in table order.

This handoff applies R2 (registered command surface), R4 (no invented
behavior), and R5e (verify the actual C assignment/call path); R1 remains
unclaimed because no player-facing C branch is dispatchable.
