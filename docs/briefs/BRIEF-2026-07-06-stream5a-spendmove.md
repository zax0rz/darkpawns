# BRIEF — Stream 5: SpendMove atomic deduction (F12)

**Linear:** DP-960 (F12 — DoBash move-point check-then-act)
**Effort:** S
**Agent:** Reek (DeepSeek)
**Source of truth:** docs/reports/REVIEW-2026-07-05-full-audit.md — F12

## Goal

Add `Player.SpendMove(n int) bool` that atomically checks and deducts move points under one lock. Replace all 6 check-then-act sites.

## Problem

`GetMove()` and `SetMove()` each acquire/release their own `sync.RWMutex`. The pattern:

```go
if ch.GetMove() < 10 {          // lock acquired, read, lock released
    return false
}
ch.SetMove(ch.GetMove() - 10)    // lock acquired, read, lock released, lock acquired, write, lock released
```

A regen tick between the two calls overwrites the deduction. Cosmetic-scale in practice (move points regen slowly), but not correct.

## Affected Sites (6 total)

| # | File | Lines | Cost | Context |
|---|------|-------|------|---------|
| 1 | `pkg/game/skill_combat.go` | 144-147 | 10 | DoBash |
| 2 | `pkg/game/skill_c10_combat.go` | 61-64 | 10 | DoDragonKick |
| 3 | `pkg/game/skill_c10_combat.go` | 244-247 | 51 | DoNeckbreak |
| 4 | `pkg/game/act_movement.go` | 249, 276 | variable | performMove (room movement) |
| 5 | `pkg/game/world.go` | 931, 935 | variable | World.movePlayer (following tick) |
| 6 | `pkg/game/spec_procs2.go` | 853, 861 | 25% of current | specJail |

## Fix

### 1. Add SpendMove to Player

In `pkg/game/player_stats.go`:

```go
// SpendMove atomically checks and deducts move points.
// Returns true if deduction succeeded, false if insufficient move points.
func (p *Player) SpendMove(n int) bool {
    p.mu.Lock()
    defer p.mu.Unlock()
    if p.Move < n {
        return false
    }
    p.Move -= n
    return true
}
```

### 2. Replace all 6 sites

For each site, replace the two-line check+deduct with a single `SpendMove()` call. Example for DoBash:

```go
// Before:
if ch.GetMove() < 10 {
    return SkillResult{Success: false, MessageToCh: "You haven't the energy!"}
}
ch.SetMove(ch.GetMove() - 10)

// After:
if !ch.SpendMove(10) {
    return SkillResult{Success: false, MessageToCh: "You haven't the energy!"}
}
```

For `performMove` and `World.movePlayer`, the cost is variable (movement cost depends on terrain). Use the existing cost variable.

For `specJail`, the cost is `vict.GetMove() / 4`. Add a `SpendMoveFraction(denom int) bool` or compute the cost first:

```go
cost := vict.GetMove() / 4
if cost < 1 {
    cost = 1
}
if !vict.SpendMove(cost) {
    return false
}
```

### 3. Tests

- `TestSpendMove_Sufficient` — 20 move, spend 10 → true, remaining 10
- `TestSpendMove_Insufficient` — 5 move, spend 10 → false, remaining 5
- `TestSpendMove_Exact` — 10 move, spend 10 → true, remaining 0
- `TestSpendMove_Zero` — 5 move, spend 0 → true, remaining 5
- `TestSpendMove_Negative` — 5 move, spend -1 → true (treat negative as no-op or error)

## Files

| File | Change |
|---|---|
| `pkg/game/player_stats.go` | Add `SpendMove()` method |
| `pkg/game/skill_combat.go` | Replace check-then-act at lines 144-147 |
| `pkg/game/skill_c10_combat.go` | Replace at lines 61-64 and 244-247 |
| `pkg/game/act_movement.go` | Replace at lines 249, 276 |
| `pkg/game/world.go` | Replace at lines 931, 935 |
| `pkg/game/spec_procs2.go` | Replace at lines 853, 861 |
| `pkg/game/player_stats_test.go` | Add SpendMove tests |

## Build Gate

```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

## Constraints

1. Do NOT modify `GetMove()` or `SetMove()` — they have other callers that use them independently.
2. Do NOT add SpendMana/SpendHealth — out of scope. Only move points.
3. Do NOT change behavior — SpendMove must return the same result as the current check+deduct pattern for all valid inputs.
4. Single PR.

## C Fidelity

C's `GET_MOVE(ch)` reads `ch->move` directly (no locking — single-threaded). The Go port added locking for goroutine safety. This fix makes the Go version correct under concurrency. No C behavior change.
