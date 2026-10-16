# DeepSeek — round 7: the mob wander skipped `do_simple_move`'s gates

Date: 2026-10-16 · Branch: `deepseek/round7` (off `origin/main` `a2f6de0dc`) ·
Rules: R1, R3a, R5c, R5e, R5g, R5h

This round found the root cause of the new shopkeeper DRAW rows the brief
listed under §2, fixed that root cause's class, and ruled several of the
brief's hypotheses out with measurements. The two remaining rows are a
*different* root cause and are handed off below with their fingerprints.

## Root cause: the mobile wander never ran C's `perform_move` gate list (fixed)

`mobileActivityForMob` checks mobact's own list (src/mobact.c:121-138) and then
moved the mob itself with `mob.SetRoom`. C does not: mobact calls
`perform_move(ch, door, 1)` (src/mobact.c:139) → `perform_move`
(src/act.movement.c:311-351) → `do_simple_move` (src/act.movement.c:95-305),
whose gate list is a **second, different list**:

```
src/act.movement.c:119-125  charmed body in its master's room does not leave
src/act.movement.c:126-132  "if this room or the one we're going to needs a
                            boat, check for one" — has_boat() at :58-78
src/act.movement.c:133-176  IS_MOUNTED (AFF_MOUNT): no rider → unmount, no move;
                            not standing / short of move points → no move
src/act.movement.c:172-176  movement-point refusal (PC-only, so a no-op for mobs)
src/act.movement.c:177-182  ROOM_TUNNEL with a player inside refuses entry
src/act.movement.c:184-189  a mount cannot ride indoors
src/act.movement.c:208-211  move points are deducted for non-NPC non-immortals only
```

The two lists also refuse *differently*: mobact's sector gates `continue` (the
rest of that mob's pulse is skipped), while `do_simple_move`'s gates only refuse
the step (the pulse keeps running). The port had ported only the first list, so
a boatless mob standing in `SECT_WATER_NOSWIM` walked out of the water in the
port while C keeps it there forever — and C keeps damaging it. Impact is not
cosmetic: those mobs are water-trapped in C and are not in the port.

**Proof (oracle + unit).** `shop-buy-count-capacity`, whose divergence the
brief recorded at §2:

```
before  draw-parity  MISMATCH  stream=62467  c=Dice(1,1) = 1  go=Number(0,20) = 9  c-step=setup go-step=setup
after   draw-parity  MATCH     entries=35604 stream=67861      result: no normalized divergence
```

The oracle-side state difference, measured with a throwaway scenario that
`goto`s each labyrinth/swamp room and `look`s (`cmd/dp-oracle-diff` fixture
`empty-players` + `quiet-zone 80`, same setup as the failing scenario):

```
--- [goto 4611] c-oracle        +++ [goto 4611] go-port
 Lost In The Swamp of Sadness    Lost In The Swamp of Sadness
    You are travelling...           You are travelling...
 A large naga slithers here.     A large naga slithers here.
-Spandaramet glides through the swamp in search of lost souls.

--- [goto 4621] c-oracle        +++ [goto 4621] go-port
 Lost In The Swamp of Sadness    Lost In The Swamp of Sadness
    Black sludge slurps...          Black sludge slurps...
                                  +Spandaramet glides through the swamp...
```

Spandaramet (mob 4600) loads into room 4611 (lib/world/zon/46.zon:4,
`M 0 4600 3 4611`; 4611's sector is 7 = `SECT_WATER_NOSWIM`, 46.wld) and is
`AGGRESSIVE|STAY_ZONE|MEMORY|NOBASH|HUNTER` with no swim affects, so C's
`do_simple_move` boat gate refuses every step out (room 4621 is a FIELD room,
so mobact's destination-sector gates never even applied). The port walked it
south on the shared door draw, which is why the oracle's room_activity took 25
drowning damage for it every pulse in C (`Dice(1,1)` at stream 62467) while the
port's room_activity had no damage there and drew the next mob's `loud_mobs`
`Number(0,20)` instead.

`TestWanderMobNeedsBoatInNoswimRoom` is the R5h control: with the boat gate
deleted the test fails (`boatless mob left the WATER_NOSWIM room for 20`) and
passes with it. `TestWanderMobMovesFromDryRoom` keeps the gate a water gate
rather than a blanket refusal.

## Ruled out, with the measurement that ruled it out

- **§1 / §7 — "a mob in the character list but missing from `activeMobs`."**
  Not possible by construction: `cOrderedCharacters` (`pkg/game/world_order.go`)
  iterates `w.players` + `w.activeMobs`, so Go has no second character list to
  fall out of — a mob cannot be in one and not the other, and `spawnMob`
  (`pkg/game/world.go:954`) is the only production registration path (the other
  `NewMob` callers are `ReadDisposableMob`'s throwaway instance and a `stat mob`
  preview). A temporary per-mob trace of the whole pass measured the pass rather
  than guessing: 2,603 mobs, 0 skipped by the `!IsAlive()` gate, 279 `loudMobs`
  draws and 2 damage-arm draws. The round-6 handoff's `DP_ZZRES` trace was never
  committed and exists in no branch; nothing from it was reused unverified.
- **§1 — "the fixture's spawn / zone-update reset / summon-style spawners."**
  The pass's membership and order match C's; the divergence was a *room*, not a
  list entry.
- **§2 — "batch-o routes shopkeepers through a `shop_keeper` special on the
  pulse that C doesn't."** Ruled out for these rows: `specShopKeeper`
  (`pkg/game/spec_procs3.go:86`) returns false for every `cmd != "steal"`, and
  C's `SPECIAL(shop_keeper)` returns FALSE for `keeper == ch` before it draws
  anything (src/shop.c:942-1005), so neither side draws on the pulse. The
  divergence at stream 62467 was the water-room state above, and no shopkeeper
  is within a hundred events of it.
- **§2 — "the harness's quiet-zone fixture."** `applyQuietZoneFixtures`
  comments `M`/`G`/`E` lines out of a disposable `zon/80.zon` for *both*
  engines (`cmd/dp-oracle-diff/main.go:1642-1685`); zone 46 was untouched, and
  the world files are byte-identical between `darkpawns-c-oracle/lib` and the
  port's `lib`.

## Frontier: the two rows this fix does not clear (a different root cause)

Both remaining rows involve a `spawn-mob` fixture (a shopkeeper the fixture
appends as the zone's last reset command, `applyMobFixtures`,
`cmd/dp-oracle-diff/main.go:1282-1308`), and both are a one-event offset in the
pulse's draw phases rather than a damage row:

```
spell-charm-shopkeeper-depth   stream=60090  c=Number(0,18)  = 11  go=Number(0,20)  = 12
shop-buy-weight-sell-steal     stream=62381  c=Number(0,900) = 230 go=Number(0,20)  = 5
```

Read with the pulse layout (mobile_activity → room_activity → perform_violence),
`c=Number(0,18)` is a *door* draw where the port is already in room_activity's
`loud_mobs`, and `c=Number(0,900)` is a *combat* attacks-count draw where the
port is still in `loud_mobs`: in the first row the port's room_activity phase
starts one event early, in the second it ends one event late. That is the
signature of a pass iterating one mob more or fewer than C — a different bug
from this round's (a room, not a phase). The `spawn-mob` append is symmetric in
the harness, so the lead is what the port does with that fixture-spawned mob
*after* it loads, not the fixture itself.

Method that produced this round's evidence, reusable for that one: build the
harness with `DP_KEEP_DRAWS=<dir>` retaining `c-draws.log`/`go-draws.log`, a
`runtime.Caller` frame in the port's draw log (`pkg/dprng/drawlog.go`) to name
the port's draw site, and a temporary `roomTrace` in `RoomActivity` naming the
mob, room, HP, position, noise and affects at every event, plus a `look`-scan
scenario for the oracle's room contents. All three are dev-only and were
reverted before the commit (the `main.go` and `drawlog.go` diffs are gone; the
scenario file was not kept).

## Census result for this change (targeted, 17 scenarios, seed 1, `--draw-parity`)

```
/home/zach/Archives/darkpawns/oracle-runs/2026-10-16/deepseek-r7-wander-boat-gate/
oracle-regression: scenarios=17 passed=14 expected=0 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0 unstable=0 … verdict=CLEAN
oracle-draw-parity: scenarios=17 draw_divergent=3
draw-parity MISMATCH scenario=carve-depth                  seed=1 stream=4098  c=Number(1,20) = 9   go=Number(0,3) = 1   c-step=warmup  go-step=probe step 1: carve
draw-parity MISMATCH scenario=shop-buy-weight-sell-steal   seed=1 stream=62381 c=Number(0,900) = 230 go=Number(0,20) = 5
draw-parity MISMATCH scenario=spell-charm-shopkeeper-depth seed=1 stream=60090 c=Number(0,18) = 11   go=Number(0,20) = 12
```

`shop-buy-count-capacity` is no longer a DRAW row. The remaining three are the
frontier above; `carve-depth`'s row is §7's: the oracle spends a `Number(1,20)`
to-hit in the warmup (`hit fish`) that the port never spends, i.e. the port's
attack still resolves no victim there while the mob is present in the room —
one step *later* in the chain than the registry the brief suspected.

## §1's carve-depth seed sweep (direct runs, seeds 1, 2, 3, 5, 8)

```
seed 1  draw-parity MISMATCH stream=4098 c=Number(1,20) = 9  go=Number(0,3) = 1  c-step=warmup go-step=probe step 1: carve
seed 2  draw-parity MISMATCH stream=4101 c=Number(1,20) = 9  go=Number(0,3) = 1  c-step=warmup go-step=probe step 1: carve
seed 3  draw-parity MISMATCH stream=4098 c=Number(1,20) = 14 go=Number(0,3) = 2  c-step=warmup go-step=probe step 1: carve
seed 5  draw-parity MISMATCH stream=4099 c=Number(1,20) = 14 go=Number(0,3) = 2  c-step=warmup go-step=probe step 1: carve
seed 8  draw-parity MISMATCH stream=4097 c=Number(1,20) = 6  go=Number(0,3) = 1  c-step=warmup go-step=probe step 1: carve
```

The row does **not** clear, and it is seed-independent: every seed splits at the
same place, with the oracle spending a `Number(1,20)` to-hit in the warmup that
the port never spends. The warmup runs `kill fish` then `hit fish` in room 8162
(`cmd/dp-oracle-diff/scenarios/carve-depth.txt`), so this is still §7's
"present, standing, resolves nothing" — but the registry is not the mechanism
(the port's character list is `players + activeMobs` and the mob is in it), so
the next round should look at the *resolver* (the name match and the visibility
call it makes) rather than at `activeMobs`. §3's DRAW rows were not re-measured
beyond the census above.

## Not taken this round

- The round-6 room-light counter gap (still open; real, per round 6, but not
  this round's cause).
- `do_simple_move`'s very first step, `special(ch, dir + 1, "")`
  (src/act.movement.c:113-115, `need_specials_check = 1` from
  src/mobact.c:139), is still **not** ported for the mobile wander: it calls the
  room's mobile/object/room special procedures with a command number, and the
  port's player-side equivalent (`movementSpecialBlocks`,
  pkg/game/act_movement.go:457-475) passes a direction string instead, so it
  cannot be reused as-is. Flagged with a `TODO(port)` at the site (R5d). No
  scenario has exposed it yet; it is the next gate in this call path to port.
- The room_activity damage arms and `loud_mobs` order for *players* were
  re-read but not changed.
