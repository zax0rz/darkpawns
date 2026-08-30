# Depth-fidelity handoff — `fly_exit_up` — 2026-08-30

## Frontier and queue position

- Started on freshly pulled `main` at `22f75ed9d`, after the merged
  `elements_guardian` slice. The opening `make fidelity-depth` report was
  **1237 total, 1191 proven/delegated, 13 blocked, 33 excluded**, actionable
  **1191/1204 (98.9%)**.
- Consumed the registered room special `fly_exit_up`, defined at
  `src/spec_procs3.c:1289-1301` and actively registered by
  `ASSIGNROOM(1389, fly_exit_up)` at `src/spec_assign.c:635`. The comment also
  mentions room 2203, but the active assignment table contains only room 1389.
- The manifest now reports **1245 total, 1198 proven/delegated, 13 blocked,
  34 excluded**, actionable **1198/1211 (98.9%)**.
- A refreshed definition/registration census confirms that `fly_exit_up` is
  the final definition in `spec_procs3.c`, but it is not the terminal active
  special overall: earlier registered definitions such as
  `brain_eater` (`src/spec_procs3.c:198-223`, mobs 14420 and 14432 at
  `src/spec_assign.c:383,386`) remain unclaimed in the depth manifest. The
  next session must resume at that first unclaimed active definition in source
  order; do not jump to the blocked sleep row yet.

## C call path and branch map

The player command reaches `special()` through
`src/interpreter.c:1407-1456`, where the current room special is checked before
the ordinary command handler. Movement also reaches room specials through the
`need_specials_check` branch in `src/act.movement.c:115-120`. The active room
assignment is room 1389, whose `up` exit leads to room 1394 in
`lib/world/wld/13.wld:1877-1997`.

`SPECIAL(fly_exit_up)` has one exact early-fallthrough predicate:
`GET_LEVEL(ch) > LVL_IMMORT`, `IS_NPC(ch)`, `!CMD_IS("up")`, or
`AFF_FLYGED(ch)` returns `FALSE`. For a mortal, non-flying player issuing the
exact `up` command, C sends the direct line
`You try and jump up there but it's just too high.\r\n`, then calls
`act("$n jumps up and down in a vain attempt to travel upwards.", ..., TO_NOTVICT)`
and returns `TRUE`, preventing ordinary movement. The `IS_NPC` arm is on C's
generic `char_data` signature; Go's room-special `SpecFunc` accepts
`*Player`, so no fake player-as-NPC branch is invented. The C audience and
newline behavior are governed by `src/comm.c:2392-2555`.

## RED → GREEN

- The high-level vehicle was first run on pre-fix `main` with
  `empty-players`, the primary at the immortal level, and a peer in the origin
  room. C passed through to room 1394 while Go incorrectly blocked the command
  with the special's refusal text. This was the required RED for the missing
  `GET_LEVEL(ch) > LVL_IMMORT` gate.
- The mortal blocked vehicle also exposed two confirmed byte/audience defects:
  Go passed a message already ending in CRLF to the `sendToChar` helper, which
  added a second CRLF, and `roomMessage` sent the `TO_NOTVICT` line back to the
  actor. The focused test captured both defects before the fix.
- A first high vehicle included a destination observer and exposed an
  unrelated dark-room arrival-visibility difference after the special allowed
  movement. Following R4/R5e, that observer was removed from the isolation
  vehicle and the immortal actor was given C-compatible `holylight`; the final
  vehicle proves only the special pass-through, origin audience, and ordinary
  destination look.
- The Go-only fix adds the strict high-level/nil/world/command gate, uses the
  helper without embedding a second newline, and replaces the broad room
  broadcast with canonical `Act(..., ToNotVict)`. No files under `src/` or
  `darkpawns-c-oracle/` were edited.

## Proof and verification

Scenarios:

- `cmd/dp-oracle-diff/scenarios/spec-proc-fly-exit-up-high.txt`
- `cmd/dp-oracle-diff/scenarios/spec-proc-fly-exit-up-block.txt`

Focused test: `pkg/game/spec_fly_exit_up_test.go`.

The high-level pass-through vehicle reported `result: no normalized divergence`
for seeds **1, 2, 3, 5, and 8**; seed 8 required one retry after a transient C
WHOD bind collision before readiness. The mortal blocked vehicle reported no
normalized divergence for seeds **1, 2, 3, 5, and 8**. Seed 1 was run with
`--show-oracle`, confirming the C destination look/origin leave on the allowed
arm and the direct refusal plus peer-only jump message on the blocked arm.

The focused test covers nil, non-`up`, high-level, exact `LVL_IMMORT` boundary,
flying, direct bytes, `TO_NOTVICT` audience, handled return, and unchanged
room state.

## Manifest and gates

Added seven proven/delegated rows and one honest excluded row to
`docs/fidelity/depth/spec-procs.tsv` for entry, high-level, fly, allowed
audience, blocked message, blocked audience, blocked state, and the unreachable
Go NPC-interface arm. All rows cite R1/R2/R3/R4/R5e as applicable.

Passed locally on the slice branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .   # clean
git diff --check
```

Branch: `glm/spec-fly-exit-up`.

## Next action

Open one PR for this slice and merge only after all required GitHub checks are
green. Then return to `main`, pull, rerun the frontier, reread
`docs/fidelity/DEPTH_TESTING.md` and this handoff, and refresh the ordered
special-procedure census. Resume with the first unclaimed active definition in
`src/spec_procs3.c` before attempting `objmagic.sleep-entry-gates`; after the
special inventory and that one blocked attempt are exhausted, sweep
un-manifested `src/interpreter.c` command families in table order.

This slice follows R1 (exact bytes and audience), R2 (registered command and
movement surface), R3 (deterministic multi-seed vehicles), R4 (no invented
behavior), and R5e (verified C dispatch and movement call paths). The remaining
inventory must continue to apply R5b/R5c when repeated evidence identifies a
shared class.
