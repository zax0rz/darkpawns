# Brief: Stream 4a — MinLevel Enforcement at Dispatch (F10)

## Context

DP-954 from the Fable Audit 2026-07-05. The command registry's `MinLevel` field
is dead code — populated during registration but never enforced at dispatch. All
49 wizard handlers compensate by calling `checkLevel()` at the top of their
function. The next wizard command added without that boilerplate is a silent
mortal escalation.

**Linear:** DP-954 (F10)
**Branch:** `fix/stream4a-minlevel`
**Agent:** Kimi

## Problem

`pkg/session/commands.go:592-635` — the dispatch function enforces
`MinPosition` (line 603) and `WaitState` (line 612) but **never checks**
`entry.MinLevel`. The field is populated on 40+ wizard commands
(`LVL_IMMORT`, `LVL_GOD`, etc.) and used in help output filtering, but
has no security function.

Meanwhile, 49 wizard handlers across 9 files all open with:
```go
if !checkLevel(s, LVL_IMMORT) {
    sendToChar(s, "Huh?!?\r\n")
    return
}
```

**Secondary gap:** `pkg/session/session_command.go:16` — the
`Manager.RegisterCommand` bridge hardcodes `MinLevel=0` for all commands
registered through that path (admin_commands.go, shop_commands.go). Those
commands rely entirely on per-handler gating.

## Fix

### 1. Add MinLevel gate at dispatch

In `pkg/session/commands.go`, after the MinPosition gate (line 610) and
before the WaitState gate (line 612), add:

```go
// MinLevel enforcement — F10/DP-954
if entry.MinLevel > 0 {
    if getEffectiveLevel(s) >= entry.MinLevel {
        // qualified — proceed
    } else {
        sendToChar(s, "Huh?!?\r\n")
        return
    }
}
```

Use `getEffectiveLevel(s)` (from `wizard_cmds.go:20`) rather than
`s.player.GetLevel()` so the gate respects switched-body wizards and
forced-command privilege levels, same as the per-handler checks.

**Gate position:** After MinPosition but before WaitState. A sleeping player
should see the position error first, not the level error. The MinLevel check
is cheap so ordering doesn't matter much for performance.

### 2. Wire MinLevel on Manager.RegisterCommand commands

`pkg/session/session_command.go:16` currently hardcodes `MinLevel=0`:
```go
cmdRegistry.Register(name, command.Handler(wrapped), name+" (registered via RegisterCommand)", 0, 0)
```

Change the `RegisterCommand` signature to accept `minLevel int`:
```go
func (m *Manager) RegisterCommand(name string, handler func(common.CommandSession, []string) error, minLevel int)
```

Update all callers:
- `pkg/command/admin_commands.go` — pass `LVL_IMMORT` (31) for report, warn, mute, kick, ban
- `pkg/command/shop_commands.go` — pass `0` for list, buy, sell (player commands)
- Any other callers (grep for `RegisterCommand`)

### 3. Keep per-handler checkLevel as defense in depth

Do NOT remove the 49 `checkLevel` calls from wizard handlers. They serve two
purposes the dispatch gate doesn't fully cover:
- Switched-body wizards (`switchedOriginalLevel`)
- Forced-command privilege levels (`ForcedPrivilegeLevel`)

The dispatch gate catches the common case (mortal trying wizard commands).
The per-handler checks are safety net for edge cases. Both layers are
valuable.

## Files

- **`pkg/session/commands.go:592-635`** — add MinLevel gate (MODIFY)
- **`pkg/session/session_command.go:16`** — add minLevel param (MODIFY)
- **`pkg/command/admin_commands.go`** — wire LVL_IMMORT on RegisterCommand calls (MODIFY)
- **`pkg/command/shop_commands.go`** — wire 0 on RegisterCommand calls (MODIFY)
- **`pkg/session/wizard_cmds.go:9-14`** — LVL_* constants (READ, may need to export)

## Regression Test

```go
func TestMinLevelBlocksMortal(t *testing.T) {
    // Register a command with MinLevel=LVL_IMMORT
    // Create a level-1 player session
    // Execute the command
    // Assert: handler was NOT called, player got "Huh?!?"
}

func TestMinLevelAllowsImmortal(t *testing.T) {
    // Same command, level-31 player
    // Execute the command
    // Assert: handler WAS called
}

func TestMinLevelZeroAllowsAll(t *testing.T) {
    // Register a command with MinLevel=0 (player command)
    // Level-1 player executes it
    // Assert: handler was called
}
```

## Build Gate

```bash
go build ./... && go vet ./... && go test ./... && gofumpt -l . | grep -v vendor
```

## Commit

```
fix: enforce MinLevel at command dispatch — registry field is no longer dead code (DP-954)
```
