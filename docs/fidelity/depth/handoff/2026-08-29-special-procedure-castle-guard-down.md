# Depth-fidelity handoff — castle_guard_down

Date: 2026-08-29  
Slice: special procedure `castle_guard_down`  
Branch/commit: `glm/spec-castle-guard-down`, `9f32af6f5` (handoff commit to
follow)  
PR: to be opened after this handoff

## Queue position

The refreshed C inventory follows the claimed `chosen_guard` procedure with
`castle_guard_down` at `src/spec_procs2.c:2134-2174`. The procedure is
registered for mob vnums 19626, 19627, 19640, and 19641 at
`src/spec_assign.c:453-456`; all four authored mob prototypes carry the active
`MOB_SPEC` bit. The next active, unclaimed procedure in source-and-registration
order is `castle_guard_up`. Do not repick `castle_guard_down` after this
handoff claims it.

## C call path and branch map

The command path enters `special()` from `src/interpreter.c:1407-1456`, where
the mobile-special scan runs before ordinary command handling. C first rejects
`mini_mud`, a sleeping/dead guard (`!AWAKE(mobile)`), and an immortal actor.
For a mortal actor issuing `down`, it checks the house at the actor's current
room VNUM plus two. The owner branch returns FALSE with no bytes. The grouped
owner branch emits the actor and non-victim allowance Acts and returns FALSE,
so normal movement remains available. The non-owner/non-group branch emits
the victim block, the non-victim path block, and the room statement, then
returns TRUE to suppress ordinary movement.

The commandless path is the registered `mobile_activity()` call from
`src/mobact.c:68-93`, which supplies the mobile as both `ch` and `me`. After
the idle guard passes, C scans `world[mobile->in_room].people`; an assigned
`castle_guard_down` character that is fighting a different character causes
`hit(mobile, FIGHTING(i), TYPE_UNDEFINED)` and returns TRUE. The Go mob ticker
represents this commandless call with `ch=nil`, so the port resolves both
player and mob target names and routes the hit through `w.mobHit`.

## RED → GREEN

The live vehicle on main was RED with an authored vnum-19627 guard spawned in
room 19627 and a disposable open `down` exit. C emitted:

```text
Fazhar the Guardian Genie blocks your way.
Fazhar the Guardian Genie states, 'Thou shalt not pass.'
```

Go emitted the statement plus an invented direct attack, and omitted both
blocking Acts. The confirmed divergence was the existing Go command branch:
it used a room broadcast for every branch and called `me.Attack(ch, w)` in the
blocking branch, while C uses the exact `TO_VICT`/`TO_NOTVICT`/`TO_ROOM` Acts
and never attacks there.

The corrected implementation preserves C's return values and audience split,
including the grouped-owner allowance, and makes the commandless scan accept
an assigned guard fighting either a player or another mob. No `src/` or
`darkpawns-c-oracle/` files were edited.

## Proof and verification

Scenario: `cmd/dp-oracle-diff/scenarios/spec-proc-castle-guard-down.txt`  
Focused tests: `pkg/game/spec_castle_guard_down_test.go`

The vehicle was run on main before the fix with `--show-oracle --seed 1` and
produced the RED transcript above. It was rerun after the fix with the same
flags and returned `result: no normalized divergence`; the normalized C blocks
show the actor's block and the primary peer's path block plus the room
statement. The focused tests cover commandless/non-matching/sleeping/immortal
gates, owner fall-through, grouped-owner audience bytes, blocked audience and
return interception, and the commandless second-guard target boundary for
both player and mob opponents.

The manifest now reports:

```text
1133 total; 1088 proven/delegated, 13 blocked, 32 excluded
Actionable completion: 1088/1101 = 98.8%
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

This slice follows R1 (exact player-facing Acts), R2 (the registered movement
special surface), R3 (preserved combat handoff boundary), R4 (no invented
attack or missing block bytes), and R5e (verified `special()` and
`mobile_activity()` call paths). R5b/R5c apply to the shared `hit()`/combat
seam; this slice claims only the special's target selection and handoff, not
the shared damage matrix.

## Next action

Open one `glm/spec-castle-guard-down` PR, self-merge only after all hosted
checks are green, then return to updated `main` and take `castle_guard_up`.
