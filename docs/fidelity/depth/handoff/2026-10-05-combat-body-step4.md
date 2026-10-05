# Combat body identity — stack 4/4: duplicate-body oracle

**Stop tier:** combat identity and movement cleanup. Base: stack 3, `1908aedc3`
(#1801). All four stacks merge together after the final combined census and
Zach's duplicate-mob playtest. Shoot Train 2 remains parked. R1, R3 and
R5b/e/g/h apply.

The oracle fixture creates two actual same-description trainees in one room,
with a zone-equipped cutlass on only the newest body. Two ordinary PCs start
independent fights; a God observes actor, victim and audience bytes, reports
both targets, kills only the armed duplicate, and observes the surviving fight.
Pulses preserve the ordered draw stream; all claimed seeds are 1,2,3,5,8.

This exposed two existing boundary defects. C's room list is prepended on
arrival, so the canonical room resolver now orders real bodies by descending
RoomEntrySequence, including mixed PCs/NPCs. Zero-sequence legacy test fixtures
retain deterministic fallback ordering. C get_char_room_vis is
`src/handler.c:1276-1300`; room insertion is `src/handler.c:532-556`.
Global fallback list ordering is retained follow-up, not certified here.
C do_kill calls raw_kill with TYPE_SLASH (`src/act.offensive.c:101-161`);
Go now passes that exact attack type, preserving the hacked-up corpse bytes.

MovePlayer drops World.mu only for a fighting body's engine cleanup. After
relocking it rejects movement, relinking, extraction, registry replacement or
a new fight, and rereads room, exit, boat, tunnel and movement cost. The final
body-locked commit checks room and sequence again and charges movement only
when committing. C boundary: `src/handler.c:504-529` and
`src/act.movement.c:126-231`. The deterministic test pauses inside the real
engine stop boundary and changes location, extraction, fighting, door, tunnel
or sector before resuming. Ordinary nonfighting movement has no new lock gap.

Lock acquisitions: initial World.mu; short body locks for snapshots/gates;
release World.mu before stopRoomFights/engine cleanup; reacquire World.mu then
body read lock for validation; fresh room gates under World.mu; body write lock
for final validation and move deduction/room commit. Output remains after
World.mu is released. No lifecycle lock is added. The ordinal resolver snapshots
through existing World accessors, then uses body read locks for sequence; no
World lock is held while sorting. Kill retains the existing Instakill/death
seam and changes only its attack-type argument.

Reproduce the compiled green/revert/restore controls in a disposable worktree:

```sh
python3 docs/fidelity/depth/handoff/2026-10-05-combat-body-step4-controls.py --output /tmp/combat-step4-triples --oracle
```

Controls restore the entire pre-review MovePlayer, remove room sequence ordering,
and restore kill attack type 0. Each must compile and fail assertions. The script
restores files in finally. The oracle's initial red also records swapped attack
verbs/targets and wrong corpse bytes before those two repairs.

Reproduce the remaining-name sweep without treating command-entry lookups as
round callbacks:

```sh
rg -n 'GetName|GetPlayer|FindMob|ResolveChar|GetFighting|GetFollowing' pkg/combat pkg/game/combat*.go pkg/game/death.go pkg/game/char_mgmt.go pkg/game/world_movement.go pkg/session/combat*.go
rg -n 'ResolveCharInRoom|ResolveCharInRoomAt|ResolveCharWorld' pkg
```

The first three stack audits retain the callback/cleanup classification.
Room resolver readers now share actual arrival order; keyword/visibility gates
are unchanged. World fallback ordering, C1 command expansion, C3 extraction
ownership and parked shoot outcomes remain separate frontier items.
