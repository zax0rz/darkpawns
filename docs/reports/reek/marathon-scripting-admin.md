# Code Audit: pkg/scripting, pkg/admin, pkg/dreaming, pkg/command

**Date:** 2026-05-15  
**Auditor:** Reek (code crawler)  
**Scope:** Lua binding safety, admin API security, command injection, error handling, resource leaks, logic bugs  
**Files reviewed:**
- `pkg/scripting/engine.go`, `pkg/scripting/types.go`
- `pkg/admin/router.go`, `pkg/admin/handlers.go`, `pkg/admin/login.go`, `pkg/admin/agent_store.go`, `pkg/admin/log_buffer.go`
- `pkg/dreaming/dream.go`, `pkg/dreaming/extract.go`, `pkg/dreaming/graph.go`
- `pkg/command/registry.go`, `pkg/command/middleware.go`, `pkg/command/admin_commands.go`, `pkg/command/interface.go`
- `web/auth.go`

---

## Finding 1 — RateLimitMiddleware: data race + cross-session interference

**Severity:** HIGH  
**File:** `pkg/command/middleware.go:44-56`  
**Category:** Logic bug, concurrency

```go
func RateLimitMiddleware(minInterval time.Duration) Middleware {
    return func(next Handler) Handler {
        var lastCommandTime time.Time  // ← shared across ALL sessions
        return func(s common.CommandSession, args []string) error {
            now := time.Now()
            if !lastCommandTime.IsZero() && now.Sub(lastCommandTime) < minInterval {
                return nil // silently drop
            }
            lastCommandTime = now
            return next(s, args)
        }
    }
}
```

**What:** The `lastCommandTime` variable is captured by the closure and shared across ALL sessions that pass through this middleware. Two problems:
1. **Data race:** Concurrent goroutines (different player sessions) read/write `lastCommandTime` without synchronization. This is a textbook Go data race.
2. **Cross-session throttling:** If Player A runs a command at T=0, Player B's command at T=0.5s (within the interval) is silently dropped — even though Player B hasn't exceeded their own rate limit.

**Why it matters:** In a MUD with multiple concurrent players, this middleware effectively rate-limits the entire server to one command per interval across all players, rather than per-session. Additionally, the data race can corrupt the time.Time value, causing undefined behavior.

**Fix:** Either remove this middleware entirely (the admin API already has per-IP rate limiting via `auth.NewIPRateLimiter()`), or make `lastCommandTime` per-session by storing it in the `CommandSession` interface. For example:

```go
func RateLimitMiddleware(minInterval time.Duration) Middleware {
    return func(next Handler) Handler {
        return func(s common.CommandSession, args []string) error {
            now := time.Now()
            if lastCmd, ok := s.GetTempData("_lastCmd").(time.Time); ok && now.Sub(lastCmd) < minInterval {
                return nil
            }
            s.SetTempData("_lastCmd", now)
            return next(s, args)
        }
    }
}
```

---

## Finding 2 — TransitItems map keyed by vnum: item loss on duplicate instances

**Severity:** HIGH  
**File:** `pkg/scripting/engine.go:28, 1773-1822`  
**Category:** Logic bug

```go
type transitEntry struct {
    obj       ScriptableObject
    placedAt  time.Time
}

// In luaObjFrom:
e.transitItems[vnum] = &transitEntry{obj: removed, placedAt: time.Now()}
```

**What:** The `transitItems` map uses the object prototype VNum as its key. In Dark Pawns, multiple runtime instances of the same prototype can exist simultaneously (e.g., two copies of a "rusty sword" with VNum 1001). If `objfrom` is called for two different instances with the same VNum, the second entry silently overwrites the first, and the first item is lost forever — neither in the room, nor in any inventory, nor in the transit map.

**Why it matters:** Any Lua script that moves two items of the same type in sequence (e.g., a cityguard script picking up multiple dropped weapons) will silently delete items from the game world. This is a persistent data loss bug that players would notice as "items vanishing."

**Fix:** Use the object's unique instance ID instead of VNum as the map key:

```go
transitItems map[int]*transitEntry // key = obj.GetInstanceID()
```

And update `luaObjFrom`/`luaObjTo` to use `obj_id` instead of `vnum` for transit lookups.

---

## Finding 3 — luaCanSee: stack imbalance in dark room path

**Severity:** MEDIUM  
**File:** `pkg/scripting/engine.go:2458-2480`  
**Category:** Logic bug

```go
L.GetGlobal("me")                    // pushes "me" table
if meTbl, meOk := L.Get(-1).(*lua.LTable); meOk {
    roomL := meTbl.RawGetString("room")
    if observerRoom, roomOk := roomL.(lua.LNumber); roomOk && e.world != nil && int(observerRoom) > 0 {
        roomVNum := int(observerRoom)
        if e.world.IsRoomDark(roomVNum) {
            L.Push(lua.LBool(false))  // push false
            L.Pop(1)                  // ← pops the false we just pushed!
            return 1                  // returns "me" table instead of false
        }
    }
}
L.Pop(1)  // pop "me" table (correct path)
```

**What:** In the dark room code path, `L.Push(lua.LBool(false))` pushes the return value, then `L.Pop(1)` immediately pops it, leaving the "me" table on top of the stack. `return 1` tells Lua "return 1 value from the stack top" — which is the "me" table, not the boolean `false`. Lua scripts calling `cansee(ch)` in a dark room would receive a table instead of a boolean, which is truthy in Lua, meaning they'd incorrectly conclude they can see the target.

**Why it matters:** Darkness checks silently fail, allowing scripts to target invisible or hidden characters in dark rooms. This affects any script that uses `cansee()` as a guard condition (combat AI, spell targeting, etc.).

**Fix:** Remove the erroneous `L.Pop(1)`:

```go
if e.world.IsRoomDark(roomVNum) {
    L.Push(lua.LBool(false))
    return 1  // return false directly
}
```

---

## Finding 4 — LState() accessor bypasses engine mutex

**Severity:** MEDIUM  
**File:** `pkg/scripting/engine.go:34-36`  
**Category:** Concurrency

```go
func (e *Engine) LState() *lua.LState {
    return e.l
}
```

**What:** The `LState()` method returns the raw Lua state pointer without requiring the caller to hold `e.mu`. The comment says "caller must NOT hold the engine mutex when calling into the LState" but this is advisory, not enforced. Any code that calls `LState()` and then operates on the returned state does so without synchronization against `RunScript()`, which holds `e.mu`.

**Why it matters:** If any code path calls `LState()` and then modifies or reads the Lua state while `RunScript()` is executing a script, the Lua state becomes corrupted. gopher-lua is not thread-safe.

**Fix:** Either remove the `LState()` accessor entirely (the comment says it exists "only for code that needs read-only access outside of script execution" — audit whether any caller actually needs it), or document that it must only be called when no script is running and verify no callers hold it during script execution.

---

## Finding 5 — Dreaming: AgentID not sanitized for path traversal in file operations

**Severity:** MEDIUM  
**File:** `pkg/dreaming/dream.go:39-40, 49-50, 125-144`  
**Category:** Path traversal

```go
graphFile := filepath.Join(cfg.OutputDir, cfg.AgentID, "memory-graph.json")
// ...
sessionPath := filepath.Join(cfg.SessionDir, cfg.AgentID)
entries, err := os.ReadDir(sessionPath)
// ...
os.MkdirAll(filepath.Dir(graphFile), 0755)
os.WriteFile(graphFile, graphData, 0644)
```

**What:** `cfg.AgentID` is interpolated directly into file paths via `filepath.Join` without sanitization. If `AgentID` contains `../` (e.g., `"../../etc"`), the dreaming cycle would read session files from and write memory graphs to arbitrary locations on the filesystem.

**Why it matters:** If AgentID is ever derived from user input (e.g., a web form, API parameter, or environment variable that an attacker can influence), this enables arbitrary file read and write. Even if AgentID is currently hardcoded, defense-in-depth requires validation.

**Fix:** Validate AgentID against a safe pattern before use:

```go
func isValidAgentID(id string) bool {
    for _, c := range id {
        if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
            return false
        }
    }
    return len(id) > 0 && len(id) <= 64
}
```

---

## Finding 6 — Admin login doesn't use LoginAttemptTracker

**Severity:** MEDIUM  
**File:** `pkg/admin/login.go:36-79`  
**Category:** Auth bypass / brute-force

```go
func handleLogin(database *db.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ... password verification ...
        if err := bcrypt.CompareHashAndPassword([]byte(rec.Password), []byte(req.Password)); err != nil {
            http.Error(w, `{"error":"invalid password"}`, http.StatusUnauthorized)
            return
        }
        // No tracking of failed attempts
    }
}
```

**What:** The admin login endpoint uses `bcrypt.CompareHashAndPassword` for password verification (good), but doesn't integrate with the `auth.LoginAttemptTracker` that exists in the codebase. The tracker provides per-IP lockout after N failed attempts, which is the standard defense against brute-force attacks. Currently, the only protection is the general IP rate limiter (5 req/s, burst 10), which allows ~300 password guesses per minute per IP.

**Why it matters:** An attacker can brute-force admin passwords at 300 attempts/minute per IP. With bcrypt's computational cost, this also creates a denial-of-service vector (bcrypt is intentionally slow, and 300 concurrent bcrypt operations could exhaust server resources).

**Fix:** Create a `LoginAttemptTracker` in main.go and pass it to `handleLogin`, checking `IsLocked(ip)` before password verification and calling `RecordFailure(ip)` / `RecordSuccess(ip)`:

```go
func handleLogin(database *db.DB, loginTracker *auth.LoginAttemptTracker) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ip := auth.GetIPFromRequest(r)
        if locked, remaining := loginTracker.IsLocked(ip); locked {
            http.Error(w, fmt.Sprintf(`{"error":"locked out, try again in %s"}`, remaining.Round(time.Second)), http.StatusTooManyRequests)
            return
        }
        // ... password verification ...
        if err := bcrypt.CompareHashAndPassword(...); err != nil {
            loginTracker.RecordFailure(ip)
            http.Error(w, `{"error":"invalid password"}`, http.StatusUnauthorized)
            return
        }
        loginTracker.RecordSuccess(ip)
        // ... generate JWT ...
    }
}
```

---

## Finding 7 — Admin role determined solely by player level

**Severity:** MEDIUM  
**File:** `pkg/admin/login.go:60-65`  
**Category:** Auth bypass

```go
role := "player"
if rec.Level >= 50 {
    role = "admin"
} else if rec.Level >= 33 {
    role = "builder"
}
```

**What:** Admin and builder roles are assigned based solely on player level (>=50 = admin, >=33 = builder). There's no explicit role field in the player database. This means:
1. Any player who reaches level 50 automatically becomes an admin with full API access.
2. Role changes happen instantly on level-up (no approval workflow).
3. There's no way to revoke admin access without reducing the player's level.

**Why it matters:** In a MUD where leveling is part of the game mechanics, tying administrative access to game progression is fragile. If the level cap changes, or if a bug allows level inflation, administrative access spreads uncontrollably. The `isAdmin()` check in `admin_commands.go` (line ~312) uses a different threshold (`lvlGod = 34`), creating an inconsistency between in-game commands (level >= 34) and the HTTP API (level >= 50).

**Fix:** Add an explicit `role` column to the player database. Default to "player". Only grant "builder" or "admin" through explicit assignment. This decouples game mechanics from administrative access.

---

## Finding 8 — AgentStore.Save() can race with concurrent readers

**Severity:** LOW  
**File:** `pkg/admin/agent_store.go:87-99`  
**Category:** Concurrency

```go
func (s *AgentStore) Save() error {
    s.mu.RLock()
    defer s.mu.RUnlock()
    sj := storeJSON{
        Agents:   s.agents,   // map reference, not a copy
        Findings: s.findings, // slice reference, not a copy
        // ...
    }
    data, err := json.MarshalIndent(sj, "", "  ")
```

**What:** `Save()` (exported) acquires `RLock` and serializes `s.agents` (a map) and `s.findings` (a slice) directly. Since `RLock` allows concurrent readers, if a write-holding goroutine is modifying `s.agents` concurrently (e.g., `UpdateAgentStatus`), the map serialization could read a partially-updated map. However, since writes hold `Lock` (which blocks `RLock`), this is actually safe in practice — the real concern is that `Save()` reads stale data if called immediately after a write that hasn't been flushed yet.

The more practical issue: `save()` (lowercase, called from mutating methods) is called while holding the write lock, which means the atomic file write (WriteFile + Rename) happens under the write lock. This is correct but means every mutation does disk I/O while holding the lock, which could block other goroutines if disk is slow.

**Fix:** No immediate fix needed, but consider debouncing the disk write (e.g., only write to disk every N seconds or on explicit request) to reduce lock contention.

---

## Finding 9 — cleanTransitItems goroutine never stops

**Severity:** LOW  
**File:** `pkg/scripting/engine.go:131-132, 145-157`  
**Category:** Resource leak

```go
go engine.cleanTransitItems()

func (e *Engine) cleanTransitItems() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        // ... cleanup ...
    }
}
```

**What:** The `cleanTransitItems` goroutine runs indefinitely with no shutdown mechanism. There's no `done` channel or context to signal termination. When the engine is closed, this goroutine leaks.

**Why it matters:** Minor resource leak. In a long-running MUD server that only shuts down on process kill, this is negligible. But if the engine is ever recreated (e.g., during hot-reload), old goroutines accumulate.

**Fix:** Add a `done` channel to the Engine struct:

```go
type Engine struct {
    // ... existing fields ...
    done chan struct{}
}
```

And select on it in the cleanup loop:

```go
for {
    select {
    case <-ticker.C:
        // cleanup
    case <-e.done:
        return
    }
}
```

---

## Finding 10 — luaDofile uses e.l instead of L parameter

**Severity:** LOW  
**File:** `pkg/scripting/engine.go:1671`  
**Category:** Logic bug (latent)

```go
func (e *Engine) luaDofile(L *lua.LState) int {
    // ... path validation ...
    if err := e.l.DoFile(fullPath); err != nil {  // ← uses e.l, not L
```

**What:** The `luaDofile` function receives `L *lua.LState` as its parameter (the Lua state from the PCall context), but calls `e.l.DoFile(fullPath)` using the engine's stored state instead. In practice, `L == e.l` because gopher-lua passes the same LState to registered functions. However, this creates a latent bug: if the engine ever uses multiple LStates (e.g., for concurrent script execution), `luaDofile` would execute code on the wrong state.

**Why it matters:** Currently harmless, but violates the principle of least surprise and makes the code harder to reason about during future refactoring.

**Fix:** Replace `e.l.DoFile(fullPath)` with `L.DoFile(fullPath)`.

---

## Summary

| Severity | Count | Findings |
|----------|-------|----------|
| HIGH     | 2     | #1 (RateLimitMiddleware race), #2 (transitItems vnum collision) |
| MEDIUM   | 4     | #3 (luaCanSee stack), #4 (LState accessor), #5 (dreaming path traversal), #6 (login brute-force), #7 (role by level) |
| LOW      | 3     | #8 (AgentStore race), #9 (goroutine leak), #10 (e.l vs L) |

### Positive Notes

- **Lua sandboxing is thorough.** `newSafeLState()` removes all dangerous globals (io, os, debug, package, dofile/loadfile/load/loadstring) and re-registers only the needed functions. The `dofile` re-registration after nil is intentional and documented.
- **Path traversal in luaDofile is properly handled** (filepath.Clean + filepath.Rel + prefix check). This is the correct pattern.
- **JWT validation is solid.** HMAC signing method is validated, secret minimum length enforced, claims carry role hierarchy.
- **Admin API has per-IP rate limiting** via `auth.NewIPRateLimiter()` with proper IP extraction (X-Forwarded-For only trusted from configured proxies).
- **Admin endpoints require both authentication AND role checks** (`requireRole` wrapper), and write operations are audit-logged.
- **Script execution has a 5-second timeout** with LState recreation on panic — prevents Lua infinite loops from hanging the server.
- **Input validation on admin updates** checks string length and control characters via `validateStringField`.
