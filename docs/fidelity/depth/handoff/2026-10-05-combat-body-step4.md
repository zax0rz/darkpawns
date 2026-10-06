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

## Post-review repair and integration

DeepSeek's review identified four residual identity consumers. Breath's engine
enrollment self-test now compares bodies (`src/fight.c:1367-1445`); NPC post-PC
kill loot compares actual killer and victim (`src/fight.c:1693-1705`); all three
literal skill-output phases exclude actor/target bodies (`src/comm.c:2485-2547`).
The ordinary, post-damage and retaliation audience cases independently preserve
both observer lines when an NPC target shares that observer's name.

Spell grouping is live-reachable through MagGroups, MagMasses, MagAreas,
hellfire and meteor swarm. All five pass their World to areGrouped. Live World
uses existing FollowingBody to test C's root/immediate-follower edges
(`src/utils.c:655-674`); no new registry or lock is introduced. Real group-heal
proof uses two PCs following separate actual same-description mobile leaders,
then joins their held leader edges. NPC group-flag support is not expanded.
Name fallback exists only for standalone legacy spell adapters without
FollowingBody; live World exposes that method. Teleport's legacy CharTransfer
fallback and no-engine cleanup adapters likewise do not run with live World.

Hunting/memory stay on the retained frontier. The jail callback first requires
a concrete Player victim; its hunting label comparison designates that unique
PC and does not choose among NPC duplicates. Other noncombat hunting decisions,
mobile riders and player-name persistence/event consumers remain separately
retained; this is not certification of every lifetime subsystem. The literal
PC-only prep audience lookup also designates a unique PC. The broader sweep
must include pkg/spells, pkg/command, combat_wire.go and death.go explicitly:

```sh
rg -n '(GetName\(\)|\.Name|casterName).*(==|!=)|(==|!=).*(GetName\(\)|\.Name|casterName)' pkg/spells/damage_spells.go pkg/spells/affect_spells.go pkg/command/skill_commands.go pkg/game/death.go pkg/game/combat_wire.go
```

The controls script now includes breath-identity, loot-identity, skill-audience
and group-leaders. Each has compiled green/revert/restore assertions. Existing
proof HEADs describe their precommit lab base, with changes present in the
working tree; the final census records the actual committed source tip.
The stack is rebased onto origin/main 7f9999464, including the reviewed transport,
channel-color and deployment changes. All checkpoint references will be updated
consistently; final CI is dispatched explicitly because sibling-base PRs do not
trigger the main-only pull_request workflow. Merge still requires all four
stacks together and Zach's complete-tip playtest.

The expanded audit also reproduced incorrect kill credit, PK credit and outlaw
flags on an unrelated PC matching an NPC killer's description. Live HandleDeath
now passes its actual killer through recordKill and player-death PK bookkeeping
(`src/fight.c:1671-1691`). Name remains only for output/log/event labels. Legacy
direct player-death adapters retain their uniquely named PC fallback; live combat
always supplies its body, including nil or NPC. NPC counters are not expanded.
Controls death-credit and pk-credit restore each erroneous lookup independently.
