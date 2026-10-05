# D2 Train 1: shoot lookup and refusals

Stop tier, per Zach's approved #1792 sequence. Base origin/main 28ab421e9.
One commit per case: shoot.target-fallback, shoot.pc-level-window,
shoot.target-fighting. Their final five-seed evidence comes from the train-tip
combined census; successful/missed target outcomes remain blocked for Train 2.
The measured blocked count is 24 after these three rows are resolved.

## C paths and proof boundaries

C do_shoot calls get_char_room, then falls back to world[targ_room].people
(src/act.offensive.c:862-872). get_char_room counts numbered abbreviated
keyword matches in room-list order without visibility or self/player shortcuts
(src/handler.c:865-881,115-134). char_to_room prepends (handler.c:532-539).
The new ResolveShootTarget merges real PC and NPC snapshots by descending
RoomEntrySequence. No CAN_SEE, synthetic bodies, new registry or persistence.
The empty-room projectile placement remains the existing canonical movement.

TestShootTargetCMatcherAndRoomOrder proves invisible bodies, mixed PC/NPC
ordinals and abbreviations, unknown/0./invalid/out-of-range/self/me fallback,
empty room and real PlayerTransfer re-entry. The hidden fixture is asserted
invisible to an ordinary viewer. TestShootFallbackReachesSentinelWithoutConsumingProjectile
exercises the actual command. The old handler produced Twang and moved the
arrow; the corrected handler reaches sentinel refusal without RNG/wait changes.
shoot-lookup-depth's C blocks exercise four lookups and retained inventory.

C's PC level refusal is strictly outside 10..30 (act.offensive.c:881-885).
TestShootPCLevelWindow proves 9/31 refusal and 10/30 entry into the target
outcome. Only the gate is claimed for 10/30; its outcome still uses the old
implementation pending Train 2. shoot-pc-window-depth proves the two refusal
bytes to the actor and silence to the peer with retained projectile inventory.

FIGHTING(to) is checked after the PC window, before sentinel protection
(act.offensive.c:887-900). TestShootTargetFightingRefusalAndOrder covers PCs,
ordinary mobs, sentinels and protected-level PCs, preserving HP, position,
room, fighting target, projectile, RNG and wait. shoot-fighting-depth uses a
real forced PC hit on the sentinel, then proves both bodies' shoot refusals.
There is no manually fabricated fighting fixture in the oracle scenario.

## R5h controls and retained evidence

All compiling assertion-based controls restore to green:
- Old command lookup fails the command fallback test; remove fallback,
  reverse RoomEntrySequence order, break abbreviation matching, or filter
  invisible bodies and the lookup proofs fail.
- Old PC path fails at 9/31; remove the window or change either inclusive
  boundary and its proof fails.
- Old fighting path fails for PC/NPC/sentinel; disable it, move it after the
  sentinel guard, or before the level guard and its proof fails.

Logs and raw C blocks:
~/Archives/darkpawns/oracle-runs/2026-10-05/dp-1371-shoot-gates-proofs/
Targeted development runs: dp-1371-shoot-lookup-iterate (3/3 CLEAN),
dp-1371-shoot-pc-iterate (1/1 CLEAN), dp-1371-shoot-fighting-iterate (1/1 CLEAN).
Fixture build/format mistakes were corrected before the retained assertion
controls; they are not counted as failing-first proofs.

## Other readers and combat/death seams

ResolveCharInRoom/At and every generic visible consumer retain their behavior.
The adjacent-room visible resolver now has no production caller; it is retained
rather than repurposed. The lookup reads the existing runtime arrival sequence
also used by mob special dispatch, room displays and Lua RoomFields; it never
writes it. Name/keyword matching, entry ownership, login, admin, session maps,
transport routing and persistence are unchanged. PC/NPC getters use real bodies.

The guards return before DoShoot/sendSkillResult, so no combat/death seam is
introduced by Train 1. The old targeted outcome path remains for Train 2's
six blocked cases, including its known damage/wait/relocation defects. This
train does not certify them. The new early refusals do not enter that path.
No kill credit, PK flag, death counter or corpse claim is made here.

## Lock acquisitions

- ResolveShootTarget: World.mu.RLock → copy player/mobile pointers → unlock.
  Each PC GetRoom/GetRoomEntrySequence/GetName takes Player.mu.RLock separately,
  after the world lock is released. Each NPC GetRoom/GetRoomEntrySequence takes
  MobInstance.mu.RLock separately. NPC keyword data comes from the existing
  immutable atomically loaded Proto snapshot. Sorting/matching takes no locks.
- PC window: Player.GetLevel takes Player.mu.RLock for each boundary read;
  NPC IsNPC is a type property, so the gate short-circuits without mob level.
- Fighting refusal: Player.GetFighting or MobInstance.GetFighting takes its
  body's RLock and releases it before SendMessage. The existing sentinel
  HasFlag/HasMobFlag RLock remains after that guard.
- No new writer lock, manager lock, engine lock or playerLifecycleMu. No world
  lock is held during body getters, messages, object movement or old outcomes.
  The pre-existing empty-room MoveObjectToRoomFront path is unchanged.

Normal gates and focused race checks are recorded alongside the controls.
Each case's commit ran build, vet, all tests and lint independently, plus fmt,
diff check, depth, units and string census. One combined census at the final
commit supplies all three scenarios at seeds 1,2,3,5,8 and the full corpus.

## Retained class follow-up (not repaired in this train)

The complete C get_char_room caller search has two other producers:
pet_shops (src/spec_procs.c:1864) and assassin (src/spec_procs2.c:871).
Both are reachable room assignments (src/spec_assign.c:618 and the existing
Go RoomSpecAssignments at 21235/8114). Go pet_shops scans GetMobsInRoom's
unordered snapshot and substring-matches short descriptions; assassinRoster
returns PCs first and an unordered mob list. These do not preserve C's named
room-list ordering; pet lookup also lacks get_number/keyword abbreviation.
Proposed follow-up: prove those storage-room callers' matching/list order in
separate spec-proc work. Do not repurpose shoot's fallback for them: C's other
callers do not fall back after a named miss. Their current measured proofs are
not broadened by this shoot train, and no code/row is changed here.
