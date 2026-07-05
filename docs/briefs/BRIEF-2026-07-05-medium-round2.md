# Brief: Round 2 Medium Fixes — Bearer + Divide-by-Zero + Goroutine Leak + Race + Act Token

**Issues:** DP-732, DP-733, DP-734, DP-753, DP-760
**Date:** 2026-07-05
**Priority:** Medium (4) / Low (1)
**Effort:** S each

---

## DP-732: Bearer token scheme parsing is case-sensitive (RFC 7235 violation)

**Problem:** `web/auth.go:34` uses `strings.TrimPrefix(authHeader, "Bearer ")` which only matches capital-B. RFC 7235 §2.6 says auth-scheme comparison is case-insensitive. Clients sending `bearer`, `BEARER`, etc. are rejected even with valid tokens.

**Fix:** In `AuthMiddleware` (`web/auth.go:34`), replace the TrimPrefix approach with a case-insensitive check:
```go
// Before:
token := strings.TrimPrefix(authHeader, "Bearer ")
if token == authHeader {
    http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
    return
}

// After:
if len(authHeader) < 7 || !strings.EqualFold(authHeader[:6], "Bearer") || authHeader[6] != ' ' {
    http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
    return
}
token := authHeader[7:]
```

**Cite:** No C equivalent — `web/` is a Go-only HTTP API layer. The C MUD has no HTTP auth.

**Regression Test:** Test that `Authorization: bearer <valid_token>`, `BEARER <valid_token>`, and `Bearer <valid_token>` all pass. Test that `BearerX <token>` and missing header still reject.

**Verification:** `go build ./... && go vet ./... && go test ./web/...`

---

## DP-733: GetQueueStats divides by zero when TasksSubmitted is 0

**Problem:** `pkg/optimization/advanced_pool.go:265` computes `success_rate = TasksCompleted / TasksSubmitted` without guarding against zero. Produces `+Inf` on fresh pools. The sibling `GetMetrics()` (line 245) already guards the analogous `WorkerUtilization` division.

**Fix:** Add the same guard pattern at line 265:
```go
// Before:
stats["success_rate"] = float64(metrics.TasksCompleted) / float64(metrics.TasksSubmitted)

// After:
if metrics.TasksSubmitted > 0 {
    stats["success_rate"] = float64(metrics.TasksCompleted) / float64(metrics.TasksSubmitted)
} else {
    stats["success_rate"] = 0.0
}
```

**Cite:** No C equivalent — `pkg/optimization/` is Go-only infrastructure code.

**Regression Test:** Create a fresh `AdvancedWorkerPool`, call `GetQueueStats()`, assert `success_rate` is `0.0`, not `+Inf`.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/optimization/...`

---

## DP-734: cleanTransitItems goroutine leaks on Engine discard

**Problem:** `pkg/scripting/engine.go:144` starts `go engine.cleanTransitItems()` which runs an infinite `for range ticker.C` loop (line 160). No `Close()`/`Shutdown()` method exists on `Engine` to stop it. Also, `e.l` (the `*lua.LState`) is never closed. Leaked goroutines prevent GC of the entire Lua runtime.

**Fix:**
1. Add a `done chan struct{}` field to `Engine`:
   ```go
   type Engine struct {
       // ... existing fields ...
       done chan struct{}
   }
   ```
2. Initialize in `NewEngine` and pass to goroutine:
   ```go
   engine.done = make(chan struct{})
   go engine.cleanTransitItems()
   ```
3. Add stop signal to `cleanTransitItems`:
   ```go
   func (e *Engine) cleanTransitItems() {
       ticker := time.NewTicker(10 * time.Second)
       defer ticker.Stop()
       for {
           select {
           case <-ticker.C:
               e.mu.Lock()
               for id, entry := range e.transitItems {
                   if time.Since(entry.placedAt) > transitItemTTL {
                       slog.Warn("transitItem orphaned, discarding", "instanceID", id, "vnum", entry.obj.GetVNum())
                       delete(e.transitItems, id)
                   }
               }
               e.mu.Unlock()
           case <-e.done:
               return
           }
       }
   }
   ```
4. Add `Close()` method:
   ```go
   func (e *Engine) Close() {
       close(e.done)
       e.l.Close()
   }
   ```
5. Ensure `Close()` is called during server shutdown wherever `NewEngine` is used.

**Cite:** No C equivalent — the Lua scripting engine is Go-only.

**Regression Test:** Create an Engine, call `Close()`, verify goroutine stops (e.g., `runtime.NumGoroutine()` delta check).

**Verification:** `go build ./... && go vet ./... && go test ./pkg/scripting/...`

---

## DP-753: Data race on tc.hasGMCP between handleConn and writeLoop

**Problem:** `pkg/telnet/listener.go:251` declares `hasGMCP bool`. Written in `handleConn` goroutine (lines 779, 802) and read in `writeLoop` goroutine (line 584) without synchronization. Violates Go memory model.

**Fix:** Replace `bool` with `atomic.Bool`:
```go
// In telnetConn struct (line 251):
hasGMCP atomic.Bool

// In handleConn init (line 262): remove hasGMCP: false (zero value is fine)

// In writeLoop guard (line 584):
if tc.hasGMCP.Load() {

// In WILL handler (line 779):
tc.hasGMCP.Store(true)

// In DO handler (line 802):
tc.hasGMCP.Store(true)
```

**Cite:** No C equivalent — telnet/GMCP negotiation is Go-only.

**Regression Test:** `go test -race ./pkg/telnet/...`

**Verification:** `go build ./... && go vet ./... && go test -race ./pkg/telnet/...`

---

## DP-760: Social position failure sends raw $N token to player

**Problem:** `pkg/game/act_social.go:94` uses `ch.SendMessage("$N is not in a proper position for that.\r\n")` which sends the literal `$N` token instead of the target's name. The C source correctly uses `act()` for `$N` expansion.

**Fix:** Replace `SendMessage` with `Act`:
```go
// Before (line 94):
ch.SendMessage("$N is not in a proper position for that.\r\n")

// After:
Act(nil, false, ch, targetActor, nil, nil, "$N is not in a proper position for that.\r\n", "", ToChar)
```

**Cite:** C source — `src/act.social.c:141` (`do_social` function). The C code uses:
```c
act("$N is not in a proper position for that.", FALSE, ch, 0, vict, TO_CHAR | TO_SLEEP);
```
The Go port incorrectly replaced `act()` with `SendMessage`, losing `$N` expansion. The `Act()` function in Go is at `pkg/game/act.go:456`. Note: this caller uses `nil` for world (single-target `ToChar`), matching the pattern used throughout `DoAction` for `ToChar`/`ToVict` sends (see lines 64, 101, 106).

**Regression Test:** Add a test that performs a social with a target below `MinVictimPosition` and asserts the actor sees the target's name, not the literal `$N`.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/game/...`
