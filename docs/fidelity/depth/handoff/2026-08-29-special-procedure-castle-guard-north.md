# Depth-fidelity handoff — castle_guard_north

Date: 2026-08-29  
Slice: special procedure `castle_guard_north`  
Branch/commit: `glm/spec-castle-guard-north`, `a712e2071`  
PR: to be opened after this handoff

## Queue position

The refreshed C inventory follows the claimed `castle_guard_up` procedure
with `castle_guard_north` at `src/spec_procs2.c:2218-2258`. The procedure is
registered for mob vnums 19510, 19601, 19602, 19675, 19676, 19690, and 19691
at `src/spec_assign.c:447-462`. The authored vnum-19510 prototype carries the
active `MOB_SPEC` bit and is used by the proof vehicle. `no_move_north` remains
unregistered and excluded under R5e. The next active, unclaimed procedure in
source-and-registration order is `wall_guard_ns`, registered for vnum 8060.
Do not repick `castle_guard_north` after this handoff claims it.

## C call path and branch map

The command path enters `special()` from `src/interpreter.c:1407-1456`, where
the mobile-special scan runs before ordinary command handling. C first rejects
`mini_mud`, a sleeping/dead guard (`!AWAKE(mobile)`), and an immortal actor.
For a mortal actor issuing `north`, it checks the house at the actor's current
room VNUM plus two. The owner branch returns FALSE with no bytes. The grouped
owner branch checks the same room+2 coordinate, emits the actor and non-victim
allowance Acts, and returns FALSE so normal movement remains available. The
non-owner/non-group branch emits the victim block, the non-victim path block,
and the room statement, then returns TRUE to suppress ordinary movement.

The commandless path is the registered `mobile_activity()` call from
`src/mobact.c:68-93`, which supplies the mobile as both `ch` and `me`. After
the idle guard passes, C scans `world[mobile->in_room].people`; an assigned
`castle_guard_north` character that is fighting a different character causes
`hit(mobile, FIGHTING(i), TYPE_UNDEFINED)` and returns TRUE. The Go mob ticker
represents this commandless call with `ch=nil`, so the port resolves both
player and mob target names and routes the hit through `w.mobHit`.

The authored vnum-19510 guard also carries `AFF_HIDE`. The C `PERS()` macro
uses `CAN_SEE()`, which checks blindness, light, and invisibility but not
`AFF_HIDE`; the separate room-listing path handles hiding explicitly. The Go
port therefore uses a dedicated C-faithful PERS visibility path while leaving
the existing hide-aware gameplay visibility helper intact.

## RED → GREEN

The live vehicle on main was RED with an authored vnum-19510 guard spawned in
room 19510 and a disposable open `north` exit. With `--show-oracle --seed 1`,
C emitted:

```text
An elven sentinel blocks your way.
An elven sentinel states, 'Thou shalt not pass.'
An elven sentinel blocks CastleNorthMortal's path.
An elven sentinel states, 'Thou shalt not pass.'
```

Go initially emitted the room statement plus an invented direct attack, and
omitted both blocking Acts. After the Act/PERS trace exposed the authored
hide flag, Go also showed `Someone` to the victim because its existing
hide-aware helper was incorrectly reused for C `PERS()`. The confirmed fixes
are the exact `TO_VICT`/`TO_NOTVICT`/`TO_ROOM` branch and the dedicated PERS
visibility boundary; no attack is made in the block branch.

The corrected vehicle returned `result: no normalized divergence` with the
same seed and `--show-oracle`; no `src/` or `darkpawns-c-oracle/` files were
edited.

## Proof and verification

Scenario: `cmd/dp-oracle-diff/scenarios/spec-proc-castle-guard-north.txt`  
Focused tests: `pkg/game/spec_castle_guard_north_test.go`

The focused tests cover commandless/non-matching/sleeping/immortal gates,
room+2 owner fall-through, grouped-owner allowance audience, blocked audience
and return interception, and the commandless second-guard target boundary for
both player and mob opponents. The existing
`TestFidelityCanSeeBlindnessAndInvis` regression also remains green after the
PERS-specific visibility change.

The manifest now reports:

```text
1147 total; 1102 proven/delegated, 13 blocked, 32 excluded
Actionable completion: 1102/1115 = 98.8%
```

Required local gates passed on this branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .   # clean
git diff --check
```

## Fidelity rulings

This slice follows R1 (exact player-facing Acts and PERS bytes), R2 (the
registered movement special surface), R3 (preserved combat handoff boundary),
R4 (no invented attack or missing block bytes), and R5e (verified `special()`,
`mobile_activity()`, `comm.c`, and `utils.h` call paths). R5b/R5c apply to the
shared `hit()`/combat seam; this slice claims only the special's target
selection and handoff, not the shared damage matrix.

## Next action

Open one `glm/spec-castle-guard-north` PR, self-merge only after all hosted
checks are green, then return to updated `main` and take `wall_guard_ns`.
