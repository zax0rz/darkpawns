# Combat body identity prerequisite for Shoot Train 2

**Stop tier: design first.** Changes combat identity, cleanup and session-facing
consumers. This PR contains documentation only. Implementation requires review
of this design; it earns no completed depth cases. Shoot Train 2 remains parked
in `/home/zach/dp-p4-shoot-outcomes`, branch `sol/p4-shoot-outcomes`, and will be
finished on top of the repaired engine after that prerequisite merges.

Base audited: `1e2ba94fa` (origin/main, including #1794 and #1795).
Authorization: Zach's approval of the combat-identity prerequisite after #1792.
R1, R3, R4, R5b, R5e, R5g and R5h govern the repair.

## Problem and C contract

Two real mobs spawned with `World.SpawnMobQuiet` from one prototype have
separate bodies/instance IDs and equal `GetName()` short descriptions. If one
is already fighting, the parked ranged entry aborts the second mob's immediate
retaliation when `startCombat` rejects that attacker name. The retained test
reports zero swings where C performs one. Its fixture and assertion failure
are in `~/Archives/darkpawns/oracle-runs/2026-10-05/`
`dp-1371-shoot-outcomes-proofs/shoot_duplicate_real_diagnostic_test.go` and
`duplicate-real-body-red.txt`. That test proves the failure in the parked
implementation, not a full census result or an original-main shoot verdict.

This is reachable with ordinary duplicate mobile loads: shipped zone 163
loads prototype 16303 twice into room 16304 (`lib/world/zon/163.zon:81-83`).
Go's current `CombatPairKey` uses names; enrollment, combatOrder deduplication,
round deduplication, target lookup, parried state and cleanup reuse those names
(`pkg/combat/engine.go`). Replacing a colliding entry or stopping by name would
also disturb the first mob's combat. Removing only the enrollment refusal
would leave later turns and callbacks ambiguous.

C uses actual bodies throughout:

- `src/fight.c:207-223`: set_fighting prepends the actual char, removes sleep
  affects, assigns `FIGHTING(ch) = vict`, then sets posture.
- `src/fight.c:230-254`: stop_fighting removes that body; when its dead/fled
  target has another live attacker fighting it, it may retarget and return.
  It is not an unconditional delete-every-pair operation.
- `src/fight.c:1314-1458`: protections, jail, charm redirect, switcheroo,
  enrollment, memory/hunting and master separation operate on char pointers.
- `src/fight.c:1022-1085,1770-1888`: skill messages and hit read the actual
  attacker's wielded object and victim's state, preserving draw/output order.
- `src/fight.c:1898-2032`: violence walks combat_list in order, with per-body
  defense state, target and native special invocation.
- `src/fight.c:534-583`; `src/handler.c:1144-1153`: raw death and extraction
  stop the actual character and fighters pointing to it.
- `src/handler.c:504-550`: leaving/entering a room compares body references
  when stopping and rechecking fights.

These source sites were read for this design (R5g). This is a fidelity repair,
not a new approved divergence or a replacement for the retained C3 lifecycle
and mobile-rider work.

## Chosen representation: concrete bodies in the existing engine

Use the existing `combat.Combatant` interface carrying the canonical concrete
`*game.Player` or `*game.MobInstance`, as an opaque body reference. Pair keys
become `{Attacker Combatant, Target Combatant}`; combatOrder membership, round
seen sets and parried state compare/key those references. This is Go pointer
identity across comparable interface values, matching C char pointers.
Production entry points accept only canonical pointer-backed bodies; tests use
pointer-backed fixtures. Validate nil/typed-nil and comparable pointer-backed
references at entry, and reject value-backed adapters before map insertion.
There is no uintptr key, bare numeric ID, prototype vnum or display-name key.
A player and mob with the same numeric ID remain distinct, as do a retired
body and its later restored replacement. Do not manufacture combatants or add
an ID-to-body registry. World and session registries keep their existing roles;
the engine's pairs/order remain its sole combat membership structures.

Store the authoritative fighting opponent on each body as `Combatant`, using
`GetFightingBody()` and `SetFightingBody(Combatant)` under the body's existing
mutex. This permits damage paths to establish C's FIGHTING before an engine
pair exists. `StopFighting` clears that reference with its existing posture
semantics. `GetFighting()` becomes a display projection: snapshot the opponent,
release the lock, then read its name. Never nest two body locks to render it.

Migrate all production `SetFighting(string)` calls and direct fighting-field
writes. Remove the string-backed authoritative fields rather than maintaining
two writable truths. Boolean/empty checks become reference-presence checks;
`GetFighting()`/`GetFightingTarget()` may remain read-only display helpers.
The mob's existing `Target *MobInstance` is an AI/scripting relation: audit its
callers, but do not substitute it for the general fighting reference or silently
merge its lifetime with FIGHTING. No reference enters a save record or JSON
payload; fighting remains transient runtime state.

Engine APIs `StopCombat`, `IsFighting`, `GetCombatTarget`, `GetCombatStatus` and
`MarkParried` receive a body, including local interfaces in game/command and
mock engines. Do not leave a name-taking overload used by production combat.
Command text still selects a target once through the existing C matcher and
visibility/ordinal gates; after that, retain the selected body. Empty-argument
combat commands use the stored opponent, not a fresh room-name search.

Implement C stop_fighting's retarget-or-remove behavior separately from full
extraction cleanup. A full cleanup must remove only entries involving the
retired body, clear only opponent references equal to it, and prune parried
state without erasing another same-description mob. Ordinary target death/flee
must preserve any C-eligible retarget, not clear every fight sharing a participant.
Keep the established head-prepend order and draw sequence; no sorting by ID,
name or map iteration to rebuild combat_list.

## Callback migration: exhaustive inventory

Every body-bearing callback receives concrete bodies, including callbacks
currently implemented only for players or currently unwired. Methods available
on Combatant may be read directly; game-only operations type-switch the supplied
body. They must not resolve it again with GetPlayer/GetMobByName/room-name scans.
Keep existing zero/no-op defaults and supported behavior while changing identity;
implementing an unrelated unwired feature needs its own case and proof.

The table inventories all fields of `pkg/combat/callbacks.go:GameCallbacks`.
The existing package-global callback accessor is not redesigned here.

| Callback fields | Identity migration / audit consequence |
|---|---|
| `Broadcast`, `SendToChar`, `SendRaw`, `SkillMessage` | Recipients and excluded participants are bodies (explicit exclusion slice, not concatenated names). Message templates still render names/pronouns and preserve color, CRLF, sleeping/visibility and RNG behavior. Skill messages take attacker/victim bodies. |
| `BroadChat` | Preserve actual speaker through dispatch and rendering; the text remains text. |
| `GetRace`, `GetRaceHate`, `GetAlignment`, `SetAlignment`, `GetSex`, `GetHP`, `GetLevel`, `IsNPC`, `GetSkill`, `GetColorLevel` | Read/mutate the supplied instance, with existing PC/NPC gates and defaults. No same-name PC-first lookup. |
| `HasAffect`, `HasAffectStr`, `RemoveAffect`, `RemoveAllAffects`, `RemoveTattoo`, `ClearNightbreed`, `ForgetVictim` | Preserve instance-specific affects and teardown; forgetting a victim carries its actual body to existing memory cleanup. Player-only hooks stay player-only. |
| `HasPlrFlag`, `SetPlrFlag`, `HasPrfFlag`, `HasMobFlag`, `HasMobVNum`, `MobHasJailGuardSpec`, `HasScriptFlag`, `IsShopkeeper` | Inspect exact body/prototype/spec, never another matching description. Flags and unsupported cases retain their existing contract. |
| `DamageRefused`, `GetRoomCombatants` | Already body-bearing; replace self/FIGHTING name comparisons inside protections. Current room callback concatenates players then mobs; merge snapshots by descending existing RoomEntrySequence to reproduce C room-list order for redirects, without allocating new identities. |
| `GetFollowing`, `StopFollowerOfMaster`, `JailGuardSubdue` | Return/compare actual master and operate on exact guard/victim. Existing follower and mount attachment helpers must preserve a supplied body; a name-backed edge cannot guess among duplicate NPCs. Audit/write those combat-relevant links at attach/detach, using existing relationship fields, no new registry. |
| `IsMounted`, `Dismount`, `Unmount` | Pass the affected body to existing mount teardown, preserving exact mount where already retained. Do not pick a same-description mount or expand C3 mobile-rider support in this prerequisite. |
| `GetWeaponInfo`, `GetWeaponDescription`, `GetDrunk` | Use exact attacker's equipment/conditions, including the shared C-slot mobile equipment path; preserve existing attack-type and damage arithmetic. |
| `GainExp`, `GetExp`, `GetKills`, `SetKills`, `GetDeaths`, `SetDeaths`, `SetLastDeath`, `GetPks`, `SetPks`, `GetConstitution`, `SetConstitution` | Awards and counters carry recipient bodies; retain supported PC/NPC semantics and bookkeeping order. No string round trip after a victim retires. |
| `RawKillNPC`, `MakeCorpse`, `MakeDust`, `ExtractChar`, `RunDeathScript` | Keep victim/killer bodies through raw death, corpse and deferred cleanup. RawKillNPC already accepts a body; migrate remaining name-only seams. |
| `GetFollowersInRoom`, `GetMasterInRoom`, `GetFellowFollowersInRoom`, `CountGroupMembers`, `ApplyToGroupMembers` | Body leader/member enumeration and callback recipients. Remove GroupGain's synthetic NewNamedCombatant construction; enumerate existing real members. Preserve existing payout/order/formulas. |
| `GetGold`, `SetGold`, `JunkInventoryItems`, `PerformCommand` | Exact owner/actor through inventory, loot and command dispatch; command arguments remain text. No broad NPC command port here (C1 stays separate). |
| `StopFollowerOfMaster`, `GetWimpyLev`, `DoFlee`, `DoRetreat` | Exact victim/master and movement actor; stop_follower is listed with following above as well. Preserve existing flee/retreat gates, waits and messages. |
| `IncreaseMaxStat` | Exact counter-proc recipient. |
| `Log`, `HasRoomFlag`, `GetAdjacentRoom`, `HealAllPlayers` | No body identity parameter to migrate; logging strings, room IDs and global operations retain their current contracts. |

Engine-owned hooks need the same treatment: `DamageFunc` takes the damaged
body; `ScriptFightFunc`/`ScriptDeathFunc` take actual actors/victims; existing
`MessageFunc`, `DeathFunc` and `MobSpecialFunc` already carry bodies. Migrate
BroadcastFunc's exclusions. `OnRoundEnd` remains unchanged.

`pkg/combat/fight_core.go` and `skill_messages.go` must thread bodies through
all cb helpers, embedded/fallback message renderers, EmitSkillMessage, group
payout, raw death and protection fallbacks. Name equality currently also treats
two distinct same-description NPCs as self-damage; replace it with body equality.
Update callers in spells/damage_spells, game/spec_fighter, game/room_activity,
and command/session skill-message emission. No name-only compatibility adapter
may recover a combat body by searching. Standalone message tests may provide
concrete fixture recipients without registering them as real combatants.

## Other readers and call-path audit

The following inventory comes from production searches for engine entry/query
APIs, Get/SetFighting, direct Fighting/FightingTarget fields and callback wiring.
Implementation must repeat the searches against its final source and classify
remaining names as command text, display or a separately owned relationship.

| Reader / writer family | Required migration or preserved boundary |
|---|---|
| `pkg/combat/engine.go`, `fight_core.go`, `combatant.go`, `skill_messages.go` | Pair keys/order/round seen/parried; self checks; target and redirect resolution; enrollment including EnrollAfterDamage; protections; message/callback/body cleanup. No unordered-map target search. |
| `pkg/game/combat_wire.go`, `damage_gate.go`, `damage_stubs.go`, `death.go` | All callback closures; direct spell/skill enrollment; jail/charm/shopkeeper gates; raw death and killed-body cleanup. Snapshot real victim before retirement and retain it through callbacks. |
| `pkg/game/char_mgmt.go`, `world.go`, `world_movement.go` | Pending players/mobs and ExtractMob, room departure/transfer and mutual fight checks; clear exact references and engine membership at the established boundary, without changing deferred timing or inventory ownership. Current transfer loops resolve mobiles by name: body-taking PlayerTransfer/MobTransfer must carry the supplied object through the helper. |
| `pkg/game/ai.go`, `world_player.go`, `mobact.go`, `graph.go`, `mobprogs.go`, `spawner.go` | Aggression/recovery gates, target writes, NPC switches/rescue/assist and reset eligibility; exact mob membership and target, preserve existing command/pulse order. Raw reset eligibility checks become body-presence tests. |
| `pkg/game/spec_procs*.go`, `spec_fighter.go`, `spec_paladin.go` | mobFightingTarget/cityguard/teleport/rescuer/alliance, switcheroo and parry use exact targets; no fallback GetMobByName for FIGHTING. Native special dispatch retains its actual mob. |
| `pkg/game/skill_*.go`, `skills*.go`, `skills2.go`, `ambush.go`, `target.go`; `pkg/command/skill_commands.go` | Rescue/assist/kick/bash/disarm/circle/backstab and default-target consumers; replace relational name tests and setters, preserve explicit command name/ordinal selection. FindFightingTargetInRoom cannot reselect the first matching description for an already selected fight. |
| `pkg/game/player*.go`, `mob.go`, `limits_misc.go`, `other_utility.go`, `other_mount.go` | Runtime reference storage/access, healing/idle/quit/binary fighting gates and teardown. No second string-backed authority. |
| `pkg/game/look.go`; `pkg/session/wiz_stats.go`, `wiz_info.go` | Render opponent from retained body under existing PERS/CAN_SEE rules; peace command stops each concrete body independently. Names remain display bytes. |
| `pkg/session/cmd_combat_basic.go`, `combat_cmds.go`, `movement_cmds.go`, `cast_cmds.go` | Resolve selected/held bodies once; engine query/stop/parry/movement/flee/assist/casting use them. Preserve command matching and visibility. |
| `pkg/session/manager.go`, `agent_vars.go` | Wiring, health/target dirty notifications, script invocation, death/autoloot, disconnect/extraction. Compare actual target to damaged body, avoiding updates for the wrong duplicate. Names/IDs in existing structured payloads stay unchanged; no serialized pointer. |
| `pkg/game/world_scriptable.go`, `pkg/scripting/engine.go` | Combat fight/death triggers use the supplied mob and actor rather than rediscovering a matching scripted mob. Lua is_fighting/target consumers retain canonical body handles. Preserve currently supported script context types; C1/C3 unsupported bindings are not certified by this repair. |
| `pkg/game/follow.go`, `other_mount.go`, `combat_wire.go` | Trace combat master/mount inputs to existing ownership writes. Player-name links can resolve the uniquely held player once; NPC links must retain a selected body. Do not invent missing mobile-rider behavior or replace all noncombat navigation/hunting systems. |
| `pkg/db`, session login/switch/persistence and admin APIs | No stored fighting reference found in db; no schema/save-format change. Reconnect/switch must retain the actual attached body, not original login identity, when calling combat APIs. Cleanup captures that body before detachment. Admin listings and agent/GMCP payload schemas retain display identities. |
| `pkg/events/queue.go`, logging, Discord death notice, metrics | Event queue numeric source ownership is a distinct retained class: do not rekey it as combatPairs. Preserve existing event cancellation and logging/display text. This repair must prove it does not clear another combat; it cannot claim a complete event-identity audit or alter unrelated notifications. |

Recipient lookup must use the existing attached-body ownership helpers for
players and MobileMessageSink for switched NPCs; snapshot the descriptor under
the manager lock and deliver after release. In particular, current name-based
combat broadcasts split exclusions on spaces, which cannot represent an NPC
short description. Explicit body exclusions fix that ownership ambiguity while
retaining the approved heartbeat staging, snooping and close barriers. No
second descriptor index or new output transaction is introduced.

The audit distinguishes membership identity from command matching. General
GetMobByName helpers remain available to name-taking commands; combat callers
already holding a body must not use them. Existing unsupported mobile command,
script actor and rider behavior stays on the C1/C3 frontier with an explicit
handoff; identity migration cannot introduce a silent no-op for a supported path.

## Cleanup and locking contract

Do not acquire playerLifecycleMu in ordinary combat, movement or world ticks.
CombatEngine.mu protects existing pairs/order/parried only. Snapshot membership
and body references then release it before world/session lookups, output,
script/spec callbacks, movement, damage, corpses, inventory work or persistence.
Body reference access acquires one body's mutex at a time. If membership
mutation uses engine-to-body acquisition, there must be no reverse body-to-engine
caller; document exact acquisitions in the implementation PR and prefer a
snapshot/mutate split where existing world cleanup holds World.mu.

World extraction/transfer currently holds World.mu across several operations.
Gather affected body references under that lock, do world-owned mutations at
the existing C boundary, and invoke engine cleanup after releasing World.mu.
No engine callback into World while engine.mu is held. Existing session
lifecycle-to-manager/world order remains unchanged; capture attached bodies
before unregister/unswitch and invoke body-aware cleanup without introducing
an engine-to-lifecycle edge. State the retained extraction visibility/timing
explicitly: this prerequisite repairs combat ownership, not C3's entire
mobile extraction lifecycle.

A round snapshot must revalidate that a body still has its concrete opponent
before its turn, and must tolerate retirement/redirect during a prior callback.
Do not let a stale snapshot re-enroll an extracted body. No borrowed field may
be read unlocked because names previously appeared immutable. Add race coverage
for round-versus-stop/extraction and fighting-reference getters versus setters.

## Implementation sequence and fail-capable proofs

Implement as a separate stop-tier prerequisite, with bounded commits and a
combined census at the tip. If the shared migration cannot be kept reviewable
within train limits, split prerequisite PRs at coherent buildable boundaries;
do not merge a half-migrated name/body system or resume shoot before all seams
are proven.

1. Introduce reference accessors and body-aware engine membership/query APIs;
   migrate fighting writes/readers and remove name-backed authority. Keep
   existing ordering/posture/gates. Real World fixtures demonstrate two
   same-description mobs fighting separate real players simultaneously, both
   immediate swings and subsequent turns, exact targets and four memberships.
   Include equal player/mob numeric IDs, same-room and different-room duplicates,
   self-versus-distinct-body checks, and a restored body replacing a retired one.
2. Migrate all callbacks/message consumers and non-engine default targets.
   Give duplicate mobs different wielded weapons, affects, flags/protection,
   race/sex, scripts and HP to prove each reader selects its actual body.
   Assert observer exclusions, sleeping audiences and switched NPC/player
   recipient routing; color and next RNG draw remain identical. Add a focused
   failing-first test for each callback family and each changed reader, not
   just one integration happy path. Preserve unwired defaults with explicit
   controls instead of pretending they demonstrate live behavior.
3. Migrate StopCombat/retarget, movement, death and extraction. Kill one actual
   duplicate while both fights run; assert the other pair, opponent reference,
   ordered turn, weapon, corpse/inventory and callbacks remain intact. Repeat
   for noncombat raw kill, ExtractMob/pending extraction, player extraction,
   movement/flee, disconnect and peace. Test C's dead/fled-target retarget path,
   one-way combat and redirect to a same-description mob. Exercise round
   callbacks that retire a body and -race for concurrent cleanup/access.
4. Add `combat-duplicate-bodies-depth`: real duplicate shipped mobs in one
   room, two real player peers and an independent observer. Use existing ordinal
   command matching (`kill 1.<keyword>` / `kill 2.<keyword>`), verify both C
   fights and target HP/state, pump violence, kill/extract just one, then pump
   again to observe the survivor's independent attack. Equip distinguishable
   weapons through existing supported setup so attack verbs expose callback
   misrouting. Inspect raw C transcripts to prove both branches occurred; never
   count an invalid setup or quiet spawn as a green. Run seeds 1,2,3,5,8 and
   retain actor/victim/observer bytes and downstream draws. Fixture viability
   must be established before publishing an oracle claim; no oracle/harness
   changes, dropped seeds or pins are authorized by this design.
5. For R5h, revert key/member/target identity separately to names, remove kind
   discrimination in an alternate ID-key control if implemented, restore a
   name-resolving weapon/affect/protection/script/message callback, and restore
   name-wide cleanup. Each control must compile, fail an assertion on the real
   fixture, then restore green. Pointer choice's equal-numeric-ID test proves
   the mandated cross-kind separation without relying on an unused ID adapter.
6. Run full normal gates plus fidelity-depth/fidelity-units/string-census;
   targeted oracle runs while iterating, then one retained `--combined` census
   at the final committed tip. Explain any scenario change with paired C bytes;
   reproduce suspected pre-existing failures on origin/main. Implementation PR
   lists actual callback/reader audit, exact lock acquisitions, revert triples
   and both combined verdicts. Open and stop for Zach.

After the prerequisite merges, integrate it into the parked Shoot Train 2
branch, remove the collision workaround/refusal seam in favor of the proven
body-aware hit enrollment, rerun its outcomes/proofs and final combined census,
open that separate stop-tier PR and stop. No shoot row is reclassified by this
prerequisite design or by its mere implementation.

## Validation of this design PR

Only this note changes. All 74 GameCallbacks fields were checked against the
inventory above. Source-reader and C citations are an audit/design, not completed
behavior proofs. Normal build, vet, full tests, game tests, formatting and lint
passed, as did fidelity-depth, fidelity-units and string-census. Gate outputs
are retained in `~/Archives/darkpawns/oracle-runs/2026-10-05/`
`dp-1371-combat-identity-design-proofs/`. No census was run: AGENTS.md's docs-only
rule applies. No production, scenario, manifest, governing document or oracle
file changes; the six shoot outcome rows stay blocked.
