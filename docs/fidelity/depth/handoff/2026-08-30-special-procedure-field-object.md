# Depth-fidelity handoff: `field_object`

Date: 2026-08-30  
Queue: special procedures, source/registration order  
Starting main: `3a2dbdf81`  
Merged PRs: `#802` (implementation), `#803` (werewolf handoff)

## Scope and source audit

This slice covered `SPECIAL(field_object)` in `src/spec_procs3.c:456-514`.
The C assignment table contains active `ASSIGNOBJ(50, field_object)` and
`ASSIGNOBJ(52, field_object)` entries at `src/spec_assign.c:524-529`.

The actual dispatch audit is decisive. `src/interpreter.c:1407-1474`
invokes object specials only while processing a player command (including
movement), so `cmd` is nonzero. The procedure's first gate is `if (cmd ||
obj->in_room <= 0) return FALSE;`, making every such object-special call
fall through. The heartbeat's `object_activity()` at `src/comm.c:758-780`
only runs `OS_ONPULSE` scripts and does not call object-special function
pointers. No other C call site invokes `field_object`; the complete source
search found only its declaration, assignment, definition, and the separate
field-object data use in `limits.c`.

## Fidelity disposition

The body is therefore unreachable from the player-facing C special-procedure
surface despite its assignment entries. Manufacturing a commandless object
special call or assigning it to a mob would violate R2 and R4. The manifest
records one D5 `excluded` case with the verified dispatcher proof; no oracle
scenario or synthetic vehicle is valid, and no Go behavior was changed.

The independent `limits.c:658-684` `point_update()` branch decrements field
object timers, emits the wear-off message, optionally creates the configured
replacement object, and extracts the expiring object. That is a separate
heartbeat family and is not silently claimed or relabeled as proof of this
`SPECIAL(field_object)` body; R5e/R5c keep its future audit on the
`point_update` call path.

No `src/` or `darkpawns-c-oracle/` file was edited. This disposition follows
R2/R4/R5e: preserve the real command surface, do not invent a reachable
branch, and verify the actual call path before claiming coverage.

## Verification

`make fidelity-depth` passes with 1,292 total cases, 1,241
proven/delegated, 14 blocked, and 37 excluded (1,241/1,255 actionable,
98.9%). The prior merged implementation PR's full repository gates remain
green: `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint run ./...` (`0 issues`), `gofumpt -l .` (no output), and
`git diff --check`.

## Next queue item

Continue special-procedure source order with `SPECIAL(portal_to_temple)` at
`src/spec_procs3.c:515`, auditing the active room registration
`ASSIGNROOM(19658, portal_to_temple)` before building its player-command
vehicle.
