# Depth-fidelity handoff — elements_platforms

Date: 2026-08-30  
Slice: special procedure `elements_platforms`  
C declaration/registration: `src/spec_assign.c:85-180,623-626`  
C definition: `src/spec_procs3.c:1004-1024`  
Go registration: `pkg/game/spec_assign.go:379-382`; `pkg/game/spec_procs3.go:91`  
PR: #785, merged as `37c59d1f8`  
Hosted checks: run `33294366215` (`test`, `security`, `lint` green; optional build/deploy skipped)

## Queue position

The fresh session began on clean `main` after the `elements_master_column`
handoff at 1,202 total cases, with 1,156 proven/delegated, 13 blocked, and 33
excluded. The next source-and-registration-order slice was
`elements_platforms`, active on rooms 1326, 1337, 1348, and 1359. The next
active special is `elements_load_cylinders`, registered on rooms 1360, 1364,
1380, and 1384 in `src/spec_assign.c:627-630`.

## C call path and branch inventory

The room special is reached from the current-room player-command dispatcher in
`src/interpreter.c:1407-1456`; it ignores the command text. The movement path
also checks room specials before movement from `src/act.movement.c:115`.

For every player in `world[ch->in_room].people`, C sends the exact direct
message `A wave of power surges through you and you feel dizzy.`, sends the
departure `TO_NOTVICT` Act, moves the player with `char_from_room` and
`char_to_room` to room 1314, then sends the arrival `TO_NOTVICT` Act. There is
no room look in this procedure. The C list is front-inserted, so the vehicle
places the passive peer first and the command actor last; the actor is then
processed first by the room-list walk.

## RED → GREEN

Added `spec-proc-elements-platforms`, a two-client vehicle using room 1326.
The C-first RED showed that the old Go implementation broadcast departure to
the departing player itself, reused the original command actor's room after
that actor had moved, omitted arrival Acts, and used direct writes with the
wrong framing. The Go path now uses canonical `Act` calls for direct,
departure, and arrival messages, checks `PlayerTransfer` errors, and preserves
the C vehicle ordering deterministically.

The oracle matrix was green at seeds 1, 2, 3, 5, and 8; `--show-oracle`
confirmed the intended C room-special block. Focused tests pin direct bytes,
self-broadcast exclusion, actor/peer audience, sequential relocation to room
1314, nil entry, and the active room-special contract.

No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .
git diff --check
```

All local gates passed. The manifest now reports:

```text
1206 total; 1160 proven/delegated, 13 blocked, 33 excluded
Actionable completion: 1160/1173 = 98.9%
```

## Fidelity rulings

This slice follows R1 (exact player-facing bytes), R2 (the active four-room
registration and command surface), R3 (deterministic occupant processing and
transfer state), R4 (no invented look or audience behavior), and R5e (the C
dispatcher, movement hook, registration table, and actual procedure body were
verified before changing Go).

## Next action

Start the next session from `main`, pull, run `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this handoff, then map and prove
`elements_load_cylinders` in C registration order. Continue the
special-procedure inventory, then attempt `objmagic.sleep-entry-gates` once via
the cast-sleep vehicle before sweeping remaining un-manifested interpreter
command families. Leave one dated handoff for the next session.
