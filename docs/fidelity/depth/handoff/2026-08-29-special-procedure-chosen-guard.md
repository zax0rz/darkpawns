# Depth-fidelity handoff — chosen_guard

Date: 2026-08-29  
Slice: special procedure `chosen_guard`  
Branch/PR: `glm/spec-chosen-guard`, PR #774  
Commit: `6d2fa670c`

## Queue position

After teleporter merged on `main` as `64c9b085d`, the refreshed C inventory
selected the next reachable, unclaimed procedure in body-and-registration
order: `chosen_guard`, implemented at `src/spec_procs2.c:2113-2132` and
registered for mob vnum 8096 at `src/spec_assign.c:302`. The unassigned
`no_move_north` body remains excluded under R5e. The next active unclaimed
procedure after this slice is `castle_guard_down`.

## C call path and branch map

The command interpreter calls registered mobile specials before the movement
handler (`src/interpreter.c:1407-1456`). `SPECIAL(chosen_guard)` requires a
nonzero movement command and an awake guard; it returns FALSE for an immortal
or already-chosen player, and otherwise handles only `south`. The blocking
path emits, in C order, the victim line, the `TO_NOTVICT` peer line, and the
room-wide say, then returns TRUE before `do_move()`.

There is no commandless/autonomous combat arm in this procedure. The native
`IS_MOVE(cmd)` gate and the guard's `AWAKE` gate were mapped from the source,
and no house-owner or group exception belongs to this procedure. No `src/` or
`darkpawns-c-oracle/` files were edited.

## RED → GREEN

The C-first two-client vehicle is
`cmd/dp-oracle-diff/scenarios/spec-proc-chosen-guard.txt`. It uses the authored
vnum-8096 guard in room 8144, makes the primary an immortal only for the
positioning warmup, moves the mortal peer into the room, and probes `south`.

The pre-fix main run was RED: C emitted `An old guard blocks your way.` and
`An old guard says 'Thou shalt not pass.'` to the actor, plus the exact peer
`TO_NOTVICT` block and say; Go returned only `Alas, you cannot go that way...`
and omitted the peer output. The confirmed Go divergence was a commandless
combat-style implementation of a command movement interceptor. The fix uses
the canonical `Act` audience calls and returns TRUE only for the C south path.

## Proof and verification

Focused test: `pkg/game/spec_chosen_guard_test.go`, symbol
`TestSpecChosenGuard_EntryGatesAndAudience`  
Oracle vehicle: `spec-proc-chosen-guard`, GREEN with
`--show-oracle --seed 1`.

The focused test pins commandless, non-movement, sleeping-guard,
immortal-target, and chosen-target fallthrough, plus actor/peer audience
bytes and blocked movement state. The live vehicle proves the registered vnum
dispatch, exact actor and peer bytes, and TRUE return interception before
ordinary movement.

At handoff, `make fidelity-depth` reports:

```text
1126 total; 1081 proven/delegated, 13 blocked, 32 excluded
Actionable completion: 1081/1094 = 98.8%
```

Required local gates passed on this branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .   # clean
```

## Fidelity rulings

This slice follows R1 (exact actor, peer, and room bytes), R2 (movement command
surface and TRUE interception), R4 (no autonomous attack invention), and R5e
(verified source registration and interpreter call path). R5b/R5c remain
applicable to the shared movement and Act seams; this slice claims only the
chosen_guard caller boundary.

## Next action

Wait for PR #774's hosted checks. Merge only if every required check is green;
otherwise leave it open and advance with the next queue item,
`castle_guard_down`. Do not repick `chosen_guard`.
