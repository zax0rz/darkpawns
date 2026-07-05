# Brief: Round 3 Kimi — Panic Swallowing + DB.Exec + Path Traversal

**Issues:** DP-857, DP-852, DP-789
**Date:** 2026-07-05
**Priority:** Medium (3)
**Effort:** S each

---

## DP-857: Recovered Lua/Go panics reported as nil errors

**Problem:** `pkg/scripting/engine.go:243-260` — `RunScript` uses `defer func() { recover() }` to catch Lua panics, but the named return value `err` is never set in the panic path. Recovered panics return `(false, nil)` — callers cannot distinguish between "script ran and returned false" and "script crashed."

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

## DP-852: DB.Exec returns interface{} instead of sql.Result

**Problem:** `pkg/db/player.go:353` — `Exec` method returns `(interface{}, error)` instead of `(sql.Result, error)`. The underlying `db.conn.Exec()` returns `(sql.Result, error)`.

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

## DP-789: PlayerName used directly in filesystem paths

**Problem:** `pkg/agentcli/config.go:84-96` — `Validate()` only checks `PlayerName` is non-empty but doesn't sanitize path separators or `..` traversal. The value flows into `filepath.Join` calls at `client.go:352,357` for log and summary paths.

**Fix:** Add sanitization in `Validate()`:
```go
// Sanitize PlayerName — reject path separators and traversal
if strings.ContainsAny(c.PlayerName, "/\\") || strings.Contains(c.PlayerName, "..") {
    return fmt.Errorf("player_name %q contains invalid characters", c.PlayerName)
}

// Sanitize LogDir — must be a clean path
if c.LogDir != "" {
    c.LogDir = filepath.Clean(c.LogDir)
}
```

**Cite:** No C equivalent — `pkg/agentcli/` is Go-only (DP-Goat agent CLI).

**Regression Test:** Test that `PlayerName = "../../etc"` fails validation. Test that normal names pass.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/agentcli/...`
