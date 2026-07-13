# Engineering Brief 02: Stability Audit — Panic Recovery + Race Conditions + Graceful Shutdown

**Date:** 2026-05-27
**Priority:** CRITICAL — data loss and crashes
**Scope:** `pkg/session/`, `pkg/game/`, `pkg/combat/`, `cmd/server/`

---

## Problem

Three stability risks in one:

1. **Panics in session handlers crash the server** — a single player's bad input or unexpected state can bring down the entire MUD
2. **Race conditions** — concurrent session access to shared world state (rooms, mobs, objects) can corrupt data
3. **No graceful shutdown** — SIGTERM kills the process, players lose unsaved state

## What to Do

### 1. Panic Recovery on Every Handler

Currently only `completeCharCreation` has panic recovery. Every handler that touches player state needs it:

```go
func (s *Session) handlePanic(handlerName string) {
    if r := recover(); r != nil {
        slog.Error("PANIC recovered",
            "handler", handlerName,
            "player", s.playerName,
            "recover", r,
            "stack", string(debug.Stack()),
        )
        // Send error to player if still connected
        s.sendError("An internal error occurred. Your session is being reset.")
        // Clean up session state
        s.manager.Unregister(s.playerName)
    }
}
```

Wrap these handlers:
- `handleLogin`
- `handleCharInput`
- `handleCommand`
- `handleCharInput` (each stage)
- `sendWelcome`
- `completeCharCreation` (already has it, verify it's complete)

### 2. Race Condition Audit

Run `go test -race ./pkg/session/ ./pkg/game/ ./pkg/combat/ ./pkg/world/` and fix every reported race.

Common suspects:
- `World.players` map — accessed by multiple session goroutines
- `World.mobs` — mob state changes during combat
- `Player` struct fields — modified by commands, combat, and scripts concurrently
- `Session.send` channel — concurrent writes from combat events and command responses

If races are found, document them with the exact call stack and proposed fix (mutex, channel, or atomic).

### 3. Graceful Shutdown

In `cmd/server/main.go` or equivalent:

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

sig := <-sigChan
slog.Info("shutdown signal received", "signal", sig)

// 1. Stop accepting new connections
listener.Close()

// 2. Save all active players
for _, session := range manager.GetAllSessions() {
    if session.player != nil {
        db.SavePlayer(session.player)
        session.sendError("Server is shutting down. Your character has been saved.")
        session.conn.Close()
    }
}

// 3. Flush logs
slog.Info("shutdown complete", "players_saved", count)
```

### 4. Add Health Check Endpoint

Expose `/health` (or use existing) that reports:
- Server uptime
- Active connection count
- Memory usage
- Last zone reset time
- Any active panics (if using panic recovery tracking)

## Verification

1. `go test -race ./pkg/session/ ./pkg/game/ ./pkg/combat/ ./pkg/world/` — zero races
2. Send malformed WebSocket message — server doesn't crash
3. Kill server with SIGTERM — all players saved, clean exit
4. `curl /health` returns structured JSON with server status
5. Run 100 concurrent connections doing random actions — no panics, no data corruption

## Files to Modify

- `pkg/session/session_login.go` — add panic recovery wrapper
- `pkg/session/char_creation.go` — add panic recovery wrapper
- `pkg/session/session_send.go` — add panic recovery wrapper
- `pkg/session/manager.go` — GetAllSessions, session cleanup
- `pkg/game/world.go` — race-prone shared state
- `cmd/server/main.go` — signal handling, graceful shutdown
- Various `pkg/game/*.go` — race fixes as found
