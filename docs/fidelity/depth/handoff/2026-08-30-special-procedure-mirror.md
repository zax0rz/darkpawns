# Depth-fidelity handoff: `mirror`

Date: 2026-08-30  
Queue: special procedures, source/registration order  
Starting main: `d0d64c11e`  
Merged PR: `#807` (`itoh` disposition)

## Scope and source audit

This slice covered `SPECIAL(mirror)` in `src/spec_procs3.c:612-666`.
The procedure is declared in `assign_objects()` at `src/spec_assign.c:526`,
but the complete `ASSIGNOBJ` table contains no active
`ASSIGNOBJ(..., mirror)` entry. A complete source search also found no mobile
or room registration.

The latent object body checks that the object is in a room, skips leading
spaces, and matches the object name with C `isname()`. Its `hit`/`kill` branch
emits break/shatter output, moves a character from room 14496 into the
object's room, creates object 14503, extracts the mirror, and renders the
arrival room. Its `look` branch emits the pull/disappearance output, moves
the actor to room 14496, and renders that room. These branches could only be
entered through the object-special dispatcher in
`src/interpreter.c:1407-1474`; the heartbeat `object_activity()` at
`src/comm.c:758-780` runs object scripts, not object-special function
pointers. No C call path reaches this unassigned special.

## Fidelity disposition

This is a D5 `excluded` case, not a blocked proof. Constructing a synthetic
object vehicle or assigning `mirror` in Go would invent a player-facing C
surface. The manifest records the verified unreachability; no Go behavior
changed and no oracle scenario is valid. This follows R2/R4/R5e: preserve
the actual registration and dispatch surface and verify the call path before
claiming coverage.

No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification

The manifest update passes `make fidelity-depth` with 1,303 total cases,
1,249 proven/delegated, 14 blocked, and 40 excluded (1,249/1,263
actionable, 98.9%). The docs-only slice also passes `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`,
and `git diff --check`.

## Next queue item

Continue special-procedure source order with the active
`SPECIAL(prostitute)` at `src/spec_procs3.c:670`, registered as
`ASSIGNMOB(8023, prostitute)` in `src/spec_assign.c:283`.
