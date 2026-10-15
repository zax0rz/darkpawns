# DeepSeek — round 4: the water-arm mob, the mount roll, and carve's fish

Date: 2026-10-15 · Branch: `deepseek/draw-water-arm-mob` (off `origin/main`
`dc9ec4da0`) · Rules: R1, R3a, R3b, R5c, R5e, R5g, R5h

No production change this round: all three assigned classes turned out to be
**state divergences upstream of the draw sites**, so the deliverable is the
identification and the ruled-out list. #2011 (aware-backstab retaliation) and
#2015 (§5) are still open, so `origin/main` still shows the §4 and §5 rows as
DRAW; the pulse-accounting theory is dropped per the brief.

Method: temporary local instrumentation (`DP_ZZWATER`-gated prints in
`roomActivityDamageArms` and the zone-reset `M` branch, `ZZHIT`/`ZZROOM` in
`cmdHit`, `ZZMOUNT` in `damage()`, plus a `DP_SAVE_DRAW_LOGS` copy of both draw
logs) — all reverted before this commit; the tree carries no harness change.

## §2 — the divergent body is Spandaramet (mob 4600)

`shop-live-inventory-control`, first mismatch unchanged:
`stream=62467 c=Dice(1,1) = 1 go=Number(0,20) = 9 c-step=setup go-step=setup`.

Go's water-room census (deduped by room+vnum) gives **exactly two eligible
bodies** — the same two Go damages:

```
ZZW idx=62465 room=4611 sector=7 vnum=4601 name=the naga Ananta  wb=false ww=false fly=false boat=false
ZZW idx=62512 room=2519 sector=7 vnum=2502 name=a grannit         wb=false ww=false fly=false boat=false
```

Every other water body is exempt by `AFF_WATERWALK`/`AFF_FLY`/`AFF_WATERBREATHE`
or by `has_boat` (e.g. `the ahuizotl` ww=true, `a great white shark` wb=true,
`a venomous churka` wb=true). C fires **three** arms, so C has one more eligible
body — and the identity follows from the reset order:

```
lib/world/zon/46.zon
  M 0 4600 3 4611      <- Spandaramet, room 4611 = SECT_WATER_NOSWIM
  E 1 4600 3 17
  M 0 4601 3 4611      <- the naga Ananta, same room
  E 1 4601 3 17
```

`read_mobile` prepends, so the 4600 instances sit **immediately after** the 4601
instances in `character_list` — exactly where C's third arm lands (one
`loud_mobs` draw after Ananta's). Go's log has no Spandaramet instance in any
water room at any point in the run, and Go's reset **does** put it there at boot:

```
ZZSPAWN vnum=4600 cmdroom=4611 actual=4611
```

So the instance has left 4611 before the recorded pass. Ruled out this round:

- **Zone-command parsing**: `M <if_flag> <mob> <max> <room>` is parsed correctly
  (`pkg/parser/zon.go:139-145`: Arg1 = 4600, Arg2 = 3, Arg3 = 4611).
- **`RANDZON` relocation**: mob 4600's act flags are `329832` = bits 3, 5, 6, 11,
  16, 18 (`MOB_ISNPC`, `AGGRESSIVE`, `STAY_ZONE`, `MEMORY`, `NOBASH`, `NOBLIND`)
  — bit 20 (`MOB_RANDZON`) is **not** set, so neither engine relocates it.
- **The movement predicates** (`pkg/game/ai.go:88-160`) match C: `CAN_SWIM` =
  `AFF_WATERWALK|AFF_FLY|PRF_NOHASSLE|has_boat` (`src/utils.h:150-153`),
  `IS_FLYING` = `AFF_FLY|PRF_NOHASSLE` (`:148-149`), plus `STAY_ZONE`,
  `ROOM_DEATH`/`ROOM_NOMOB`, and closed exits — and `roomActivityHasBoat` already
  covers the `has_boat` terms. (Round 3's `has_boat` tightening, which implements
  C's non-wearable rule, moved no row and is not on this branch.)
- **Innate affects**: mob 4600's affected-flag field is `0`, and the parser maps
  the numeric mask through tables whose order matches `src/structs.h` (`WATERWALK`
  6, `FLY` 26, `FLAMING` 31, `WATERBREATHE` 36) — no mis-mapping to find.

**Next step:** the instance moves differently, and with identical door draws the

## §3 — the extra draw is `damage()`'s mount roll, and Go never damages a player

Captured both draw logs for `spec-proc-dragon-breath-combat`. C draws
`Number(0,99)` **twice** (streams 4132 and 4172); Go draws it **once** (4132).
The window tail is identical up to the last entry:

```
4126 Number(0,18)      mobile_activity door
4127 Number(0,15)      mobile_activity sound
4128-4131 Number(0,3)  x4  (the dragon's per-round MOB_SPEC gate, src/spec_procs.c:959)
4132 Number(0,99)      <- BOTH
4133 Dice(1,1)         <- BOTH
4134 Number(0,500) 4135 Number(0,10000) 4136 Number(1,20) 4137 Dice(1,4)   PC round
4138 Number(0,900) 4139 Number(1,20) 4140 Dice(30,4)                       NPC round
4170 Number(0,80)      hit()'s high-level switcheroo scan (src/fight.c:1419-1440)
4171 Dice(1,2)         the swing's damage dice
4172 Number(0,99)      C ONLY
```

Within `damage()` the only `number(0,99)` sites are the mount knockoff and its
landing roll (`src/fight.c:1548`, `:1552`); `raw_kill` and the rest of the body
draw nothing else. Instrumenting Go's twin (`pkg/combat/fight_core.go:469`)
printed **zero** lines for the whole run — Go's `damage()` is never called with a
player victim in this scenario at all. So the divergence is **upstream of the
mount predicate**: C's attack reaches a player victim (and that victim carries
`AFF_MOUNT`, which is what makes the roll draw at all) where Go's does not. The
brief's framing holds for the *site*; the condition to chase is why C has a
player victim there and Go does not — next step is to log both engines'
`damage()` calls (attacker, victim, dam) across the window rather than the
predicate.

## §7 — carve: the fish is not in Go's room when `kill fish` runs

First mismatch unchanged: `stream=4098 c=Number(1,20) = 9 go=Number(0,3) = 1
c-step=warmup go-step=probe step 1: carve`. C's warmup adds 8 entries, Go's 5.

**The wait/drain hypothesis is disproved at the code level.** Go's DP_CLOCK pump
is `GameLoop.PumpPulses` (`cmd/server/main.go:660`) → `gl.heartbeat(pulse)`
(`pkg/engine/gameloop.go:210-213`) → `heartbeatTurn`, which dispatches
`OnDrainInput` **first**, before any other callback (`:305-309`). A queued command
therefore does drain on pumped pulses.

What actually happens: Go's `kill fish` resolves **no target**.

```
ZZHIT target-missing:fish
```

and the room is empty at that moment — `GetMobsInRoom` returned nothing, so the
`ZZROOM` dump printed no lines at all — even though Go's reset *does* put the
fish in the room at boot:

```
ZZSPAWN vnum=4350 cmdroom=8162 actual=8162
```

Mob 4350's keywords are `fish carve_meat` (`lib/world/mob/43.mob:725-726`) and
the fixture spawns two in 8162 (`spawn-mob 4350 2 8162 80`), so the resolution
input is right: the fish has **left 8162** in Go by the time the warmup runs,
while C's is still there. That is the same *mob-position* class as §2, and the
two should be chased together — the common question is why a wandering mobile
that both engines spawned in the same room is somewhere else in Go a few pulses
later, with identical door draws.

## §4 — not this branch

Per the brief, §4 was the aware-backstab retaliation (#2011), not DP_CLOCK pulse
accounting; the pulse-accounting theory is dropped for §5 (already fixed by
#2015 as the `set` save-path bad-stats arm) and for §7 (disproved above at the
pump call path).

difference must be *when* it moves (how many `PULSE_MOBILE` ticks it sees between
the reset and the recorded pass) or a gate on a room it passes through. Compare
the two engines' positions for mob 4600 over the first few passes, not just at
the recorded one.
