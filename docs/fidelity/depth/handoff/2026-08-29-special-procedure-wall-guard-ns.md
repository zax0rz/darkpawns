# Depth-fidelity handoff — wall_guard_ns

Date: 2026-08-29  
Slice: special procedure `wall_guard_ns`  
Registration: mob vnum `8060` at `src/spec_assign.c:287`  
Code commit: `063aa2846`  
PR: #778, merged as `57a00b8e0`; hosted run `33288837905`

## Queue position

The C inventory was refreshed through `src/spec_procs2.c:2260-2300` and its
registration. `wall_guard_ns` is claimed here and must not be repicked. The
next active special is the first unclaimed procedure in `src/spec_procs3.c`
after the completed `spec_procs2.c` inventory; refresh the declarations,
definitions, and registration tables before selecting it.

## C call path and branch map

The audit followed `special()` from `src/interpreter.c:1407-1456` and the
autonomous `mobile_activity()` path from `src/mobact.c:68-93`. The registered
8060 guard rejects non-empty commands, sleeping position, and an existing
fighting target. On an awake commandless call it chooses the only available
one-way wall direction, moves through the canonical `perform_move()` seam, and
then scans the destination room. If an NPC vnum 8020 is present, C emits four
`TO_ROOM` Acts—salute, greeting, nod, and response—and resets its per-call
`talk` latch. Shared fighter and generic movement behavior remain delegated to
their existing proof surfaces.

## RED → GREEN

The first stable vehicle spawned the authored vnum-8060 city guard at room
8060 and a scriptless vnum-8020 church guard at the north destination. The
destination exit was frozen after setup so the patrol and post-move scan were
observable. On main, the literal procedure direction was reversed for the
RED check; C emitted the four greeting lines while Go emitted none. The
confirmed fixes restored C's north/south selection, used `MobTransfer`, and
replaced lower-case room broadcasts with the four exact `Act` calls. No
`src/` or `darkpawns-c-oracle/` files were edited.

## Proof and verification

Scenario: `cmd/dp-oracle-diff/scenarios/spec-proc-wall-guard-ns.txt`  
Focused tests: `pkg/game/spec_wall_guard_ns_test.go`

The vehicle is oracle-green at seeds 1, 2, 3, 5, and 8. Focused tests cover
command/sleep/fighting entry gates, both one-way patrol directions, destination
audience boundaries, exact greeting bytes, and the per-call `talk` reset.

Required local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .   # clean
git diff --check
```

Hosted `test`, `security`, and `lint` checks all passed. The optional
`build-and-push` and `deploy` jobs were skipped by workflow policy.

The manifest now reports:

```text
1152 total; 1107 proven/delegated, 13 blocked, 32 excluded
Actionable completion: 1107/1120 = 98.8%
```

## Fidelity rulings

This slice follows R1 (exact greeting bytes and audience), R2 (the registered
commandless special surface), R3 (patrol state and per-call reset), R4 (no
invented room output), and R5e (verified `special()`, `mobile_activity()`,
`perform_move()`, and `Act` call paths). R5b/R5c apply to shared fighter and
movement seams; this slice claims only the wall guard's dispatch, topology,
and greeting behavior.

## Next action

Return to `main`, run the frontier and reread the depth-testing instructions
and this newest handoff. Then inventory `src/spec_procs3.c` registrations and
take the first unclaimed active special in source-and-registration order.
