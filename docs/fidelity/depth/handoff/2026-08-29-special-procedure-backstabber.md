# Depth-fidelity handoff — backstabber

Date: 2026-08-29  
Slice: special procedure `backstabber`  
Branch/PR: `glm/spec-backstabber`, PR #772  
Merged main: `373e92019`

## Queue position

The special-procedure inventory was refreshed from the registration tables in
`src/spec_assign.c` and the procedure bodies in `src/spec_procs.c`,
`src/spec_procs2.c`, and `src/spec_procs3.c`. The next unclaimed registered
procedure in source-and-registration order is `teleporter`, body
`src/spec_procs2.c:2019-2030`, registered for mob vnum `14411` at
`src/spec_assign.c:381`. No item claimed by an earlier handoff was repicked.

## C path and behavior proved

The audit followed the real dispatch path: `mobile_activity()` only dispatches
the procedure for an awake, non-fighting mob with `cmd == 0`; the procedure
scans `world[ch->in_room].people` for the first player satisfying
`!PRF_NOHASSLE(i) && CAN_SEE(ch, i)`, then calls
`do_backstab(ch, GET_NAME(i), 0, 1)` (`src/mobact.c:68-93`,
`src/spec_procs2.c:2003-2016`, `src/act.offensive.c:165-235`).

The C-first branch map established these observable rules:

- command, mob fighting, sleeping, no visible player, and `PRF_NOHASSLE`
  gates fall through without consuming backstab rolls;
- the NPC's missing/non-piercing weapon, mounted state, and fighting target
  gates are handled inside `do_backstab` before its percent/probability draws;
- `subcmd == 1` uses `number(1,101)` followed by `number(50,100)`; a failed
  percent roll calls `damage(..., 0, SKILL_BACKSTAB)`, while the other arm
  calls the native NPC `hit(..., SKILL_BACKSTAB)` path;
- the hit arm consumes the NPC THAC0 roll and mob damage dice, applies the
  C backstab multiplier, bypasses `get_minusdam`, routes attack type 131
  through the shared skill-message path, establishes combat, and applies
  `WAIT_STATE(ch, PULSE_VIOLENCE)`;
- the NPC's descriptor-only gate messages and no extra room announcement are
  not player-visible.

The Go implementation now follows those gates and draw/state/message paths.
It also keeps the existing shared damage redirect seam before damage-side
effects. No files under `src/` or `darkpawns-c-oracle/` were edited.

## Proof and verification

Scenario: `cmd/dp-oracle-diff/scenarios/spec-proc-backstabber.txt`  
Focused tests: `pkg/game/spec_backstabber_test.go`

The original Go path produced a genuine RED on main: after the autonomous
vehicle's second pulse it announced a fabricated backstab and reduced the
player from 500 to 490 HP, while the C oracle emitted no player-visible bytes
and kept HP at 500. The corrected scriptless vnum-9151 vehicle is
oracle-green for seeds 1, 2, and 3 across eleven mobile pulses. Because the
disposable `spawn-mob` fixture injects the M reset without its following zone E
equipment command, the live vehicle proves the no-weapon/dispatch surface;
the focused tests equip the authored vnum-9101 sharp dagger and prove both
percent arms, NPC damage dice, multiplier/state, skill-message type, combat
entry, and cooldown.

Required local gates passed on the slice branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .   # clean
```

PR #772 had green `lint`, `security`, and `test` checks before squash merge;
the workflow's build-and-push and deploy jobs were correctly skipped.

At handoff, `make fidelity-depth` reports:

```text
1117 total; 1072 proven/delegated; 13 blocked; 32 excluded
Actionable completion: 1072/1085 = 98.8%
```

## Fidelity rulings

This slice follows R1 (player-facing bytes), R2 (command/dispatch surface),
R3 (deterministic draw and state parity), R4 (no invented behavior), and R5e
(verify the actual C call path). R5b/R5c remain applicable to the shared
damage and combat seams: the procedure owns its native gates and arms, while
the verified common hit/damage/message boundaries stay centralized.

## Next action

On the next session: check out and pull `main`, run `make fidelity-depth`, read
`docs/fidelity/DEPTH_TESTING.md` and this newest handoff, refresh the
special-procedure census, and take `teleporter` only if it remains the next
unclaimed registered procedure. Do not revisit `backstabber`.
