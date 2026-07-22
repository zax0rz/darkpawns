# Report: Per-Zone Draw Bisection — Progress (INCOMPLETE)

**Status:** Investigation in progress. Core measurements taken, root cause localized to M commands, but the exact per-mob draw-site divergence not yet pinned. Instrumentation reverted, both engines build clean.

## What Was Measured

### 1. Per-zone draw deltas (C vs Go, matched by zone name)

Both engines are now deterministic (C seam PR #397, Go map-order fix PR #398). Total gap: **C=55685, Go=38332, delta=17353** during zone resets.

Top offenders (C draws more than Go):

| Zone | C draws | Go draws | Delta |
|---|---|---|---|
| The Great Plains | 6405 | 2566 | +3839 |
| The Great Forest | 4984 | 2630 | +2354 |
| Alaozar, the Holy City | 3819 | 1483 | +2336 |
| The Swamp | 1928 | 356 | +1572 |
| Kir Drax'in | 2364 | 1256 | +1108 |
| The Grey Keep | 2590 | 1628 | +962 |
| Sulfur Fur Mountains | 849 | 235 | +614 |
| Orc Burrows | 1487 | 884 | +603 |
| Noirwood Forest | 1126 | 625 | +501 |
| Kir-Oshi Main | 1262 | 791 | +471 |

**C consistently draws more per zone than Go.** The gap is proportional to the number of M commands in each zone.

### 2. Per-command-type aggregation

| Cmd | C draws | C count | Go draws | Go count | Diff | C/cmd | Go/cmd |
|---|---|---|---|---|---|---|---|
| **M** | **53663** | **2723** | **36404** | **2723** | **+17259** | **19.7** | **13.4** |
| O | 269 | 273 | 254 | 276 | +15 | 1.0 | 0.9 |
| G | 433 | 437 | 335 | 367 | +98 | 1.0 | 0.9 |
| E | 1255 | 1257 | 1119 | 1155 | +136 | 1.0 | 1.0 |
| P | 65 | 66 | 55 | 60 | +10 | 1.0 | 0.9 |
| R | 0 | 29 | 0 | 29 | +0 | 0.0 | 0.0 |

**M commands are the culprit.** C draws **19.7 per M**, Go draws **13.4 per M** — a **6.3 draw per mob** difference. With 2723 M commands, that accounts for ~17,155 of the 17,353 gap (99%). All other command types contribute <200 draws combined.

### 3. What M commands draw

Both C (`read_mobile` db.c:1730-1782) and Go (`NewMob` mob.go:95-174) draw for:
- **HP dice**: `dice(hit, mana)` consuming `hit` draws
- **Gold variance**: `number(0,1)` + `number(1,20)` = 2 draws (if gold > 0)

Go's `NewMob` (mob.go:106-113) has a comment confirming PR #396 removed the spawn-time stat boost double-draw. Stat boosts now happen only at parse time in both engines.

**The 6.3 extra draws per mob in C are NOT from:**
- Stat boosts (both do parse-time only, confirmed)
- Random room placement (only 26 zone79 mobs, 0 RANDZON mobs — negligible)
- Gold variance (identical 2-draw logic in both)

## What Remains (the gap I couldn't close)

The 6.3 draws/mob difference is unexplained. Possibilities I was investigating when time ran out:

1. **Parse-time vs spawn-time stat boost interaction**: C's `parse_simple_mob` (db.c:1053-1066) draws 6 stat boosts at parse time during `boot_world()`. Go's parser (`parser/mob.go:361-378`) also draws 6 at parse time. But Go's `NewMob` does NOT re-draw them (PR #396 fix). So both should draw the same per M. Yet C draws 6.3 more. **Something else is drawing in C's `read_mobile` that Go's `NewMob` doesn't replicate.**

2. **C's `max_hit` branching**: C has `if (!mob->points.max_hit) { dice(hit, mana) } else { number(hit, mana) }`. The `max_hit=0` branch (set at parse time, db.c:1072) always fires, consuming `hit` draws. Go has `if proto.HP.Num > 0 && proto.HP.Sides > 0 { Dice(Num, Sides) }`. If some mobs have `HP.Num=0` or `HP.Sides=0` in Go but `hit>0` in C, Go would skip the draw (defaulting to 100) while C would still draw. **This needs a mob-file comparison to verify.**

3. **C draws something else in `read_mobile` I missed**: I compared the C and Go code line-by-line and couldn't find it. But 6.3 draws/mob × 2723 mobs = 17,255 draws — exactly the gap. There must be a systematic per-mob draw difference.

## Recommended Next Steps

1. **Compare mob HP field values**: Check if any mobs in Go's `lib/world/mob/` have `HP.Num=0` or `HP.Sides=0` while C's equivalent `points.hit` is non-zero. This would explain Go skipping HP dice draws.

2. **Per-mob draw instrumentation**: Instrument a single zone (e.g., Great Plains) with per-M-command draw logging on both engines. Compare draw-by-draw to find exactly where C consumes extra draws.

3. **Check if C's `read_mobile` has hidden draws**: The C code at db.c:1753-1754 has `dice(mob->points.hit, mob->points.mana) + mob->points.move` — the `+ mob->points.move` is just addition, not a draw. But verify there's no other `number()`/`dice()` call in the function I missed.

## Deliverable Status

| Deliverable | Status |
|---|---|
| 1. Per-zone draw-delta table | Done — see above |
| 2. Divergent command type + call-sequence diff | Partial — M commands identified, exact draw site not pinned |
| 3. Verdict on lead (i)/(ii) | Lead (i) confirmed: zone-reset mob-spawn path is the culprit. PR #396 removed double-draw correctly, but C still draws more per mob for an unidentified reason. |
| 4. Minimal faithful fix | Not yet — need to pin the exact draw site first |
| 5. Instrumentation reverted, both build clean | Done |

## Files Touched (all reverted)

- C: `random.c`, `random.h`, `class.c`, `db.c` — draw counter + per-zone/per-command logging
- Go: `pkg/dprng/cmwc.go`, `pkg/game/spawner.go`, `pkg/game/world_zone.go`, `cmd/dp-oracle-diff/main.go` — draw counter + per-zone/per-command logging
