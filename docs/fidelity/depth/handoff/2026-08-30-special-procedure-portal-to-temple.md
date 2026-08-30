# Depth-fidelity handoff: `portal_to_temple`

Date: 2026-08-30  
Queue: special procedures, source/registration order  
Starting main: `6515cd898`  
Merged PR: `#805` at `abfe01ece`

## Scope and source audit

This slice covered `SPECIAL(portal_to_temple)` in
`src/spec_procs3.c:515-540`, actively registered as
`ASSIGNROOM(19658, portal_to_temple)` in `src/spec_assign.c:615`.
The player-command path is the room-special dispatch in
`src/interpreter.c:1407-1415`, followed by the C `do_say` path in
`src/act.comm.c:824-870`, `act()` audience routing in `src/comm.c:2392-2555`,
room relocation, and `do_look` in `src/act.informative.c:665-840`.

The entry gates are `say` or the apostrophe alias only, then C
`skip_spaces()` followed by a case-insensitive exact comparison with
`setchswayno`. On success C says the argument, sends the direct teleport
message, emits the disappearance to the origin room, relocates the actor to
room 8008, emits the appearance to the destination room, and renders the
landing room. The trailing-argument behavior is intentionally preserved:
`skip_spaces()` removes leading whitespace but does not trim trailing bytes.

## Confirmed divergence and fix

The initial seed-1 oracle vehicle was RED. Go omitted the canonical `say`
line and landing-room look, accepted a trailing argument that C rejects, and
sent disappearance/appearance through the wrong room audience. The fix uses
the shared Go `DoSay` and `Act` paths, preserves C's argument parsing, emits
disappearance before relocation and appearance after relocation, and calls
the canonical landing-room look.

No `src/` or `darkpawns-c-oracle/` file was edited. This follows R1/R2/R3/R4
and R5e/R5c: preserve player-facing bytes and command aliases, retain the C
draw/call ordering, do not invent a trimmed argument rule, and use the actual
shared audience paths.

## Proof and verification

The live scenario is
`cmd/dp-oracle-diff/scenarios/spec-proc-portal-to-temple.txt`; it includes
`--show-oracle` proof and the six annotated branches for entry gates,
fallthrough, say audience, teleport audiences, destination look, and final
state. Seeds `1,2,3,5,8` all report no normalized divergence. Focused tests
in `pkg/game/spec_portal_to_temple_test.go` cover the exact argument gate,
say and apostrophe paths, audience routing, relocation, and landing output.

The merged change passed `make fidelity-depth`, `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`,
`gofumpt -l .`, and `git diff --check`; hosted lint, security, and test
checks were green. Build/deploy checks were correctly skipped for the PR.
After merge the frontier is 1,300 total cases, 1,249 proven/delegated, 14
blocked, and 37 excluded (1,249/1,263 actionable, 98.9%).

## Next queue item

Continue special-procedure source order with `SPECIAL(turn_undead)` at
`src/spec_procs3.c:543`, after verifying its active registration entries and
the object-special dispatch path before deciding whether a player-facing
vehicle is reachable.
