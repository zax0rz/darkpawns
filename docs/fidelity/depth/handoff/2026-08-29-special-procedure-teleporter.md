# Depth-fidelity handoff — teleporter

Date: 2026-08-29  
Slice: special procedure `teleporter`  
Branch/commit: `glm/spec-teleporter`, `4b662f7ec` (handoff commit
`225771b4e`)
PR: #773, merged as `64c9b085d` after the initial transient DNS failure cleared.

## Queue position

The C inventory was refreshed from the `SPECIAL` declarations and the final
`ASSIGNMOB` registrations. `teleporter` is the next registered procedure after
the already-claimed `backstabber`, with body `src/spec_procs2.c:2019-2030` and
registration `src/spec_assign.c:381` for mob vnum 14411. The next active,
unclaimed procedure after this slice is `chosen_guard`; the unassigned
`no_move_north` body remains outside the reachable surface under R5e.

## C call path and branch map

The audit followed the actual C path. `mobile_activity()` skips fighting mobs;
`perform_violence()` performs the ordinary combat turn and then invokes the
registered native special as `(ch, ch, 0, "")` (`src/fight.c:1898-2032`).
`SPECIAL(teleporter)` returns FALSE for non-empty command, no fighting target,
or `!AWAKE(ch)` and otherwise handles the pulse only below half health by
emitting the room say and calling `call_magic(ch, ch, 0, SPELL_TELEPORT, ...)`.
The C `AWAKE` macro is `GET_POS(ch) > POS_SLEEPING` (`src/utils.h:450`), while
`call_magic()` itself rejects only `POS_SITTING` on this direct native path
(`src/spell_parser.c:406-486`).

The reachable spell branches were mapped through `src/spells.c:168-217`:

- self-target NPCs skip the different-NPC saving throw and low-level victim
  checks;
- the random draw selects a C room index until `ROOM_PRIVATE` is false;
- the NPC receives no descriptor-visible black-screen line, the origin room
  receives the fade-out Act, the character is moved, and the destination room
  receives the fade-in Act;
- the native special returns TRUE even when `call_magic()` rejects a sitting
  caster, so that sitting gate is separately proved.

The authored mob line for vnum 14411 is `18524`, which has no `MOB_SPEC` bit.
The vehicle therefore uses the disposable-only `set-mob-flag 14411 SPEC`
fixture after stripping the Lua script; no source or authoritative world file
was edited. This preserves the R5e finding that the production registration is
otherwise dormant while explicitly exercising the registered native call path.

## RED → GREEN

The first valid vehicle on main was RED after the disposable `MOB_SPEC` flag was
enabled. C emitted five ordinary claw lines, the exact teleporter say, and the
origin fade-out. Go emitted the claw lines and say, then ordinary player attack
bytes instead of the fade-out. The confirmed cause was that Go's generic
`worldTransfer` interface could not be implemented by `*game.World`: its room
lookup returns `*parser.Room`, and its transfer methods take typed game values.
The corrected spell path uses the exact room APIs, maps C RNUM draws through
stable world room order, and uses the existing `CharTransfer` boundary.

The same vehicle also confirmed that C `do_set` resolves active NPCs: before
the fix C acknowledged `set master hit 100`, while Go said `No one by that name
online.` The Go wizard path now supports the confirmed NPC `hit` field and has
a focused regression test; it remains deliberately scoped to that vehicle
enabler.

The corrected path also removes the old invented teleport failure text and
caster success echo, adds the C origin/destination Acts, skips the self-NPC
save throw, and uses a direct-special spell entry so resting mobs are not
blocked by the command parser's `POS_FIGHTING` minimum. No `src/` or
`darkpawns-c-oracle/` files were edited.

## Proof and verification

Scenario: `cmd/dp-oracle-diff/scenarios/spec-proc-teleporter.txt`  
Focused tests: `pkg/game/spec_teleporter_test.go` and
`pkg/session/cmd_set_test.go`

The live vehicle is oracle-green with `--show-oracle --seed 1`. Seeds 3 and 8
also match. Seeds 2 and 5 expose an unrelated existing combat transcript
extra (`Master ... dodges TeleporterGod's attack!`) in the warm-up pulse, so
they are not claimed as teleporter multiseed proof. The focused tests pin the
room-index-to-VNUM draw, destination relocation, reciprocal combat teardown,
room Acts, resting acceptance, and sitting rejection. The vehicle's NPC target
also proves that no player-visible victim black-screen or invented caster line
appears.

At handoff, `make fidelity-depth` reports:

```text
1123 total; 1078 proven/delegated, 13 blocked, 32 excluded
Actionable completion: 1078/1091 = 98.8%
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

This slice follows R1 (player-facing bytes), R2 (the registered commandless
special surface), R3 (room-index draw and state parity), R4 (no invented
teleport output or failure loop), and R5e (the dormant authored flag and actual
`perform_violence()` call path were verified). R5b/R5c apply to the shared
spell and combat seams; this slice claims only the self-NPC branch reached by
`teleporter`.

## Next action

PR #773 is merged with green hosted checks. The next queue item is
`chosen_guard`; do not repick `teleporter` because this handoff claims it.
