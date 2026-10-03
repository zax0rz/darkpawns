# DP-1371: shared mobile equipment train

Stop tier: shared game/combat/stat state, deferred ownership, descriptor output,
and a symmetric oracle fixture extension. Base origin/main 42f878b49 (#1764).
Zach approved the shared repair proposed in that PR's handoff. R1/R3/R5b/R5c/R5h.

## Convention and boundary

Mobile Equipment and LocEquippedMob.Slot use C WEAR_* (structs.h:390-412):
wield 16, head 6, body 5, legs 7, light 0. Player EquipmentSlot is unchanged.
Every live attach/removal uses equipMobileLocked/unequipMobileLocked through
World ownership: reset E, Lua, generic movement, spec procedures, object
extraction, mobile extraction, corpse and dust. Standalone test mobiles use
the same core without a World. Occupied-slot objects remain floating; rejected
alignment objects return to the front of inventory (handler.c:690-727).

handler.c:609-795 supplies weighted AC, room light, TAKE_NAME restoration,
all equipment modifiers/affect bits, affect_total and zero-stat priority.
NPC class checks always permit (class.c:737-764). Incremental points keep C
signed byte/short assignment widths (structs.h:885-935); attributes recompute
with signed-byte assignment before the existing NPC bounds. Later effective
point overwrites are preserved and removal subtracts the equipment modifier.
The modifier itself is a signed byte; C -128 negation remains -128, including
affect_total's strip/reapply point effects. The width test preserves that quirk.
The separate damage-dice representation does not double-count prototype
Damage.Plus when a damroll override is installed. Saves feed the existing
spell resolver and stat renderer. Equipment RaceHates preserve wear ordering.

Light follows equipped objects, including negative fuel; floor objects do
not change room light (handler.c:897-913). Movement follows C's one-light
char_from_room/char_to_room adjustment (handler.c:503-559,823-839), even if
several worn lights incremented the counter at equip time. Room updates
publish copies. Alignment room acts and bad-stat messages keep C text and
priority. New actor messages select the concrete switched NPC body, never a
same-name NPC, after game locks release. Ordinary NPCs have no descriptor.
Zero CHA selects the newest pestilence mobile and retains the concrete NPC
hunting target; zero DEX uses existing native self-damage.

## Readers and retirement

Migrated combat weapon lookup, HitModifiers, fighter parry/headbutt, paladin
charge/disarm, player disarm, backstabber and brain-eater, equipment looks,
boat/steal/loot queries, armor/light/stat and saving-throw consumers. Live
weapon values win over prototypes. Reader assertion controls and before-main
failures are retained under dp-1371-mobile-equipment-proofs.

C queues extraction while reset/Lua can still retain the character pointer
(handler.c:1100-1260; fight.c:395-454). A pending mobile work queue retains
ownership until the existing drain; ordinary active-world lookup retirement
is unchanged. Object-owner lookup and retained Lua references can resolve
that pending body, and prototype caps still count it. Death captures/removes
current possessions through unequip before creating corpse/dust. A later E
in that same reset pass can attach to the pending owner and is cleaned by
the drain. This queue is extraction work, not a replacement actor registry.

## Oracle vehicle

combat-zone-equipped-mobile loads an actual M/E armed checker in zone 80.
The new equip-object fixture writes E 1 <object> <max> <C-slot> to both copied
worlds after validating the last M owner. It never edits src/, the shared
oracle or authored world files. A live player receives the normal checker
special's attack and the subsequent combat round at DP_CLOCK pulse 40;
weapon slash bytes prove C slot 16 reaches the actual combat path. No repins,
dropped seeds or expected-divergence changes.

Exploratory vehicles exposed retained pre-existing frontiers: aggressive
non-special initial combat ordering, the normal checker's sleeping-victim
warning, and NPC switched-command identity. They are not equipment proofs
and are not repaired here. The aggregate reset row remains blocked until
its whole matrix is audited; narrow new rows do not declare that aggregate
complete.

## Lock acquisitions

* Equip/unequip, object movement and extraction: World.mu -> Mob.mu; object
  access uses the existing object accessors. No session/lifecycle lock.
* Room movement: World.mu -> Mob.mu -> room copy publication under World.mu.
* Death: World.mu -> Mob.mu; pending work recorded under World.mu. Drain:
  World.mu -> Mob.mu. Existing session drain takes playerLifecycleMu only
  when there is pending work, then enters World; no new lifecycle acquisition.
* Reset serialization remains World.resetMu outside the existing reset path;
  Spawner.mu is not held across ExecuteZoneReset or deferred effects.
  Admin ResetZone uses that same existing serialized reset entry.
* Effects release World/Mob before room Act, hunting, self-damage or output.
  New concrete NPC output takes Manager.mu.RLock for descriptor selection,
  releases it, then uses guarded send. No playerLifecycleMu acquisition.
* Reader snapshots/accessors take Mob.mu.RLock, release it before callbacks.
  Concurrent equip/remove and stat/light/flag snapshots run under -race.

Final normal gates, oracle revert triple, multiseed vehicle and combined census
results are recorded in the PR and retained evidence manifest.
