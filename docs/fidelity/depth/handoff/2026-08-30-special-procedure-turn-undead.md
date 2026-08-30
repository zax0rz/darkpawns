# Depth-fidelity handoff: `turn_undead`

Date: 2026-08-30  
Queue: special procedures, source/registration order  
Starting main: `abfe01ece`  
Merged PR: `#805` (`portal_to_temple`)

## Scope and source audit

This slice covered `SPECIAL(turn_undead)` in `src/spec_procs3.c:543-580`.
The procedure is declared in `assign_objects()` at `src/spec_assign.c:525`,
but the complete `ASSIGNOBJ` table contains no active
`ASSIGNOBJ(..., turn_undead)` entry. A complete source search also found no
mobile or room assignment.

The latent body has a player-command branch for `use` in rooms 19875 and
19876: C skips leading spaces, applies `isname(argument, obj->name)`, emits
the room ray-of-flame message, and creates the reciprocal north/south exit.
It also has a `!cmd` branch that frees those exits. The actual object-special
dispatcher in `src/interpreter.c:1407-1474` is reached while handling a
player command and supplies a nonzero command. The heartbeat
`object_activity()` in `src/comm.c:758-780` runs object scripts, not object
special function pointers. No C call path reaches this unassigned special.

## Fidelity disposition

This is a D5 `excluded` case, not a blocked proof. Building a synthetic
object vehicle or assigning the procedure to a Go object would invent a
player-facing C surface and misrepresent the commandless cleanup branch.
The manifest records the verified unreachability; no Go behavior changed and
no oracle scenario is valid. This follows R2/R4/R5e: preserve the actual
registration and dispatch surface, and verify the C call path before claiming
coverage.

No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification

The manifest update passes `make fidelity-depth` with 1,301 total cases,
1,249 proven/delegated, 14 blocked, and 38 excluded (1,249/1,263
actionable, 98.9%). The docs-only slice also passes `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`,
and `git diff --check`.

## Next queue item

Continue special-procedure source order with `SPECIAL(itoh)` at
`src/spec_procs3.c:583`, first auditing its active room registration before
building a say/apostrophe vehicle.
