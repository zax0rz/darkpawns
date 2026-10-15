# DeepSeek — round 5: the wander trace, and what it actually shows

Date: 2026-10-15 · Branch: `deepseek/draw-wander-class` (off `origin/main`
`f8b99284f`, which carries #2011) · Rules: R1, R3a, R3b, R5c, R5e, R5g, R5h

No production change: the trace redirected the diagnosis rather than confirming
the wander-gate theory, and it **corrects round 4** on one point. All
instrumentation was temporary and is reverted; the tree carries no harness
change.

Method (the brief's ask): a `DP_ZZWANDER`-gated trace inside
`pkg/game/ai.go`'s `wanderMobWithDoor` that logs, per pass for mobs 4600 and
4350, the draw index, room, position, drawn door, every gate verdict
(`door-range`, `no-room-snapshot`, `no-exit`, `closed`, `stay-zone`,
`death-nomob`, `water-noswim`, `flying`) and the move result — plus a
`ZZRES` trace in `pkg/game/target.go`'s resolver.

## §7 carve — round 4's conclusion was wrong: the fish never leaves 8162

```
ZZW vnum=4350 idx=4061 room=8162 pos=8 door=2 start
ZZW vnum=4350 gate=no-exit dir=south REFUSE
ZZW vnum=4350 idx=4091 room=8162 pos=8 door=17 start
ZZW vnum=4350 gate=door-range REFUSE
```

The fish is in 8162, POS_STANDING (its mob-file position is `8`, and Go's
`NewMob` reads `proto.Position`), and it **never moves**: its only two wander
attempts drew doors 2 (south) and 17, and both were refused — 2 because Go's
snapshot of 8162 has no south exit, 17 because it is outside `NUM_OF_DIRS`.
So the "fish has left 8162" line in round 4's handoff is wrong, and so is the
inference built on it: `kill fish` fails for a **target-resolution** reason, with
the mob present in the room, standing, keywords `fish carve_meat`. The resolver's
two gates (`canSee`, `isnameWithAbbrevs`) are the place to look — the `LIGHT_OK`
visibility change landed recently, and `GetMobsInRoom` filters `w.activeMobs` by
`GetRoom()`, a different source than the character list `MobileActivity` walks.

## §2 Spandaramet — Go moves it on a door C's predicate also allows

```
ZZW vnum=4600 idx=61393 room=4611 pos=8 door=2 start
ZZW vnum=4600 MOVE room=4611 -> 4621 dir=south sector=2
```

Room 4611's south exit is open and leads to 4621
(`lib/world/wld/46.wld`: `D2 / ~ / ~ / 0 0 4621` — state 0), 4621 is sector 2
with no flags, mob 4600 has no `STAY_ZONE`, and both engines' positions are 8.
So **C's predicate permits that move too** — the gate is not the divergence.

What is left is *which instance drew that door*. The per-mob draws (wander door,
sound gate, scavenger, aggression) are consumed in `character_list` order, so two
engines with the same stream but a different **list order** assign the same values
to different mobs: identical draw logs, different moves, different water arms.
That also explains why this class hid behind "same door draws, different room".

Go's ordering machinery is `cOrderedCharacters` (`pkg/game/world_order.go:19-49`):
descending `characterSequence` — newest-first, which is C's `read_mobile`
prepend — with `GetID()`/`GetVNum()` tie-breaks. `nextCharacterSequence`
increments once per spawn (`pkg/game/world.go:950-951`), so there are no ties and
the order is exactly the *reverse spawn order*. For it to differ from C's, the
**spawn order** must differ. The concrete suspect, unverified: Go's zone resets
walk `GetAllZones()`, which **sorts by zone number**
(`pkg/game/world_zone.go:191-199`), while C walks `zone_table` in the order
`boot_db` read the files. If those orders differ, every reset-spawned mobile's
list position shifts, which is precisely the observed class.

**Next step (cheap, decisive):** print both engines' spawn sequence for the mobs
in one zone (Go's `nextCharacterSequence` order vs the zone file's `M` order) and
compare against `zone_table`'s file order. If the zone order is the difference,
the fix is in the boot/zone-iteration order, not in the wander predicate — and
§3's "C's attack reaches a player victim where Go's does not" may well be the same
shifted-assignment effect, since `damage()`'s target selection walks room lists.
