# Depth-fidelity handoff: `alien_elevator`

Date: 2026-08-30  
Queue: special procedures, source/registration order  
Starting main: `7e98c58c5`

## Scope and source audit

This slice covered `SPECIAL(alien_elevator)` in `src/spec_procs3.c:401-424`.
The C body trims an argument, checks the exact `close door` command, requires
the east exit of the current room to be open, moves every occupant from room
19551 (or the other side) to the paired room 19599/19551, and sends each moved
occupant `The room starts to move!\r\n`. If the command or exit gate fails it
returns false.

The registration audit followed the actual C dispatch path. The procedure is
only declared with `SPECIAL(alien_elevator)` in the room declaration block at
`src/spec_assign.c:573-595`; a complete search of the active assignment tables
found no `ASSIGNMOB(..., alien_elevator)`, `ASSIGNOBJ(..., alien_elevator)`, or
`ASSIGNROOM(..., alien_elevator)` entry. C command dispatch checks registered
room, equipment/inventory/present-object, and present-mobile pointers through
`src/interpreter.c:889-947` and `1407-1456`; no such pointer can be populated
for this procedure. There is likewise no autonomous mobile path for this
room-only body without an active assignment.

## Fidelity disposition

Because the procedure has no active C registration, its latent relocation and
message branches are unreachable from the player-facing C surface. A fixture
that manufactured a room special or assigned it to a mob would violate R2 and
R4, so no oracle scenario or synthetic vehicle is claimed. The manifest records
the procedure as D5 `excluded` with the owning C registration proof.

No Go behavior was changed and no `src/` or `darkpawns-c-oracle/` file was
edited. The disposition follows R2/R4/R5e: preserve the real command surface,
do not invent a reachable path, and verify the actual registration and
dispatcher call path. R5c requires no broader audit because no divergence was
reached.

## Verification

All required gates pass on `glm/spec-alien-elevator`: `make fidelity-depth`
reports 1,284 total cases, 1,234 proven/delegated, 14 blocked, and 36
excluded (1,234/1,248 actionable, 98.9%); `go build ./...`; `go vet ./...`;
`go test ./...`; `golangci-lint run ./...` (`0 issues`); `gofumpt -l .` (no
output); and `git diff --check` all pass. The manifest validator accepts the
new excluded row.

## Next queue item

Continue special-procedure source order with `SPECIAL(werewolf)` at
`src/spec_procs3.c:427`; audit its active registrations before choosing a
vehicle.
