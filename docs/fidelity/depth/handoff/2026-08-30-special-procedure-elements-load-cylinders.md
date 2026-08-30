# Depth-fidelity handoff — elements_load_cylinders

Date: 2026-08-30  
Slice: special procedure `elements_load_cylinders`  
C declaration/registration: `src/spec_assign.c:85-180,596,627-630`  
C definition: `src/spec_procs3.c:1026-1144`  
Go registration: `pkg/game/spec_assign.go:379-382`; `pkg/game/spec_procs3.go:92`  
PR: #786, merged as `1f6777f7c`  
Hosted checks: run `33295176270` (`test`, `security`, `lint` green; optional build/deploy skipped)

## Queue position

The fresh session began on clean `main` after the `elements_platforms` handoff
at 1,206 total cases, with 1,160 proven/delegated, 13 blocked, and 33
excluded. The next source-and-registration-order slice was
`elements_load_cylinders`, active on rooms 1360, 1364, 1380, and 1384. The
next active special is `elements_galeru_column`, registered on room 1372 at
`src/spec_assign.c:631` and defined at `src/spec_procs3.c:1137-1182`.

## C call path and branch inventory

The room special is reached from the current-room player-command dispatcher in
`src/interpreter.c:1407-1456`; the movement hook also checks room specials
before movement from `src/act.movement.c:115`. The four active registrations
are `ASSIGNROOM(1360/1364/1380/1384, elements_load_cylinders)`.

The `get` branch calls ordinary `do_get`, then checks only the current room via
`elements_remove_cylinders`; it returns TRUE. The helper scans the C arrays
`talisman={1300,1301,1302,1303}` and `cylinder={1304,1305,1306,1307}` in
order, returns immediately when the current pass sees its talisman, and
otherwise removes the last matching cylinder in that room after sending the
room-wide colored sink line.

The `drop` branch first rejects the command when any cylinder vnum 1304-1307
is already present, returning FALSE so ordinary `do_drop` owns that path. With
no cylinder it calls `do_drop`, parses only the first argument, resolves the
visible dropped object from the room, and loads the matching cylinder only for
the room/talisman pairs: earth 1360→1304/green, air 1364→1305/yellow, fire
1380→1306/red, and water 1384→1307/blue. The creation announcement is sent
to the whole room before the cylinder is inserted with C prepend ordering.
All other commands return FALSE without output or state changes.

## RED → GREEN

The C-first vehicle is
`cmd/dp-oracle-diff/scenarios/spec-proc-elements-load-cylinders.txt`. On clean
main, the earth drop showed two divergences: Go sent the creation line only to
the actor, and the Go room listing rendered the generated ITEM_GLOW cylinder
with parenthesized `(glowing)` text instead of C's separate plain
`...it glows white` line. The C source also confirmed that the existing Go
cleanup used unrelated vnums 1310-1313 and scanned every room.

The Go path now preserves the C room-wide create/remove audience, all four
registered mappings, any-cylinder FALSE fallthrough, first-argument visible
object selection, current-room cleanup, C's early return, and exact extraction
messages. Room object flag rendering now follows the C `oc_show_list` plain
flag-line vocabulary. No `src/` or `darkpawns-c-oracle/` file was edited.

The oracle matrix was GREEN at seeds 1, 2, 3, 5, and 8. Focused tests are in
`pkg/game/spec_elements_load_cylinders_test.go`; the room-list assertion was
updated in `pkg/game/look_test.go` to match `src/oc.c:82-170`.

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
1212 total; 1166 proven/delegated, 13 blocked, 33 excluded
Actionable completion: 1166/1179 = 98.9%
```

## Fidelity rulings

This slice follows R1 (exact bytes and audience), R2 (the four active room
registrations and get/drop command surface), R3 (current-room object state and
C ordering), R4 (no invented cross-room cleanup or output), and R5e (the C
dispatcher, movement hook, registration table, helper, and actual procedure
body were verified before changing Go).

## Next action

Start the next session from `main`, pull, run `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this handoff, then map and prove
`elements_galeru_column` in C registration order. Continue the special-
procedure inventory, then attempt `objmagic.sleep-entry-gates` once via the
cast-sleep vehicle before sweeping remaining un-manifested interpreter command
families. Leave one dated handoff for the next session.
