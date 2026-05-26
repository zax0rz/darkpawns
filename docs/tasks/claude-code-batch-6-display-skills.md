# Claude Code Batch — Run 6: Combat Skills & Display

## Issues
- DP-384: itemTypeString mappings scrambled (HIGH)
- DP-392: StunTarget ignored, mobs immune to knockdown (HIGH)
- DP-393: Sleeper hold mechanically useless (HIGH)
- DP-395: Order command is a mock (HIGH)
- DP-386: Carry weight uses hardcoded formula (MEDIUM)
- DP-396: Shoot command truncated (MEDIUM)

## Task: Fix itemTypeString mappings (DP-384)

**File:** `pkg/session/examine.go:254`

The switch statement maps type integers to wrong names. Compare with `pkg/game/item_helpers.go` constants:

| Constant | Value | examine.go maps to | Should be |
|----------|-------|-------------------|-----------|
| ITEM_LIGHT | 1 | "light source" | correct |
| ITEM_SCROLL | 2 | "scroll" | correct |
| ITEM_WAND | 3 | "wand" | correct |
| ITEM_STAFF | 4 | "staff" | correct |
| ITEM_WEAPON | 5 | "weapon" | correct |
| ... | ... | ... | ... |
| ITEM_CONTAINER | 15 | "container" | correct |
| ITEM_DRINKCON | 17 | "drink container" | correct |
| ITEM_FOOD | 19 | "jewelry" | **WRONG — should be "food"** |
| ITEM_MONEY | 20 | "money" | correct |

Fix each wrong mapping. Use `pkg/game/item_helpers.go` as the source of truth.

**C source:** Not applicable — Go-specific display issue

## Task: Wire sendSkillResult for mobs (DP-392)

**File:** `pkg/command/skill_commands.go:1371-1381`

`sendSkillResult` handles `TargetFalls` for Players but explicitly skips Mobs with a comment.

**Fix:**
1. Add `SetPosition` to the mob interface (or use the existing `MobInstance.SetPosition` if it exists)
2. Remove the mob skip in `sendSkillResult` — apply knockdown/stun to mobs too
3. Check `pkg/game/mob_instance.go` for position management

**C source:** `src/fight.c` — skill result handling applies to all targets

## Task: Implement sleeper hold (DP-393)

**File:** `pkg/game/skill_c10_combat.go:214`

`DoSleeper` returns `StunTarget: true` but nothing acts on it.

**Fix:** In `DoSleeper`, after the skill check succeeds:
1. Set target position to `PosSleeping`
2. Apply `AFF_SLEEP` affect to target
3. Don't rely on `StunTarget` — set these directly like other combat skills do

**C source:** `src/fight.c — do_sleeper()`

## Task: Implement order command (DP-395)

**File:** `pkg/session/cmd_combat_special.go:9-39`

`cmdOrder` finds the mob and prints a message but doesn't execute the command.

**Fix:** After finding the charmed follower, execute the remaining command string on the mob. This is the hardest fix in this batch — you need to dispatch a command string to a mob's AI. Check how mob scripts or Lua scripting dispatches commands.

**C source:** `src/act.other.c — do_order()`

## Task: Fix carry weight formula (DP-386)

**File:** `pkg/game/item_transfer.go:17`

Go uses `ch.Inventory.Capacity * 10`. C uses `str_app[GET_STR].carry_w` strength table.

**Fix:** Look up the strength table in `pkg/game/` (likely `class_tables.go` or similar). Replace the hardcoded formula with the table lookup.

**C source:** `src/act.item.c — CAN_CARRY_W()`

## Task: Scope shoot command (DP-396)

**File:** `pkg/game/skill_c10_combat.go:110-136`

This is a larger feature — ranged combat with direction parsing, exit traversal, and mob dragging. Mark this as "needs design" and skip for this batch. The other fixes in this batch are higher priority.

**C source:** `src/act.other.c — do_shoot()`

## Verification
1. `go build ./...` — must pass
2. `go vet ./...` — must pass
3. `go test ./...` — must pass
