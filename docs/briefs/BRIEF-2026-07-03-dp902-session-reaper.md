# Brief: Session Reaper + Linkdead Cleanup — 2026-07-03

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Context

DP-902 is a CRITICAL infra issue from the Fable Review. Sessions wedge permanently — when a client dies without sending `quit`, the server never detects the disconnect. The ghost player remains in the room, blocks tunnel rooms (combined with DP-896 tunnel fix), and accumulates over uptime.

**The Architect classified this as "infra, not fidelity."** It needs a proper infrastructure brief, not a quick patch.

---

## Fix 1: DP-902 — Session disconnect detection (CRITICAL)

**Files:**
- `pkg/session/session_pump.go` — `readPump()` (line 14), `writePump()` (line 74)
- `pkg/session/manager.go` — `Unregister()` (line 720), `cleanupSession()` (line 642)

**Problem:**
When a WebSocket client dies (network drop, process kill, tab close), the server's `readPump` eventually gets an error from `conn.ReadMessage()` and breaks out of the loop. The deferred `Unregister` in `readPump` runs. **However**, if the write side is what fails first (e.g., client is gone, server tries to write a tick message), `writePump` gets a write error and returns — but the deferred cleanup in `writePump` only does `s.Close()`, it does NOT call `Unregister`. So the session stays in the manager's map.

Additionally, there's a race: `readPump` calls `Unregister(s.playerName)` in its defer, but `s.playerName` might be empty if the session died during the login phase. The `Unregister` becomes a no-op and the session leaks.

**Root cause chain:**
1. Client dies → TCP connection drops
2. Server's `writePump` tries to send a tick message → `conn.WriteMessage` fails → `writePump` returns
3. `writePump` deferred cleanup only calls `s.Close()` — does NOT call `Unregister`
4. `readPump` may still be blocked on `conn.ReadMessage()` (TCP read can linger)
5. Eventually `readPump` gets an error → deferred `Unregister` runs → but by now the session may have been replaced by a reconnection
6. Ghost player stays in the world

**Fix:**

A. In `writePump`, add `Unregister` to the deferred cleanup (matching `readPump`):

```go
func (s *Session) writePump() {
    ticker := time.NewTicker(54 * time.Second)
    defer func() {
        if r := recover(); r != nil {
            slog.Error(
                "CRITICAL PANIC RECOVERED in writePump",
                "player", s.playerName,
                "recover", r,
                "stack", string(debug.Stack()),
            )
        }
        ticker.Stop()
        s.Close()
        // NEW: ensure session is cleaned up if writePump exits first
        s.manager.Unregister(s.playerName)
    }()
    // ... rest unchanged
}
```

B. Add a `lastActive` timestamp to Session, updated on every inbound message in `readPump`:

```go
// In Session struct (session_types.go or similar):
type Session struct {
    // ... existing fields ...
    lastActive atomic.Int64 // unix nano, updated on each message
}

// In readPump, after successful ReadMessage:
s.lastActive.Store(time.Now().UnixNano())
```

C. Add a linkdead reaper to the Manager — a method called from the game tick:

```go
// In manager.go:

const (
    linkdeadThreshold    = 5 * time.Minute  // move to void
    linkdeadExtract      = 15 * time.Minute // extract from world
)

// ReapLinkdeadSessions checks for sessions that haven't sent a message
// in threshold time. Called periodically from the game tick.
func (m *Manager) ReapLinkdeadSessions() {
    m.mu.RLock()
    var stale []*Session
    now := time.Now().UnixNano()
    for _, s := range m.sessions {
        if !s.authenticated || s.player == nil {
            continue
        }
        last := s.lastActive.Load()
        if last == 0 {
            continue // never received a message (shouldn't happen post-auth)
        }
        elapsed := time.Duration(now - last)
        if elapsed > linkdeadThreshold {
            stale = append(stale, s)
        }
    }
    m.mu.RUnlock()

    for _, s := range stale {
        elapsed := time.Duration(time.Now().UnixNano() - s.lastActive.Load())
        slog.Warn("reaping linkdead session",
            "player", s.playerName,
            "idle", elapsed.Round(time.Second),
        )
        // Close the WebSocket — this triggers readPump/writePump exit → Unregister
        if s.conn != nil {
            _ = s.conn.Close()
        }
    }
}
```

D. Wire the reaper into the game tick in `pkg/game/world.go` or wherever the heartbeat fires:

```go
// In the tick/heartbeat callback:
w.manager.ReapLinkdeadSessions()
```

**Why conn.Close() instead of direct Unregister:** Closing the WebSocket triggers the deferred cleanup in readPump/writePump, which handles all the teardown (stop combat, broadcast leave, save player, remove from world). Direct Unregister from the reaper would race with the pump goroutines. Let the pumps exit cleanly.

**Regression Test:** `pkg/session/reaper_test.go`
- `TestReapLinkdeadSession`: create session, set lastActive to 10 min ago, call ReapLinkdeadSessions, assert conn was closed and session was cleaned up
- `TestReapDoesNotKillActiveSession`: create session with recent lastActive, call ReapLinkdeadSessions, assert session is still alive
- `TestWritePumpExitTriggersCleanup`: create session, close conn externally, wait for writePump to exit, assert session is unregistered

---

## Fix 2: Login-screen idle timeout (part of DP-912, already Done)

DP-912 was already closed with pre-auth idle timeout. Verify it's actually working — the `CheckIdlePasswords` method exists but needs confirmation it's wired into the tick. This is a verification step, not new code.

---

## Execution Order

1. Fix 1A: writePump deferred Unregister (one line)
2. Fix 1B: lastActive timestamp on Session (3 lines in struct, 1 line in readPump)
3. Fix 1C: ReapLinkdeadSessions method (~30 lines)
4. Fix 1D: Wire reaper into game tick (1 line)
5. Regression tests

---

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
git add pkg/session/session_pump.go pkg/session/manager.go pkg/session/session_types.go pkg/session/reaper_test.go
git commit -m "fix: session reaper + linkdead cleanup (DP-902)"
git push -u origin fix/dp-902-session-reaper
gh pr create --title "fix: session reaper + linkdead cleanup (DP-902)" --body "Fixes DP-902. See docs/briefs/BRIEF-2026-07-03-dp902-session-reaper.md for details."
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-902: Add comment "Fixed — session reaper + writePump cleanup, commit <hash>", move to Done
