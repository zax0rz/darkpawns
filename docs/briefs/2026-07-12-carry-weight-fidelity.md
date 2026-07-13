# DP-1056: Carry Weight Only Enforced for Brand-New Characters — Load Paths Skip Recompute

**Target:** `pkg/game/inventory.go`, `pkg/db/convert.go`, `pkg/game/save.go`, `pkg/game/player.go`, `pkg/combat/formulas.go`
**Repo:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Branch:** Create from `main`, name `fix/carry-weight-live-recompute`
**After fixing:** `go build ./... && go vet ./... && go test ./...`

---

## Problem

`Inventory.SetCapacity(str, dex, level)` computes `MaxWeight` and `Capacity` once at character creation. Three paths never recompute:

1. **`pkg/db/convert.go:55-67` `RecordToPlayer`** — calls `NewCharacter()` (capacity computed from freshly-rolled level-1 stats), then overwrites `p.Stats`/`p.Level` with saved values **without recomputing capacity**. A str-18 level-25 warrior loaded from DB gets whatever the roll happened to produce for capacity.

2. **`pkg/game/save.go:329`** — builds the Player literal with bare `NewInventory()` → `MaxWeight = 0` = weight entirely unenforced on the save-to-world-state path.

3. **`AdvanceLevel()` in `pkg/game/level.go:127`** — never recompute capacity after stats/level change. C evaluates live at every check.

## C Source Behavior

C does NOT cache carry limits. `CAN_CARRY_W(ch)` and `CAN_CARRY_N(ch)` are macros in `src/utils.h:448-449` that compute from current stats every time:

```c
#define CAN_CARRY_W(ch) (str_app[STRENGTH_APPLY_INDEX(ch)].carry_w)
#define CAN_CARRY_N(ch) (5 + (GET_DEX(ch) >> 1) + (GET_LEVEL(ch) >> 1))
```

`STRENGTH_APPLY_INDEX` uses the **str_add** (exceptional strength) field for str=18:

```c
// src/utils.h — STR 18 with StrAdd 0-100 maps to str_app[20..30]
#define STRENGTH_APPLY_INDEX(ch) \
    (GET_STR(ch) >= 18) ? 18 + GET_STR_ADD(ch) : GET_STR(ch)
```

So a warrior with STR 18/97 would get `str_app[25].carry_w`, not `str_app[18].carry_w`.

## Go's Current Gaps

### Gap 1: `combat.CarryWeight(str)` ignores StrAdd

`pkg/combat/formulas.go:288`:
```go
func CarryWeight(str int) int {
    // ...direct index into strApp, no StrAdd lookup
    return strApp[str].CarryW
}
```

And `Inventory.SetCapacity` calls it with raw str only — no StrAdd parameter.

### Gap 2: `MaxCarryWeight()` on Player also ignores StrAdd

`pkg/game/player.go:401` uses a hardcoded `strCarry` array indexed by raw STR (0-18). Str 19+ clamps to 18. No exceptional strength handling.

### Gap 3: `SetCapacity` only called at creation

Called in `NewCharacter()` and `DoStart()`. Never called after `RecordToPlayer` overwrites stats, never called after `AdvanceLevel()`.

## The Fix

**Recommended approach: compute at check-time from live stats (match C).**

1. **Add `STRENGTH_APPLY_INDEX` to `pkg/combat/formulas.go`:**
```go
// StrengthIndex returns the str_app index accounting for exceptional strength.
// Source: utils.h STRENGTH_APPLY_INDEX
func StrengthIndex(str, strAdd int) int {
    if str >= 18 {
        idx := 18 + strAdd
        if idx >= len(strApp) {
            idx = len(strApp) - 1
        }
        return idx
    }
    if str < 0 { return 0 }
    if str >= len(strApp) { return len(strApp) - 1 }
    return str
}
```

2. **Update `CarryWeight` to accept strAdd:**
```go
func CarryWeight(str, strAdd int) int {
    idx := StrengthIndex(str, strAdd)
    return strApp[idx].CarryW
}
```

3. **Update `Player.MaxCarryWeight()` to use StrAdd:**
```go
func (p *Player) MaxCarryWeight() int {
    return combat.CarryWeight(p.Strength, p.GetStrAdd())
}
```

4. **Update `Inventory.CanCarry` to compute weight limit at check-time from the player's live stats, not from cached `MaxWeight`.** The `CanCarry` method takes `str, dex, level` params but is only called with creation-time values. Either:
   - Pass the player's current stats at every call site, or
   - Drop the cached `MaxWeight` field and always compute live

5. **Recompute in `RecordToPlayer`** (interim fix until check-time approach is complete): after overwriting stats, call `p.Inventory.SetCapacity(p.Strength + p.GetStrAdd(), p.Stats.Dex, p.Level)`. Note: this requires passing effective str (raw + StrAdd mapping) to SetCapacity.

6. **Recompute in `AdvanceLevel()`** (interim): call `p.Inventory.SetCapacity(...)` with new stats after level-up completes.

7. **Fix the save.go path** at line 329: after building the Player, call `SetCapacity` with the loaded stats.

## Key Reference Files

- `src/utils.h:448-449` — C macros
- `src/constants.c:1049-1053` — str_app table with carry_w values for indices 20-30 (exceptional str)
- `pkg/combat/formulas.go:288` — `CarryWeight()`
- `pkg/game/player.go:401` — `MaxCarryWeight()`
- `pkg/game/player.go:92` — `GetStrAdd()`
- `pkg/game/inventory.go:165` — `SetCapacity()`
- `pkg/game/inventory.go:30` — `CanCarry()` check
- `pkg/db/convert.go:55-67` — `RecordToPlayer` load path
- `pkg/game/save.go:329` — save/restore path
- `pkg/game/level.go:127` — `AdvanceLevel()`

## str_app carry_w reference (indices 18-30, for exceptional str)

From `src/constants.c`:
- Index 18 (str 18/00): carry_w = 255
- Index 20 (str 18/20): carry_w = 280  
- Index 25 (str 18/50): carry_w = 480
- Index 30 (str 18/100): carry_w = 1750

(Read the full table from constants.c to get all values — these are approximate, verify.)

## Tests

Add a unit test that:
1. Creates a character with str 18, strAdd 50 (warrior-class only)
2. Verifies `MaxCarryWeight()` returns the correct value for str_app[25], NOT str_app[18]
3. Loads via `RecordToPlayer` with different stats and verifies capacity is recomputed
4. Calls `AdvanceLevel()` and verifies capacity is recomputed

**Commit message:** `fix: compute carry weight from live stats with StrAdd support (DP-1056)`
