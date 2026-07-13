# Engineering Brief 01: Structured Logging + Context Propagation

**Date:** 2026-05-27
**Priority:** HIGH — observability foundation
**Scope:** `pkg/session/`, `pkg/game/`, `pkg/combat/`, `pkg/spells/`

---

## Problem

Logging is inconsistent. Some functions use `slog.Info()`, others use `log.Printf()`, others silently swallow errors. When something breaks in production, there's no way to trace a request through the system.

## What to Do

### 1. Standardize on `slog`

Replace all `log.Printf` / `log.Println` / `fmt.Printf` debug output with `slog`. Use structured fields everywhere:

```go
// Before:
slog.Error("char creation failed", "error", err)

// After:
slog.Error("char creation failed",
    "player", s.playerName,
    "stage", s.charStage,
    "error", err,
)
```

Every log line from a session handler should include:
- `player` — player name (or "guest_XXXX")
- `session` — session ID or pointer
- `room` — current room vnum (if authenticated)
- `stage` — char creation stage (if applicable)

### 2. Add `context.Context` to Session Handlers

The following functions need a context parameter added:

| Function | File |
|----------|------|
| `handleLogin()` | session_login.go |
| `handleCharInput()` | char_creation.go |
| `handleCommand()` | session_login.go |
| `completeCharCreation()` | char_creation.go |
| `sendWelcome()` | session_send.go |

Context should carry:
- Player name (via `context.WithValue`)
- Session start time
- Request ID (for tracing)

This enables:
- Timeout on database calls
- Cancellation on disconnect
- Tracing across function calls

### 3. Remove Silent Error Swallowing

Find all instances of:
```go
if err != nil {
    slog.Error("...", "error", err)
    return nil  // <-- silent swallow
}
```

These should either:
- Return the error to the caller, OR
- Be documented why the error is intentionally ignored

### 4. Add Request Tracing

Add a `requestID` to each incoming WebSocket message. Pass it through context. Include it in all log lines for that request. This lets you grep for one request and see every function it touched.

## Verification

1. `grep -r "log.Printf\|log.Println\|fmt.Printf" pkg/` — should find zero instances after
2. Every session handler has a `context.Context` parameter
3. `go vet ./...` passes
4. `go test ./...` passes
5. Manual test: create character, verify log lines include player/stage/room fields

## Files to Modify

- `pkg/session/session_login.go` — handleLogin, handleCommand
- `pkg/session/char_creation.go` — handleCharInput, completeCharCreation
- `pkg/session/session_send.go` — sendWelcome, sendError
- `pkg/session/manager.go` — session creation/destruction
- `pkg/game/act_movement.go` — movement functions
- `pkg/combat/fight_core.go` — combat functions
