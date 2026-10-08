# DP-1371 D7 game-diagnostic train

Stop tier: the spell-world consumer bridge and the RawKillVerb result field exceed the mudlog-only exception. Six sites repaired at their live boundaries; two retained frontiers below. Fresh main 9a0a0f343. R1/R3/R5e/R5g/R5h.

## Death-cry NOWHERE frontier (case 7)

`src/fight.c:506-516` logs BRF/31/file TRUE and places the body in C room index 0. However, `raw_kill` guards NOWHERE before any work (`534-539`), so this producer cannot be justified by simply calling raw_kill with an unplaced body. This matches Go's guard; it is not a demonstrated raw-kill divergence.

The other direct callers are `src/act.movement.c:293,298`. The rider has NOWHERE checks after entry and greet (`266-287`). A retained mount is moved to the destination before those callbacks (`193-203`); the mount pointer survives their execution. Lua teleport validates the destination (`src/scripts.c:1527-1536`), while extchar requests deferred extraction (`480-489`). This is not yet a complete proof that every nested callback preserves a valid mount room, and the default remains blocked. Do not inject an invalid descriptor/body and call it a valid-play oracle proof.

Next: finish the whole callback/retained-mount audit. If no legal path exists, add a fail-capable bounded caller proof and exclude this producer from valid play. If a legal path exists, demonstrate it in unchanged C first and design the body-placement boundary across combat/world death-cry readers, using the existing room-index conversion and body ownership. No second registry, guessed vnum 0, approximate log, or proposed divergence.

Retained C caller sweep: `dp-1371-mudlog-game-diagnostics-proofs/C-deathcry-sites.txt`.

## Integration and resolved boundaries

The train was parked while #1848 repaired the independently reproduced lethal
spell message ordering. It now includes merged main `be5adf13d` (including
#1848 and #1849's lockout manifest correction). Six production cases retain
separate commits; the six controls can be rerun independently:

    python3 docs/fidelity/depth/handoff/2026-10-07-dp-1371-game-diagnostics-controls.py --case milestone --output /absolute/evidence/path

Replace `milestone` with `cross-room`, `protection-evil`, `protection-good`,
`death-trap`, or `spike-stake`; omit `--case` to run all six. Each control uses
an overlay, requires a compiling named assertion failure, and restores green.

| Case | C contract | Boundary / other readers | Locks at producer |
|---|---|---|---|
| Kill milestone | `src/fight.c:1300-1309`: NRM/31/file FALSE, after reward/blessing | Actual World death-credit caller. Existing reward draws, heal and kill bookkeeping retained; independent immortal oracle observer. Legacy combat `CounterProcs` not activated or claimed. | World snapshot from `AllPlayers` and each body setter release before producer; `recordKill` releases killer mutex before `counter_procs`. |
| Cross-room damage | `src/fight.c:1318-1333`: NRM/31/file FALSE, after corpse guard | Shared live `World.DamageRefused` rejects before mutation; immortal and corpse silence. Callback-free combat fallback stays separate. State test is the proof: ordinary commands cannot request cross-room damage directly. | Body getters release; gate takes no world or player lock across producer. Combat pair snapshots are released before `performOneHit` callbacks. |
| Protection evil | `src/magic.c:1142-1148`: BRF/31/file TRUE, after caster warning, before caster raw kill | Real `MagAffects` and World's forwarding consumer; separate aligned victim proves caster naming and caster death. No affect, raw-kill or credit changes. | Spell branch holds no world/body lock. Warning send completes before producer. |
| Protection good | `src/magic.c:1162-1168`: same flags and order | Same shared bridge, independently reverted good arm and safe-branch silence. | Same as evil. |
| Player death trap | `src/act.movement.c:261-292`, `src/utils.c:145-148`: BRF/31/file TRUE | Existing player `DoMove`/`MovePlayer` death-trap boundary after look and before cry/extraction. Actual destination VNum/name. Mounted and mobile callers remain frontier; no whole-movement parity claim. | `MovePlayer` releases world/player mutexes before `deathTrap`; destination lookup releases before delivery. |
| Spike/stake | `src/new_cmds.c:1155-1175`: BRF/31/file TRUE after acts before PK/deaths/raw kill | `sendSkillResult` real command tail; carries subcommand noun in `RawKillVerb`, never guesses from item keywords. Unit tests both verbs with dual-keyword weapon, separate actor/victim/witness and unchanged counters/extraction. Oracle stages spike on an innate werewolf NPC; stake is unit-proven. | Command dispatch and result delivery hold no manager/lifecycle/world/body mutex across this call; target room lookup and getters release before producer. |

MudLog's file write uses the existing log-writer lock, then releases it. The
provider registration lock is a snapshot only. Manager `EachSession` snapshots
sessions under its mutex and releases it before callback/delivery; its per-session
player snapshot also releases first. Filtering reads observer body flags/level,
then the existing body delivery routes through switch-aware routing, heartbeat
staging and snooping. No new lock is held across those reads or delivery. Unit
lock probes exercise world/killer availability at milestone, cross-room,
protection and death-trap boundaries; focused race covers the new cases.

## Hunting design frontier (case 8)

C `set_hunting` at `src/utils.c:708-729` clears old state; mobile prey retain
concrete body identity without logging. Positive-ID player prey emit CMP/31/file
FALSE **before** assigning the player ID. Nil prey clear without logging; the
actual existing mobile edge short-circuits. These are identity and ordering
semantics, not merely a missing producer.

Go `MobInstance.SetHunting` takes a name and clears `HuntingID`/`HuntingMobID`.
`World.SetHunting` calls it under the world mutex. The live Lua bridge resolves a
body and then discards it to a name; `huntVictim` uses `HuntingMobID` for the
special equipment edge and otherwise resolves a PC name. It does not read
`HuntingID`. Retaliation, assassin hire, zero-CHA equipment, graph cleanup and
Lua are all readers/callers to migrate together; legacy string APIs need an
explicit adapter boundary. A name-based log would confuse same-description mobs
with PCs, would miss C's positive-ID condition, and could run under world/body
locks before shared delivery.

This remains blocked. Next design must preserve concrete `(kind, ID)` or body
identity, use existing world ownership without a second registry, list every
reader, and prove real duplicate-description mobile prey, a same-name PC/mobile,
PC identity across rename/extraction, clear/same-edge behavior and log-time
pre-assignment state. Audit player hunters too: Lua's C binding accepts generic
characters (`src/scripts.c:1345-1358`), while Go's current bridge accepts only a
mobile hunter. This is a separate reviewed design train, not a seventh
approximate producer in this PR.

## Final proof and validation record

All five new vehicles match unchanged reference C at seeds 1,2,3,5,8 (25/25).
The C transcript contains each intended diagnostic; the milestone's independent
observer receives the blessing before the NRM log, protection warnings precede
raw kill, the mortal death trap records actual room 8008, and spike's independent
BRF observer receives the authored act, diagnostic and death cry in that order.
Five manifest rows now use standard `vehicle@1,2,3,5,8`; the sixth cross-room
boundary retains its state-level unit proof. Stake's chosen noun and PC counters
remain explicit unit scope, rather than an unstaged oracle claim.

Retained integration proofs and six compiling revert triples:
`~/Archives/darkpawns/oracle-runs/2026-10-07/dp-1371-mudlog-game-diagnostics-final-proofs/`.
The historical spell stop is retained there as `historical-spell-message-stop.md`;
#1848 resolves it. Earlier per-case proof folders retain their original HEADs;
the upcoming combined census records this train's final committed source.

Formatting, build, vet, full tests, game tests, clean-cache lint (0 issues),
focused race, depth gate, unit gate and string census passed during integration.
No production access, oracle edit, repin, dropped seed or new divergence.
