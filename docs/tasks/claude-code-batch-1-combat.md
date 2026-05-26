# Claude Code Batch — Run 1: Combat System Fixes

## Issues
- DP-389: Wimpy auto-flee hooks never wired (URGENT)
- DP-391: Flee uses flat 50% coin flip instead of directional loop (HIGH)
- DP-390: Flee XP penalty skipped for level ≤10 (HIGH)
- DP-394: Assist/order locked to LVL_IMMORT (HIGH)
- DP-397: Hit doesn't auto-dismount (MEDIUM)

## Task: Wire wimpy auto-flee (DP-389)

**File:** `pkg/combat/fight_core.go:45-46, 542-556`

`DoFlee` and `DoRetreat` are package-level hook variables checked when HP drops below wimpy threshold. They are never assigned in production — only mocked in tests.

**Fix:**
1. Find where `DoFlee` and `DoRetreat` are declared as hook variables
2. Find `cmdFlee` in `pkg/session/combat_cmds.go` — that's the actual flee implementation
3. Wire the hooks at init/startup: `fight_core.DoFlee = session.CmdFlee` (or whatever the correct import path is — check for import cycles)
4. If there's an import cycle, use the same hook pattern the codebase already uses (check how `DoAction` or other game→session hooks are wired)

**C source:** `src/fight.c` — wimpy auto-flee in `process_hit()`

## Task: Fix flee mechanics (DP-391)

**File:** `pkg/session/combat_cmds.go:138-144`

C source (`src/fight.c do_flee`):
1. Loop up to 6 times trying random directions
2. For each: check exit exists, check door is open, check room isn't DEATH flag
3. Try MovePlayer — if it succeeds, break
4. If all 6 fail, player stays in combat

Go currently: flat 50% random fail, then pick any exit.

**Fix:** Rewrite `cmdFlee` to loop up to 6 random directions, checking exits/doors/room flags before attempting movement.

**C source:** `src/fight.c — do_flee()`

## Task: Fix flee XP penalty (DP-390)

**File:** `pkg/session/combat_cmds.go:148-169`

C source applies base XP loss (`loss = dam * level_of_opponent`) to ALL levels, then adds bonus penalty for level > 10.

Go only applies loss inside `if level > 10` block. Players ≤10 flee for free.

**Fix:** Move `LoseExp` call outside the `if level > 10` block so it applies to all levels.

**C source:** `src/fight.c — do_flee()` XP penalty section

## Task: Unlock assist/order for mortals (DP-394)

**File:** `pkg/session/commands.go:211, 221`

Both commands registered with `LVL_IMMORT` minimum. C source has them available to all mortals.

**Fix:** Change minimum level from `LVL_IMMORT` to `0` (or whatever the mortal minimum is — check other combat commands for the pattern).

**C source:** `src/act.other.c — do_assist()`, `src/act.other.c — do_order()`

## Task: Auto-dismount on hit (DP-397)

**File:** `pkg/session/combat_cmds.go` (`cmdHit`)

C source calls `do_dismount` before `hit` if player is mounted.

**Fix:** At the top of `cmdHit`, check if player has `affMounted` flag. If so, remove it and send dismount message before proceeding with attack.

**C source:** `src/act.other.c — do_hit()`

## Verification
1. `go build ./...` — must pass
2. `go vet ./...` — must pass
3. `go test ./...` — must pass
