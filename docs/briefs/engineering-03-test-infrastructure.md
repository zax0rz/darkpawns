# Engineering Brief 03: Test Infrastructure — Fixtures, Patterns, Coverage Baseline

**Date:** 2026-05-27
**Priority:** HIGH — safety net for all future changes
**Scope:** `tests/`, `pkg/*/`, all test files

---

## Problem

Tests exist but are thin and inconsistent. 7 packages have zero test files. Each test reinvents setup. No reusable test utilities. When the fidelity audits touched 254 files, there was no safety net to catch regressions.

## What to Do

### 1. Create `pkg/testutil/` Package

Reusable test builders:

```go
// pkg/testutil/helpers.go

func NewTestPlayer(name string, class, race int) *game.Player {
    p := game.NewCharacter(0, name, class, race)
    p.Health = 100
    p.MaxHealth = 100
    p.Mana = 20
    p.MaxMana = 20
    p.Move = 100
    p.MaxMove = 100
    p.Level = 1
    return p
}

func NewTestRoom(vnum int, name string) *world.Room {
    return &world.Room{
        VNum:        vnum,
        Name:        name,
        Description: "A test room.",
        Exits:       make(map[string]*world.Exit),
    }
}

func NewTestWorld() *world.World {
    // Minimal world with a few rooms for testing
}

func NewTestSession(player *game.Player, manager *session.Manager) *session.Session {
    // Session wired to a mock WebSocket
}
```

### 2. Add Tests for Packages with Zero Coverage

| Package | Current Tests | Priority | What to Test |
|---------|---------------|----------|--------------|
| `pkg/telnet/` | 0 | HIGH | Connection handling, protocol negotiation |
| `pkg/web/` | 0 | HIGH | HTTP routes, WebSocket upgrade |
| `pkg/optimization/` | 0 | MEDIUM | Database pool, connection management |
| `pkg/storage/` | 0 | MEDIUM | Save/load player data |
| `pkg/admin/` | 0 | MEDIUM | Admin API endpoints |
| `pkg/secrets/` | 0 | LOW | Secret loading, env vars |
| `pkg/profiling/` | 0 | LOW | Profiler start/stop |

### 3. Expand Existing Test Coverage

For packages that have tests, add edge cases:

| Package | Current | Target | Add |
|---------|---------|--------|-----|
| `pkg/session/` | 4 tests | 15+ | Error paths, race conditions, edge cases |
| `pkg/game/` | 3 tests | 10+ | Combat formulas, movement, object interaction |
| `pkg/combat/` | 3 tests | 10+ | Damage calculation, skill checks, death |
| `pkg/spells/` | 3 tests | 10+ | Spell effects, saves, duration |

### 4. Add Integration Test Patterns

Create a `tests/integration/` directory with end-to-end scenarios:

```go
// tests/integration/char_creation_test.go
func TestFullCharCreationFlow(t *testing.T) {
    // Connect → name → confirm → password → retype → color → sex → race → class → hometown → stats → motd → enter world
}

// tests/integration/combat_test.go
func TestBasicCombatFlow(t *testing.T) {
    // Spawn player + mob → attack → verify damage → verify death → verify respawn
}

// tests/integration/group_test.go
func TestGroupFormationAndCombat(t *testing.T) {
    // Two players → form group → fight mob → verify exp split
}
```

### 5. Test Naming Convention

Standardize on:
```
TestUnit_Condition_ExpectedBehavior
```

Examples:
```
TestHandleLogin_InvalidPassword_SendsErrorAndCloses
TestHandleCharInput_BadRace_DisplaysErrorAndReprompts
TestCmdMove_ClosedDoor_SendsBlockedMessage
TestDamage_CriticalHit_DoublesDamage
```

## Verification

1. `go test ./pkg/testutil/...` — testutil package compiles and tests pass
2. `go test ./...` — all existing tests still pass
3. `go test -cover ./pkg/session/` — coverage > 60%
4. `go test -cover ./pkg/game/` — coverage > 40%
5. `go test -cover ./pkg/combat/` — coverage > 50%
6. `go test ./tests/integration/...` — integration tests pass
7. New test files follow naming convention

## Files to Create

- `pkg/testutil/helpers.go` — test builders
- `pkg/testutil/helpers_test.go` — verify builders work
- `pkg/telnet/*_test.go` — telnet tests
- `pkg/web/*_test.go` — web tests
- `pkg/optimization/*_test.go` — optimization tests
- `pkg/storage/*_test.go` — storage tests
- `pkg/admin/*_test.go` — admin tests
- `tests/integration/char_creation_test.go` — end-to-end creation
- `tests/integration/combat_test.go` — end-to-end combat
- `tests/integration/group_test.go` — end-to-end grouping
