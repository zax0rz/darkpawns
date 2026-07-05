# Brief: Round 5 — SQLite DSN + Parser Tilde Whitespace + ObjectPool.Clear + Audit Close + Gauge Floor + Lua Bool Return + Profile Filenames + Lua Sandbox Registry

**Issues:** DP-845, DP-843, DP-842, DP-841, DP-839, DP-834, DP-833, DP-840
**Date:** 2026-07-05
**Priority:** Low (7) + Medium (1)
**Effort:** S each

---

## DP-845: SQLite DSN constructed without URI-escaping the file path (LOW → infra risk)

**Problem:** `pkg/storage/sqlite.go:29` — `NewSQLiteBackend` builds the DSN as:
```go
db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
```
If `dbPath` contains URI-special characters (`?`, `#`, `%`), the resulting DSN is malformed — the driver silently ignores WAL mode, degrading concurrency.

**Fix:** Use `net/url` to properly encode the path:
```go
import "net/url"

func NewSQLiteBackend(dbPath string) (*SQLiteBackend, error) {
    dir := filepath.Dir(dbPath)
    if err := os.MkdirAll(dir, 0o750); err != nil {
        return nil, fmt.Errorf("create db dir: %w", err)
    }

    // file:// scheme + path-escaped dbPath + query params
    dsn := url.URL{
        Scheme:   "file",
        Path:     dbPath,
        RawQuery: "_journal_mode=WAL&_busy_timeout=5000",
    }.String()
    db, err := sql.Open("sqlite3", dsn)
    // ... rest unchanged
```
Note: mattn/go-sqlite3 accepts `file:path?params` format. If `file://` causes issues, use simple approach: validate dbPath is absolute and doesn't contain `?` or `#`, or use `filepath.Clean` + manual check.

**Cite:** No C counterpart — this is Go-only storage layer.

**Regression test:** `TestNewSQLiteBackend_DSNWithSpecialChars` — pass a dbPath with spaces (valid), verify sql.Open succeeds. If time, test that a path containing `?` produces a warning or is rejected.

**Verification:** `go build ./pkg/storage/...` passes. Existing tests pass.

---

## DP-843: readTildeString doesn't trim whitespace before checking tilde terminator (LOW → parser fidelity)

**Problem:** `pkg/parser/wld.go:120-131` — `readTildeString` checks `strings.HasSuffix(line, "~")` on the raw scanner line. If a world file line has trailing whitespace before `~` (e.g., `"A dark room~  "`), the check fails and the parser reads past the intended end-of-string.

Meanwhile, `pkg/parser/obj.go:209-218` correctly does `trimmed := strings.TrimSpace(descLine)` before checking.

**Fix:** In `pkg/parser/wld.go`, add TrimSpace in the loop:
```go
func readTildeString(scanner *bufio.Scanner) (string, error) {
    var parts []string
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())  // ADD THIS
        if strings.HasSuffix(line, "~") {
            parts = append(parts, strings.TrimSuffix(line, "~"))
            return strings.Join(parts, "\n"), nil
        }
        parts = append(parts, line)
    }
    return "", fmt.Errorf("unexpected EOF while reading ~-terminated string")
}
```

**Cite:** The C parser uses `fread_string` in `src/db.c` which reads until `~` and handles whitespace naturally via fgets + manual scanning. The Go port should match that tolerance.

**Regression test:** `TestReadTildeString_TrailingWhitespace` — pass a scanner with `"hello ~  "` (trailing spaces after tilde) — should return `"hello "`.

**Verification:** `go test ./pkg/parser/...` passes.

---

## DP-842: ObjectPool.Clear() resets borrowed/created counters while objects are still in-flight (LOW → concurrency)

**Problem:** `pkg/optimization/object_pool.go:168-176` — `Clear()` sets `op.borrowed = 0` and `op.created = 0` while callers may still hold borrowed objects. When those callers later `Put()`, `borrowed` goes negative. Also `created == 0` means the `created < maxSize` guard allows over-creation.

**Fix:** Guard Clear() against in-flight objects:
```go
func (op *ObjectPool) Clear() {
    op.mu.Lock()
    defer op.mu.Unlock()

    if op.borrowed > 0 {
        slog.Warn("object pool clear skipped: objects still borrowed",
            "borrowed", op.borrowed, "created", op.created)
        // Only drain the available pool, don't reset counters
        op.pool = make([]interface{}, 0, op.maxSize)
        op.stats.LastReset = time.Now()
        return
    }

    op.pool = make([]interface{}, 0, op.maxSize)
    op.created = 0
    op.borrowed = 0
    op.stats.LastReset = time.Now()
}
```

**Cite:** No C counterpart — optimization layer is Go-only.

**Regression test:** `TestObjectPool_ClearWithBorrowedObjects` — Get() an object, then call Clear(), then Put() it back. Verify `borrowed` is 0 (not -1).

**Verification:** `go test ./pkg/optimization/...` passes.

---

## DP-841: AuditLogger.Close() silently discards file close errors (LOW → data loss)

**Problem:** `pkg/audit/logger.go:76-79` — Close discards the error:
```go
func (a *AuditLogger) Close() {
    _ = a.file.Close()
}
```
For audit logs, close errors can be the only signal that buffered writes failed.

**Fix:** Return the error and log it:
```go
func (a *AuditLogger) Close() error {
    if err := a.file.Close(); err != nil {
        slog.Error("audit log close failed", "error", err)
        return err
    }
    return nil
}
```
Then update all callers of `Close()` to handle the error (or at minimum, log it).

**Cite:** No C counterpart — Go audit logger.

**Regression test:** `TestAuditLogger_CloseReturnsError` — create logger with a temp file, close it, verify no error. (Error-on-close is hard to trigger in test, but the signature change is the fix.)

**Verification:** `go build ./pkg/audit/...` passes. Check all callers compile.

---

## DP-839: Active connection gauge can go negative on unmatched close events (LOW → metrics)

**Problem:** `pkg/metrics/metrics.go:109-111` — `ConnectionClosed()` unconditionally decrements:
```go
func ConnectionClosed() {
    connectionsActive.Dec()
}
```
A double-close or unmatched close makes the gauge negative, polluting dashboards.

**Fix:** Add a floor guard:
```go
func ConnectionClosed() {
    if v := connectionsActive.Value(); v > 0 {
        connectionsActive.Dec()
    } else {
        connectionsUnderflow.Inc()  // optional: track the anomaly
    }
}
```
Register a `connectionsUnderflow` counter (or just skip the else).

**Cite:** No C counterpart — Go metrics layer.

**Regression test:** `TestConnectionClosed_NegativeFloor` — call ConnectionClosed() without ConnectionOpened(). Verify gauge stays >= 0.

**Verification:** `go build ./pkg/metrics/...` passes.

---

## DP-834: Lua function returning boolean true is treated as "not handled" (LOW → scripting)

**Problem:** `pkg/scripting/engine.go:468-472`:
```go
if ret.Type() == lua.LTNumber {
    return lua.LVAsNumber(ret) == 1, nil
}
return false, nil
```
If a Lua script returns `true` (boolean), the `LTNumber` check fails and it's treated as "not handled". The C Lua API convention is that both `1` and `true` mean handled.

**Fix:** Also handle boolean true:
```go
if ret.Type() == lua.LTNumber {
    return lua.LVAsNumber(ret) == 1, nil
}
if ret == lua.LTrue {
    return true, nil
}
return false, nil
```

**Cite:** C Lua scripts in `src/` use `return 1` convention, but Lua idioms also support `return true`. The Go gopher-lua binding should handle both.

**Regression test:** `TestScriptReturnBoolean` — create a Lua state with a function that returns `true`, call CallFunction, verify it returns `true, nil`.

**Verification:** `go test ./pkg/scripting/...` passes.

---

## DP-833: Profile output filenames can overwrite runs started in the same second (LOW → data loss)

**Problem:** `profiling/profiler.go:53,110,151,195,233` — All profile filenames use `time.Now().Unix()` (second precision) with `os.O_TRUNC`. Two runs within one second silently overwrite each other.

**Fix:** Use nanosecond precision:
```go
// Replace all time.Now().Unix() with:
ts := time.Now().UnixNano()
// Or for shorter filenames: time.Now().UnixMilli()
```
Apply to all 5 profile output sites.

**Cite:** No C counterpart — Go profiling tool.

**Regression test:** `TestProfileFilename_Uniqueness` — call the filename generation twice in rapid succession, verify filenames differ.

**Verification:** `go build ./profiling/...` passes.

---

## DP-840: Lua sandbox nils package global but leaves package.searchers in registry (LOW → security)

**Problem:** `pkg/scripting/engine.go:98-99`:
```go
L.SetGlobal("package", lua.LNil)
```
This only nils the global variable. `L.OpenLibs()` on line 58 registered `package.searchers` in the Lua registry, which persists. While `debug` is also nil'd (blocking casual registry access), the registry entries remain.

**Fix:** After nil-ing the global, also clear registry entries. With gopher-lua:
```go
// After L.SetGlobal("package", lua.LNil):
// Also nil out package.loaded and package.preload from registry
L.SetField(L.Get(lua.RegistryIndex), "_LOADED", lua.LNil)
L.SetField(L.Get(lua.RegistryIndex), "_PRELOAD", lua.LNil)
```
Or, better yet, don't open the package library at all by calling `OpenLibs` selectively.

**Cite:** No C counterpart — Go Lua sandbox is a new layer.

**Regression test:** Verify that `require` still fails after the sandbox setup (it should — no loaders available).

**Verification:** `go test ./pkg/scripting/...` passes.

---

## Build Gate

```bash
go build ./...
go vet ./...
go test ./...
```

All three MUST pass before committing. Commit as:
```
fix: reek round 5 — DSN escaping, tilde parser, ObjectPool, audit close, gauge floor, Lua bool return, profile names, sandbox registry (DP-845, DP-843, DP-842, DP-841, DP-839, DP-834, DP-833, DP-840)
```
