# Brief: Stream 4b — Combat Hook Cleanup (F8)

## Context

DP-952 from the Fable Audit 2026-07-05 (originally C-02 from April review).
57 package-level function hooks in `pkg/combat/fight_core.go:14-80`. The audit
called for migrating them to a `GameCallbacks` struct, but triage reveals most
are dead code — the engine path (struct callbacks on `CombatEngine`) is the
live path. Only 6 hooks are wired; 51 are never set and silently no-op.

**Linear:** DP-952 (F8)
**Branch:** `fix/stream4b-combat-hooks`
**Agent:** Kimi

## Problem

57 package-level `var` hooks in `fight_core.go`. Characteristics:
- **Zero synchronization** — write at boot, read at runtime, no mutex/atomic
- **1 hook lacks a nil guard** (`GetExp` at lines 565, 1088) — nil-panic risk
- **51 hooks are never wired** — all call sites silently no-op
- **6 are wired and functional** — but could be nil-panicked on partial init
- **5 are duplicated** on `CombatEngine` — engine path works, package-level is dead

### The 6 Wired Hooks

| Hook | Type | Wired In |
|---|---|---|
| `BroadcastMessage` | `func(roomVNum int, msg string, exclude string)` | `cmd/server/main.go` |
| `SendToCharFunc` | `func(name string, msg string)` | `cmd/server/main.go` |
| `DoFlee` | `func(name string)` | `pkg/session/commands.go` |
| `DoRetreat` | `func(name string)` | `pkg/session/commands.go` |
| `SkillMessageFunc` | `func(dam int, ch, vict string, attackType int, roomVNum int) bool` | `combat/skill_messages.go` |
| `NowUnix` | `func() time.Time` | `cmd/server/main.go` |

### The 1 Dangerous Hook (Missing Nil Guard)

`GetExp` — called at `fight_core.go:565` and `fight_core.go:1088` without
nil check. If `GetExp` is nil, any combat that triggers group gain or death
XP calculation will panic.

## Fix Strategy — Delete Dead Hooks, Guard Live Ones

Do NOT do a big-bang `GameCallbacks` struct migration. The engine already has
struct callbacks for the live combat path. The `fight_core.go` hooks serve the
skill/spell path (which also uses the engine now via Stream 3's F7 MessageFunc).
The right approach is surgical cleanup:

### 1. Delete the 51 dead hooks

Remove all package-level vars that are never wired. For each, verify:
- It has no assignment outside fight_core.go (search all of `pkg/`)
- All call sites already have nil guards (or add one before deleting)
- The function it would call is unreachable (no caller sets it)

Dead hooks to remove (partial list — verify each before deleting):
All hooks in `fight_core.go:14-80` EXCEPT: `BroadcastMessage`,
`SendToCharFunc`, `DoFlee`, `DoRetreat`, `SkillMessageFunc`, `NowUnix`.

When removing a hook, convert its call sites to no-ops or guarded calls.
Example: if `SomeDeadHook != nil { SomeDeadHook(args) }` — just remove the
entire block.

### 2. Add nil guards on the 6 live hooks

For each of the 6 wired hooks, audit all call sites and ensure they're guarded:
```go
if BroadcastMessage != nil {
    BroadcastMessage(roomVNum, msg, exclude)
}
```

The `GetExp` hook at lines 565 and 1088 is the priority — it currently has
NO nil guard. Add one or default to 0 if nil:
```go
expGain := 0
if GetExp != nil {
    expGain = GetExp(victimName)
}
```

### 3. Add construction-time validation

After wiring hooks in `cmd/server/main.go`, add a validation block:
```go
// Verify critical combat hooks are wired
if combat.BroadcastMessage == nil || combat.SendToCharFunc == nil {
    slog.Error("critical combat hook not wired — combat messages will be silent")
}
```

This doesn't block startup (game should still work) but makes partial init
visible in logs.

### 4. Document the remaining hooks

Add a comment block above the 6 remaining hooks explaining:
- Which path uses them (skill/spell via fight_core)
- Where they're wired
- Why they exist as package-level vars instead of engine struct fields
- That the engine path (CombatEngine callbacks) is the primary path

## Files

- **`pkg/combat/fight_core.go:14-80`** — delete dead hooks, guard live ones (MODIFY)
- **`pkg/combat/fight_core.go:565,1088`** — add GetExp nil guard (MODIFY)
- **`cmd/server/main.go`** — add construction-time validation (MODIFY)

## Regression Test

```go
func TestDeadHookNoopDoesNotPanic(t *testing.T) {
    // With no hooks wired (all nil), trigger a combat round
    // via the fight_core path. Verify no panic.
}

func TestGetExpNilGuard(t *testing.T) {
    // With GetExp = nil, call the function that calculates group gain.
    // Verify it returns 0 instead of panicking.
}
```

## Build Gate

```bash
go build ./... && go vet ./... && go test ./... && gofumpt -l . | grep -v vendor
```

## Commit

```
refactor: delete 51 dead combat hooks, add nil guards on live hooks (DP-952)
```
