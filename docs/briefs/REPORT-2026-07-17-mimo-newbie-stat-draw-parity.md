# Report: Newbie Stat-Roll Draw Parity (C↔Go)

## 1. Root Cause — Pinned

**The stat-roll PRNG stream diverges because C's boot-time zone resets consume ~8904 more draws than Go's, putting the two engines at different stream positions when `roll_real_abils` fires.**

### Measured draw counts (DP_SEED=1, DP_CLOCK=1):

| Checkpoint | C draws | Go draws | Delta |
|---|---|---|---|
| After PRNG seed | 1 | 1 | 0 |
| After boot_world (parse) | 3709 | 3709 | 0 |
| After zone resets | **59394** | **50490** | **+8904** |
| At roll_real_abils entry | 59394 | 50492 | +8902 |

Both engines start zone resets at the same stream position (3709). The divergence is entirely within the zone reset phase.

### Three contributing factors (ordered by impact):

#### A. Zone file mismatch (PRIMARY — ~8500 draws)

The C oracle has **2 extra zone files** not present in Go's `lib/world/zon/`:
- `150.zon`
- `165.zon`

This results in more total zone reset commands:

| Command | C count | Go count | Extra in C |
|---|---|---|---|
| M (load mobile) | 3418 | 3295 | +123 |
| O (load object to room) | 356 | 349 | +7 |
| G (give obj to mob) | 535 | 526 | +9 |
| E (equip obj on mob) | 1398 | 1358 | +40 |
| P (obj in obj) | 115 | 115 | 0 |
| **Total** | **5822** | **5643** | **+179** |

Each M command consumes draws for HP dice (`dice(hit, mana)` = `hit` draws) + gold variance (2 draws). Each O/G/E/P command consumes draws for `percent_load` (1 draw) + `init_rare` (0-N draws for rare items). At ~50 draws per command on average, 179 extra commands ≈ **~8900 draws** — accounting for nearly the entire gap.

#### B. Go's NewMob double-draws stat boosts (~6 draws per high-level mob)

In C, level-based stat boosts for mobs > level 15 happen at **parse time** during `boot_world()` (`db.c:1058-1067`), consuming draws from the stream before zone resets begin. These draws are already included in the 3709 "after boot_world" count.

In Go, stat boosts happen at **both** parse time (`parser/mob.go:361-378`) AND spawn time (`game/mob.go:115-129`). The spawn-time boosts consume 6 extra `dprng.Number(0, statmod)` draws per high-level mob instance during zone resets.

This means Go actually draws **more** per high-level mob spawn than C, but the zone file mismatch (factor A) overwhelms this effect.

#### C. Random room selection loop (MINOR — ~200 draws)

For zone79 mobs (vnums 7900-7998), C uses a retry loop:
```c
do { to_room = number(0, top_of_world); } while(invalid);
```
This draws repeatedly until a valid room is found — potentially many iterations.

Go's `pickRandomRoom()` draws at most 5 times, then falls back to a zero-draw linear scan. With only 26 zone79 mobs, this accounts for ~200 extra draws in C — minor compared to factor A.

### Why it's character-dependent

The divergence is character-dependent because the 8904-draw offset shifts the entire PRNG stream. Whether a particular character name ends up with aligned or misaligned stats depends on whether the shifted stream happens to produce the same values at the new position. The `combat-death` scenario (char "Cfighter") aligns by coincidence; `hunger-thirst` (char "Oracletst") does not.

### The masking

The `character-creation` oracle scenario normalizes `<ROLLED_STATS>` output, so it never detects this divergence. The `hunger-thirst` and `guild-practice` scenarios expose it because they check stat-derived values (max HP from CON, practices from WIS).

## 2. Which Side Is Wrong

**Go is wrong.** The C oracle defines correct behavior. Go's `lib/world/zon/` directory is missing zones 150 and 165 that exist in the C oracle's `lib/world/zon/`. This causes Go's zone resets to process fewer commands, consuming fewer PRNG draws, and desyncing the stream before character creation.

### C source that defines correct behavior:
- `db.c:385-392`: `reset_zone(i)` loop iterates `0..top_of_zone_table`
- `db.c:2079-2278`: `reset_zone()` processes all M/O/G/E/P/R/D commands
- `db.c:1730-1782`: `read_mobile()` rolls HP dice + gold variance per mob instance
- `db.c:1933-1964`: `read_object()` calls `init_rare()` for ITEM_RARE objects

## 3. Proposed Fix

### Fix 1 (PRIMARY): Sync zone files

Copy `150.zon` and `165.zon` from the C oracle's `lib/world/zon/` to Go's `lib/world/zon/`. This is the mechanical fix — it aligns the zone reset command count and eliminates the ~8900-draw gap.

```bash
cp ~/.openclaw/workspace/darkpawns-c-oracle/lib/world/zon/150.zon \
   ~/.openclaw/workspace/darkpawns_repo/lib/world/zon/
cp ~/.openclaw/workspace/darkpawns-c-oracle/lib/world/zon/165.zon \
   ~/.openclaw/workspace/darkpawns_repo/lib/world/zon/
```

If Go also needs corresponding `.mob` and `.obj` files for those zones, copy those too:
```bash
# Check if these exist and copy as needed
cp ~/.openclaw/workspace/darkpawns-c-oracle/lib/world/mob/15*.mob \
   ~/.openclaw/workspace/darkpawns_repo/lib/world/mob/ 2>/dev/null
cp ~/.openclaw/workspace/darkpawns-c-oracle/lib/world/obj/15*.obj \
   ~/.openclaw/workspace/darkpawns_repo/lib/world/obj/ 2>/dev/null
```

### Fix 2 (SECONDARY): Remove double stat boosts in NewMob

`game/mob.go:115-129` applies stat boosts at spawn time, but `parser/mob.go:361-378` already applies them at parse time. The spawn-time boosts should be removed — they're a redundant draw that will cause divergence once Fix 1 is applied (since the mob prototypes already carry boosted stats).

```go
// game/mob.go — REMOVE lines 114-129 (the stat boost block)
// The prototype already has boosted stats from parser/mob.go:361-378
```

### Fix 3 (TERTIARY): Align random room selection with C's retry loop

`spawner.go:155-179` `pickRandomRoom()` should match C's do-while semantics: draw `number(0, total_rooms-1)` and check conditions, retrying until valid. The current 5-attempt cap + linear fallback consumes fewer draws than C's unbounded loop. Same for `pickRandomZoneRoom()`.

## 4. Blast Radius

The stat-roll divergence is **systemic** to every scenario that exposes stat-derived values:
- `hunger-thirst`: max HP (from CON) and practices (from WIS) diverge
- `guild-practice`: practices diverge
- `character-creation`: masked by `<ROLLED_STATS>` normalization
- `combat-death`: coincidentally green for "Cfighter" — but fragile

After Fix 1 (zone file sync), the draw counts should align at the stat roll entry point. The currently-green scenarios (`combat-death`, `look-start-room`, etc.) should remain green because the draw alignment means both engines see the same PRNG values at every point.

**Verify after fix:** run all oracle scenarios:
```bash
for s in hunger-thirst guild-practice character-creation combat-death observation; do
  DP_ORACLE_BIN="$HOME/.openclaw/workspace/darkpawns-c-oracle/bin/circle" \
    go run ./cmd/dp-oracle-diff --scenario "$s" 2>&1 | grep "result:"
done
```

## 5. Instrumentation Confirmation

All temporary draw-count instrumentation has been reverted from both engines:
- C: `random.c`, `random.h`, `class.c`, `comm.c`, `db.c` — all reverted via `git checkout`
- Go: `pkg/dprng/cmwc.go`, `pkg/game/character.go`, `cmd/server/main.go`, `cmd/dp-oracle-diff/main.go` — all reverted via `git checkout`
- Verified: `grep -r "dp_draw_count\|DP_DRAW\|dpDrawLog\|DrawCount"` returns empty across both codebases
- C oracle rebuilds clean: `make` succeeds with no errors
- Go builds clean: `go build ./...` succeeds, `go test ./pkg/dprng/... ./pkg/game/...` passes

---

## SECONDARY: Observation Co-Location Bug

### Root Cause

The `observation.txt` scenario has asymmetric setup commands for the target peer:

**C oracle** (`[setup:oracle:target]`, lines 41-55):
```
...
1           ← enter game
look        ← triggers room description
recall      ← moves to MortalStartRoom (8004)
```

**Go port** (`[setup:port:target]`, lines 57-71):
```
...
1           ← enter game
            ← NO look, NO recall
```

In C, both players recall to room 8004 (Temple Altar) during setup, so they're co-located for the probe. In Go, only the primary recalls (via `[warmup] recall`), while the target stays in the newbie hometown room (8162). They're in different rooms — `diagnose Obstarget` returns "No-one by that name here."

### Proposed Fix

Add `look` and `recall` to the Go port's target setup in `observation.txt`:

```diff
 [setup:port:target]
 Obstarget
 y
 oraclepass
 oraclepass
 Y
 N
 F
 H
 W
 K
 Y
 <ENTER>
 1
+look
+recall
```

This is a scenario-file fix only — no Go code changes needed.
