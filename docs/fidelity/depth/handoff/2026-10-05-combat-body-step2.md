# Combat body identity — stack 2/4: callbacks and messages

**Stop tier:** combat callbacks, descriptor recipients and identity consumers.
Base: stack 1, #1798 (`c1c1bee4c`). Do not merge this or #1798 independently.
The approved #1796 sequence merges together after stack 4's combined census
and Zach's duplicate-mobile live playtest. Shoot Train 2 remains parked.
R1, R3, R4, R5b, R5e, R5g, R5h apply.

## Boundary

A new body-bearing SendText hook preserves the ordinary text-event envelope
for formerly direct combat messages; SendToChar retains combat-event framing.
This closes the remaining direct Player.SendMessage name round trip without
changing the global noncombat MessageSink.

Every body-bearing GameCallbacks parameter and nested group recipient is now
Combatant. This includes currently unwired fields: no name-taking compatibility
adapter remains. The exhaustive field inventory in the approved design applies;
logging text, command arguments, room IDs and global operations remain scalar.
Engine DamageFunc, ScriptFightFunc, ScriptDeathFunc and broadcast exclusions
also retain bodies. Default command targets, native specials, skill/spell message
callers and fighting presence retain the already selected opponent.

World callbacks type-assert the supplied Player/MobInstance. Equipment uses the
shared mobile C-slot path; affects, flags, protection self checks, counters,
script owners and actors cannot rediscover a same-description mobile or a later
same-name player. GetRoomCombatants merges existing room snapshots by descending
RoomEntrySequence. GroupGain enumerates actual member bodies, removing the
runtime synthetic namedCombatant (the old standalone fixture remains test-only).

Player follow attachment retains its selected master alongside the existing
social display name. Follower detach messages use that retained master too,
so a same-name PC is an observer rather than a substitute master. Legacy name-only edges can resolve a uniquely held player;
they never guess a duplicate NPC. Lua master fields share that boundary.
Lua isfighting reads the canonical kind/ID bridge handle (or the actual legacy
mobile table handle), retaining the existing incomplete target table shape.

Descriptor routing uses existing attachedBodyLocked and the existing switched
mobile descriptor fields under Manager.mu. MobileMessageSink and combat share
that ownership snapshot. Explicit exclusions compare bodies, so excluding a
mobile cannot exclude a same-name PC. Deliveries retain existing sendGuarded,
heartbeat staging, snoop and close handling. Sleeping room audiences are omitted;
direct damage-victim messages remain deliverable to sleeping victims. Names and
pronouns are rendered only after selection; color/control bytes and message
selector RNG order stay unchanged.

## Retained boundaries

This step does not complete C1 NPC command dispatch, script actor/table semantics
or C3 mobile riders, memory/hunting identity or extraction lifecycle. PC-only
weapon descriptions, conditions, counters, personal structured health updates,
flee/retreat dispatch and other previously unsupported mobile hooks retain their
zero/no-op defaults. NPC text can reach an existing switched descriptor, but
NPC hooks must not act on the wizard's original player. Unwired callbacks stay
unwired. No persistence schema, event-source registry or second body registry.

World transfer's remaining name comparisons and stop/death/extraction membership
cleanup are deliberately stack 3. Noncombat script APIs that explicitly accept
names, social following and command matchers are separate retained surfaces;
this step makes no claim that all names everywhere are identity-safe.

## Locks

- Engine pair/order snapshots release CombatEngine.mu before any callback,
  world/session lookup, script invocation, message or inventory work.
- World lookup snapshots take World.mu then release it before per-body reads.
  Callback reads/mutations use the supplied body's existing mutex. Fighting and
  master reads snapshot one body, release it, then read the other body's fields.
- Group enumeration snapshots the existing player registry, then reads members
  separately. Room sequence sorting reads existing bodies after the snapshot.
- Session recipient/audience selection takes Manager.mu.RLock, uses the existing
  attachment/transport fields and body getters, then releases it before delivery,
  dirty-var flushing, GMCP, snooping or sendGuarded. No playerLifecycleMu.
- Lua retains its existing scripting engine lock; kind/ID resolution snapshots
  World.mu and releases it before the fighting-body getter.
- Counter getters/setters now take Player.mu individually; no engine callback
  introduces a body-to-engine lock acquisition.

## Reproduce proof controls

From an isolated checkout of the stack-2 tip (no concurrent editors):

```sh
python3 docs/fidelity/depth/handoff/2026-10-05-combat-body-step2-controls.py \
  --output /tmp/combat-body-step2-triples
go test -race ./pkg/combat ./pkg/game ./pkg/session \
  -run '^TestCombatBody' -count=1
```

The replay requires green → compiled assertion failure → restored green for
state/affect/prototype callbacks, retired-player counters, armed/bare duplicate
weapons, room order, both default-target readers, script dispatch, protection
self checks, look/PERS, the live Lua handle, NPC and switched-PC descriptors,
body exclusions, target dirty notifications, direct text ownership, real group recipients, follower detach recipients and
defense audiences/sleep gates.
A compiler error cannot satisfy a red. Retained logs are under
`~/Archives/darkpawns/oracle-runs/2026-10-05/`
`dp-1371-combat-body-identity-proofs/step2-final-triples/`.

The message proof also asserts actor/victim bytes, pronouns, raw colors and the
next shared RNG draw. The live Lua proof exercises RunScript with a bridge
MeRef rather than a mock table. Other focused tests assert PC/NPC defaults,
selected masters and separate script actors.

## Reproducible leftover-name sweep

```sh
rg -n 'GetPlayer\(|GetMobByName|GetFighting\(|GetFightingTarget\(|NewNamedCombatant' \
  pkg/combat pkg/game/combat_wire.go pkg/game/world_bridge.go \
  pkg/game/world_scriptable.go pkg/game/world_movement.go \
  pkg/session/manager.go pkg/session/combat_cmds.go pkg/command/skill_commands.go
rg -n 'GetFighting\(|GetFightingTarget\(' pkg --glob '!**/*_test.go'
rg -n '(name|Name)[[:space:]]+string' pkg/combat/callbacks.go
```

Classify results rather than asserting an empty repository: test-only fixture
names/display assertions; read-only fighting display methods and wizard reports;
World transfer/cleanup pending stack 3; unique-player legacy follow lookup;
explicit name-taking noncombat/script/command input APIs and database login
lookups. No production NewNamedCombatant and no callback body parameter taking
string. The final combined census belongs only to stack 4, as approved.

## Gate outcome

All passed on the final source: make fmt; go build ./...; go vet ./...;
go test ./...; go test ./pkg/game/...; golangci-lint run ./...;
make fidelity-depth; make fidelity-units; make string-census; and the focused
-race command above. The 21 compiled controls pass. No census was started for
stack 2. Normal gate logs are step2-gate-*.txt alongside the retained triples.

## Review follow-edge repair before stack 3

NPC-led edges are reachable, so they are retained rather than classified away.
C `src/utils.c:463-498` attaches the actual `struct char_data *leader` without
an NPC-leader exclusion. `src/spells.c:406-458` rejects a charmed caster, not an
NPC caster, and attaches a successful charm victim to that caster. In Go,
`World.executeMobCommand`'s existing `follow` arm and `World.SetFollower` can
attach an NPC leader. Both now retain the selected leader body once; known-body
pet, tattoo and mount attachments do the same. String setters clear the retained
reference on detach. `cmdUngroup` now uses that setter instead of leaving a stale
private reference behind.

`combatFollowingBody` feeds `GetFollowing` (charm redirect and master separation
in engine/fight_core), following/master fields in the Lua bridge, and follower
stop messages. Its group enumeration consumers feed group awards. The existing
`GetMasterInRoom`, `GetFellowFollowersInRoom`, group recipient/count and follower
enumeration hooks remain PC-only: they previously used GetPlayer on their input,
so this repair does not certify or implement NPC group dispatch/assist. Go's
quiet spell adapter also retains its existing PC-leader gate. The NPC command
surface and the broader social/movement follower scans remain their separately
owned surfaces, audited again in stack 3 where cleanup uses actual bodies.

MobInstance now has a transient followingBody reference, protected by its own
mutex just like Player. There is no registry or persisted field. Tests use two
same-description NPC leaders, assert the exact combat callback and kind/ID Lua
master, exercise both existing name-taking attachment paths, and verify detach
and PC-only defaults. The `npc-follow` compiled revert triple restores the old
nil result and fails the actual-leader assertion; logs are in
`step2-npc-follow-triple/`. Full normal and race gates pass again in
`step2-npc-follow-gate-*.txt`. The replay now contains 22 controls.
