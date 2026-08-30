# Depth-fidelity handoff: `itoh`

Date: 2026-08-30  
Queue: special procedures, source/registration order  
Starting main: `ed55f46e5`  
Merged PR: `#806` (`turn_undead` disposition)

## Scope and source audit

This slice covered `SPECIAL(itoh)` in `src/spec_procs3.c:583-610`.
The procedure is declared in `assign_rooms()` at `src/spec_assign.c:591`,
but the complete `ASSIGNROOM` table contains no active
`ASSIGNROOM(..., itoh)` entry. A complete source search also found no active
object or mobile registration.

The latent body gates on `say` or the apostrophe alias, uses C
`skip_spaces()` plus a case-insensitive exact `itoh` phrase, says the phrase,
emits direct and room teleport messages, moves the actor to room 19875, and
renders the destination room. Those branches can only run through the room
special call in `src/interpreter.c:1407-1415`, but no room is registered to
invoke them.

## Fidelity disposition

This is a D5 `excluded` case, not a blocked proof. Constructing a synthetic
room vehicle or assigning `itoh` in Go would invent a player-facing C command
surface. The manifest records the verified unreachability; no Go behavior
changed and no oracle scenario is valid. This follows R2/R4/R5e: preserve
the actual registration and dispatch surface and verify the call path before
claiming coverage.

No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification

The manifest update passes `make fidelity-depth` with 1,302 total cases,
1,249 proven/delegated, 14 blocked, and 39 excluded (1,249/1,263
actionable, 98.9%). The docs-only slice also passes `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`,
and `git diff --check`.

## Next queue item

Continue special-procedure source order with `SPECIAL(mirror)` at
`src/spec_procs3.c:612`, first auditing its active object registration and
the object-special command path.
