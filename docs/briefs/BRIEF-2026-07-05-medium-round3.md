# Brief: Round 3 Medium Fixes — Panic Swallowing + Mob Dice + DB.Exec + Zone L + Path Traversal

**Issues:** DP-857, DP-806, DP-852, DP-895, DP-789
**Date:** 2026-07-05
**Priority:** Medium (5)
**Effort:** S–M

---

## DP-857: Recovered Lua/Go panics reported as nil errors (M)

**Problem:** `pkg/scripting/engine.go:243-260` — `RunScript` uses `defer func() { recover() }` to catch Lua panics, but the named return value `err` is never set in the panic path. Recovered panics return `(false, nil)` — callers cannot distinguish between "script ran and returned false" and "script crashed." The `slog.Warn` log is the only indication of failure.

**Callers affected:** 12+ call sites across `scripts.go`, `world_scriptable.go`, `mobact.go`, `item_transfer.go`, `act_movement.go`, `session/commands.go`. Most silently proceed with fallback behavior (e.g., "You can't give that here." on `ongive` panic).

**Fix:** Set `err` in the recover closure:
```go
defer func() {
    if r := recover(); r != nil {
        slog.Warn("lua script panic, recreating LState", "reason", r, "file", fname, "trigger", triggerName)
        needsRecreate = true
        err = fmt.Errorf("lua script panic: %v", r)  // ADD THIS
    }
    if needsRecreate {
        slog.Info("recreating Lua state after script crash", "file", fname)
        e.l.Close()
        e.l = e.newSafeLState()
    }
}()
```

**Cite:** No C equivalent — Lua scripting is Go-only.

**Regression Test:** Trigger a panic via a script with instruction limit exceeded, verify `err != nil` is returned.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/scripting/...`

---

## DP-806: Mob dice-expression stats silently left at zero (CRITICAL — Fidelity Bust)

**Problem:** `pkg/parser/mob.go:260-279` — The stats line parser assumes 9 space-separated integer fields (`38 -18 -28 38 5 5078 25 4 25`), but actual `.mob` files use `NdS+P` dice notation (`38 -18 -28 38d5+5078 25d4+25`). When `strings.Fields()` produces only 5 tokens, the `if len(stats) >= 9` guard fails and **all dice fields remain zero**. Every mob in the game gets 100 HP (default) and zero base damage.

**Cite:** C source — `src/db.c:1041-1047` (mob load function). Uses `sscanf(line, " %d %d %d %dd%d+%d %dd%d+%d ", t, t+1, ..., t+8)` which natively parses `NdS+P` dice notation. On parse failure, C exits with error. Go silently leaves zeros.

**Fix:** Replace the `strconv.Atoi` calls with a dice-expression parser. Add a helper like:
```go
func parseDiceExpr(s string) (num, sides, plus int, err error) {
    // Parse "NdS+P" format, e.g. "38d5+5078"
    // Support "NdS", "NdS+P", "NdS-P" variants
    // ...
}
```

Then in the stats parsing:
```go
// After parsing level, thac0, ac (first 3 fields):
if len(stats) >= 5 {
    hpNum, hpSides, hpPlus, hpErr := parseDiceExpr(stats[3])
    dmgNum, dmgSides, dmgPlus, dmgErr := parseDiceExpr(stats[4])
    if hpErr == nil {
        mob.HP.Num = hpNum
        mob.HP.Sides = hpSides
        mob.HP.Plus = hpPlus
    }
    if dmgErr == nil {
        mob.Damage.Num = dmgNum
        mob.Damage.Sides = dmgSides
        mob.Damage.Plus = dmgPlus
    }
}
```

**Regression Test:** Add test with real `.mob` file format (`38d5+5078`), verify HP.Num=38, HP.Sides=5, HP.Plus=5078. Also test edge cases: `1d1+0`, `1d1`, `10d10+100`, `0d0+5`.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/parser/... && go test ./pkg/game/...`

---

## DP-852: DB.Exec returns interface{} instead of sql.Result (S)

**Problem:** `pkg/db/player.go:353` — `Exec` method returns `(interface{}, error)` instead of `(sql.Result, error)`. This erases the type contract; callers must type-assert to get `RowsAffected()` or `LastInsertId()`. The underlying `db.conn.Exec()` returns `(sql.Result, error)`.

**Fix:** Change signature:
```go
// Before:
func (db *DB) Exec(query string, args ...interface{}) (interface{}, error) {
    return db.conn.Exec(query, args...)
}

// After:
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
    return db.conn.Exec(query, args...)
}
```

**Cite:** No C equivalent — Go-only DB layer.

**Regression Test:** Verify callers that use this method (if any) still compile. Search for `db.Exec(` usages.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/db/...`

---

## DP-895: Zone 'L' command parsing semantics disagree with C (M)

**Problem:** Three mutually contradictory interpretations of the 'L' zone command:
- `pkg/parser/zon.go:122,172` — says "Lock door", arg3 = lock_state (0/1/2)
- `pkg/parser/parser.go:172,177-179` — says arg3 = key_vnum, validates against objVnums
- `src/db.c:2097-2104` — C truth: 'L' means **"Start/End Looping"** (flow control for zone resets). arg2 = loop control, arg3 = loop count.

The Go port fundamentally misinterpreted 'L' as a door/lock command. This is a fidelity bug that could cause zone reset scripts to malfunction.

**Cite:** C source — `src/db.c:2097-2104` (zone reset execution). C code:
```c
case 'L':         /* Start/End Looping */
    if (!ZCMD.arg2) {
        tmp_cmd = cmd_no;
        loop = ZCMD.arg3;
        last_cmd = 1;
    } else
        if (--loop > 0)
            cmd_no = tmp_cmd;
    break;
```
Also `src/db.c:997` — vnum resolution only resolves arg1 (room vnum) for both 'L' and 'D'.

**Fix:** This needs careful C cross-reference. The 'L' command in zon.go should be relabeled from "Lock door" to "Start/End Looping" with proper comments. The parser.go validation of arg3 as an object vnum is wrong and should be removed. The loop execution logic needs to be implemented (or verified it already exists elsewhere).

**Regression Test:** Parse a zone file with 'L' commands, verify they are interpreted as loop control, not door locking.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/parser/...`

---

## DP-789: PlayerName used directly in filesystem paths (S)

**Problem:** `pkg/agentcli/config.go:84-96` — `Validate()` only checks `PlayerName` is non-empty but doesn't sanitize path separators or `..` traversal. The value flows into `filepath.Join` calls at `client.go:352,357` for log and summary paths. A name like `../../etc` escapes intended subdirectories. Same for `LogDir`.

**Cite:** No C equivalent — `pkg/agentcli/` is Go-only (DP-Goat agent CLI).

**Fix:** Add sanitization in `Validate()`:
```go
func (c *AgentConfig) Validate() error {
    // ... existing checks ...

    // Sanitize PlayerName — reject path separators and traversal
    if strings.ContainsAny(c.PlayerName, "/\\") || strings.Contains(c.PlayerName, "..") {
        return fmt.Errorf("player_name %q contains invalid characters", c.PlayerName)
    }

    // Sanitize LogDir — must be a clean absolute or relative path
    if c.LogDir != "" {
        c.LogDir = filepath.Clean(c.LogDir)
    }

    return nil
}
```

**Regression Test:** Test that `PlayerName = "../../etc"` fails validation. Test that normal names pass.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/agentcli/...`
