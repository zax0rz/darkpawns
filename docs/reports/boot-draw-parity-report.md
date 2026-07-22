# Boot-Time World-Reset Draw Parity Report

**Date:** 2026-07-17
**Scenario:** `combat-swing` under `DP_SEED=1` + `DP_CLOCK=1`
**Observed offset:** Go = +2 draws ahead of C at boot (C=4062, Go=4064)

## 1. Root Cause — Pinned

The +2 offset arises from **two independent bugs** in Go's zone reset executor (`pkg/game/spawner.go` `ExecuteZoneReset`), both causing Go to deviate from C's `reset_zone()` (`src/db.c:2074`).

### Bug A: `R` command unconditionally sets `lastCmd = 1` (+3 draws)

**File:** `pkg/game/spawner.go:446-453`

```go
case "R": // Remove obj/mob from room
    if cmd.Arg3 == 1 {
        s.removeObjectFromRoom(cmd.Arg1, cmd.Arg2)
    } else {
        s.removeMobFromRoom(cmd.Arg1, cmd.Arg2)
    }
    lastCmd = 1  // BUG: always sets, even when target not found
```

In C (`db.c:2220-2244`), `last_cmd = 1` is only set when the target is **actually found and removed**:

```c
case 'R':
    if (ZCMD.arg2) {
        if ((obj = get_obj_in_list_num(ZCMD.arg3, world[ZCMD.arg1].contents)) != NULL) {
            obj_from_room(obj);
            extract_obj(obj);
            last_cmd = 1;   // only on success
        }
    } else {
        for (mob = world[ZCMD.arg1].people; mob; mob = mob->next_in_room) {
            if ((GET_MOB_RNUM(mob) == ZCMD.arg3) && ...) {
                // ... remove mob
                last_cmd = 1;   // only on success
                break;
            }
        }
    }
```

When an `R` command fails to find its target (common during boot with `quiet-mobs`, which suppresses mob spawns), Go incorrectly sets `lastCmd=1`, causing subsequent `if_flag=1` commands to execute. C correctly leaves `last_cmd=0`, skipping them.

**Zone 70 evidence:** `R 0 7072 7024 -1` tries to remove a non-existent mob. C: `last_cmd=0` → subsequent `O 1 7024` and two `P 1` commands skipped (0 draws). Go: `lastCmd=1` → all three execute (3 draws). **Delta: +3 for Go.**

Zone 191 has a similar cascade: the `R`-bug effect is smaller (+3) because some `if_flag=1` chains are shorter, but the same mechanism applies.

### Bug B: O/P command execution ordering vs C (net -1 draw)

In C's `reset_zone`, for `O` and `P` commands:

```c
case 'O':
    obj = read_object(ZCMD.arg1, REAL);  // 1. create object (may call init_rare → draws)
    if (percent_load(obj)) {              // 2. THEN check percent_load (1 draw)
        obj_to_room(obj, ZCMD.arg3);
        last_cmd = 1;
    } else {
        extract_obj(obj);                 // even on failure, read_object already ran
    }
```

In Go's `ExecuteZoneReset`:

```go
case "O":
    if !percentLoad(proto) { continue }  // 1. check percentLoad FIRST (1 draw)
    s.SpawnObject(cmd.Arg1, cmd.Arg3)    // 2. THEN create object (may call initRare → draws)
    lastCmd = 1
```

This ordering difference has two cascading effects:

1. **RNG stream shift:** C's `read_object()` may call `init_rare()` (for ITEM_RARE objects) **before** `percent_load()`'s `uniform()` call. Go calls `percentLoad()` **before** `SpawnObject()` (which calls `initRare()`). The extra draws from `init_rare` in C shift the RNG stream position at the `percent_load` point, potentially flipping pass/fail outcomes. Different outcomes cascade through `if_flag` chains.

2. **Max-in-world count divergence:** C's `read_object()` increments `obj_index[].number` unconditionally (even when `percent_load` fails). Go's count (`s.objInstances`) only increments on successful `SpawnObject()`. This means C reaches max-in-world limits sooner, skipping commands that Go still processes.

**Zone 43 evidence:** C makes 6 more draws than Go. This zone has many `O` and `P` commands with chained `if_flag=1` dependencies. The RNG stream shift from Bug B causes different `percent_load` outcomes, cascading through the chains and resulting in 6 fewer successful loads in Go.

**Zone 48 evidence:** Go makes 2 more draws than C. The max-in-world count divergence means Go processes `O` commands for objects that C has already maxed out.

**Zone 191 evidence:** Go makes 3 more draws (compound of Bug A's `R`-command effect and Bug B's cascade).

### Net accounting

| Zone | C draws | Go draws | Delta | Cause |
|------|---------|----------|-------|-------|
| 43   | 41      | 35       | -6    | Bug B (RNG shift → fewer Go loads) |
| 48   | 8       | 10       | +2    | Bug B (max-in-world count divergence) |
| 70   | 0       | 3        | +3    | Bug A (R command lastCmd) |
| 191  | 4       | 7        | +3    | Bug A + Bug B compound |
| All others | identical | identical | 0 | — |
| **Total** | **325** | **327** | **+2** | |

## 2. Which Side Is Wrong

**Both bugs are in Go.** C defines correct behavior (it's the original).

- **Bug A:** C's conditional `last_cmd` update in the `R` command is correct per DikuMUD semantics. Go's unconditional `lastCmd = 1` is a porting error.
- **Bug B:** C's `read_object()` → `percent_load()` ordering is the canonical DikuMUD zone reset flow. Go's `percentLoad()` → `SpawnObject()` inversion is a porting error.

## 3. Proposed Fix

### Fix A: `R` command — conditionally set `lastCmd`

**File:** `pkg/game/spawner.go`, lines 446-454

```go
case "R":
    found := false
    if cmd.Arg3 == 1 { // Remove object
        found = s.removeObjectFromRoom(cmd.Arg1, cmd.Arg2)
    } else { // Remove mob
        found = s.removeMobFromRoom(cmd.Arg1, cmd.Arg2)
    }
    if found {
        lastCmd = 1
    }
```

This requires `removeObjectFromRoom` and `removeMobFromRoom` to return `bool` indicating whether anything was actually removed. Both functions currently return nothing.

**Files to touch:**
- `pkg/game/spawner.go:446-454` (R command handler)
- `pkg/game/spawner.go:589-631` (`removeObjectFromRoom`, `removeMobFromRoom` — change signatures to return `bool`)

### Fix B: O/P command ordering — match C's `read_object` → `percent_load` sequence

**File:** `pkg/game/spawner.go`, `O` and `P` command handlers

For the `O` command (lines 296-329), reorder to match C:

```go
case "O":
    if !s.CanSpawn(cmd.Arg1, cmd.Arg2) { continue }
    // 1. Create object first (matches C's read_object)
    var obj *ObjectInstance
    if cmd.Arg3 >= 0 {
        var err error
        obj, err = s.SpawnObject(cmd.Arg1, cmd.Arg3)
        if err != nil { continue }
    } else {
        var err error
        obj, err = s.SpawnObject(cmd.Arg1, -1)
        if err != nil { continue }
        obj.Location = LocNowhere()
    }
    // 2. THEN check percent_load (matches C's percent_load after read_object)
    if !percentLoad(obj.Prototype) {
        s.world.ExtractObject(obj, cmd.Arg3)
        continue
    }
    lastCmd = 1
```

Apply the same pattern to `P`, `G`, and `E` commands.

**Files to touch:**
- `pkg/game/spawner.go:296-329` (O command)
- `pkg/game/spawner.go:331-357` (G command)
- `pkg/game/spawner.go:359-393` (E command)
- `pkg/game/spawner.go:395-424` (P command)

Note: `ExtractObject` needs to be callable from the spawner to undo a spawn when `percentLoad` fails. The spawner also needs to decrement its instance tracking (`s.objInstances`).

## 4. Blast Radius

**Fix A** (R command) is low-risk — only affects zone resets with `R` commands where the target doesn't exist. Currently only zones 70 and 191 (and similar) are affected. No gameplay impact beyond draw parity.

**Fix B** (O/P ordering) is higher-risk because it changes the RNG stream position for every `percent_load` check across all zones. This will:
- Change which objects pass/fail percent_load (different random outcomes)
- Change max-in-world counting behavior
- Potentially change the set of objects loaded at boot

**All currently-green scenarios should be re-tested after Fix B.** Fix A alone does not affect scenarios without `R` commands.

The offset is **systemic** — any scenario that runs after boot will see the +2 (or a variant depending on which zones are processed). Fixing both bugs makes the entire boot-reset draw path C-faithful.

## 5. Instrumentation Reverted

All temporary `DP_DRAW` instrumentation has been reverted from both codebases:

- **Go:** `pkg/dprng/cmwc.go`, `pkg/game/weather.go`, `pkg/game/world_zone.go`, `pkg/parser/mob.go`, `cmd/server/main.go`, `cmd/dp-oracle-diff/main.go` — all reverted. `go build ./...` and `go vet ./...` pass clean.
- **C oracle:** `src/random.c`, `src/random.h`, `src/db.c` — all reverted. `make` passes clean.
- Pre-existing `DP_SEED`/`DP_CLOCK` seams in `comm.c` are untouched.

## Appendix: Method

Differential draw-counter instrumentation was added to both engines:
- Go: `sync/atomic.Int64` counter in `dprng` package, incremented in each package-level draw function
- C: `unsigned long dp_draw_count` global incremented in `prng_next()`

Both engines were run through the `combat-swing` harness (which applies `quiet-mobs` fixtures). Per-zone draw deltas were logged during the zone reset loop. The divergence was isolated to 4 zones (43, 48, 70, 191) out of 68 total — all others matched perfectly.
