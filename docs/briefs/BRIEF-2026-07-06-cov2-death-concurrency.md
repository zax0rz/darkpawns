# BRIEF — COV-2: Death-path concurrency tests (DP-963)

**Linear:** DP-963 (COV-2: death-path concurrency tests — assert idempotent kills under -race)
**Effort:** S
**Agent:** Reek (DeepSeek)
**Source of truth:** docs/reports/REVIEW-2026-07-05-full-audit.md — §3C item 2

## Goal

Write `-race`-clean concurrency tests that verify `handlePlayerDeath` and `handleMobDeath` are idempotent when two goroutines deliver lethal damage simultaneously. Locks in the F2 fix (DP-943) — the `dying` atomic CAS guard.

## Background

F2 (DP-943) added a `dying` atomic bool CAS guard to `handlePlayerDeath` (death.go:373):
```go
if !player.dying.CompareAndSwap(false, true) {
    return  // second concurrent caller no-ops
}
defer player.dying.Store(false)
```

This prevents double death processing (double EXP loss, double corpse, double CON loss). But we have no test that exercises this race condition. The fix could regress silently — only the narrow timing window protects us.

## Fix

### Test 1: `TestHandlePlayerDeath_ConcurrentKills` (pkg/game/death_test.go)

Create a World with a room, a mob, and a player at low HP (say 5 HP, level 10). Launch two goroutines that simultaneously call `w.HandleDeath(player, mob, attackType)` (or set HP to 0 and trigger the death path — whichever is more natural through the public API).

Assert:
1. Exactly one corpse created in the room (check room items for a corpse object)
2. Player EXP decreased exactly once (compare before/after)
3. Player respawned (HP > 0, in recall room)
4. No `-race` violations when run with `go test -race`

Implementation approach:
```go
func TestHandlePlayerDeath_ConcurrentKills(t *testing.T) {
    // Setup: world with room, mob, player at 5 HP
    // Record initial EXP

    var wg sync.WaitGroup
    wg.Add(2)

    // Goroutine 1: deliver lethal damage
    go func() {
        defer wg.Done()
        player.SetHP(0) // or deal damage > 5
        // trigger death path
    }()

    // Goroutine 2: deliver lethal damage simultaneously
    go func() {
        defer wg.Done()
        time.Sleep(time.Nanosecond) // tiny offset to increase collision probability
        player.SetHP(0)
        // trigger death path
    }()

    wg.Wait()

    // Assert single corpse, single EXP loss, respawn
}
```

**Important:** The exact mechanism to trigger death depends on the public API. Look at how `HandleDeath` is called — it's the main entry point. You may need to set HP to 0 and then call `HandleDeath` or the combat engine's damage function. Check what works with the test harness (no CombatEngine? mock one).

### Test 2: `TestHandleMobDeath_ConcurrentKills`

Same pattern but for mob death. Two goroutines deliver lethal damage to a mob simultaneously. Assert:
1. Exactly one corpse in the room
2. Killer gets EXP reward exactly once
3. No `-race` violations

### Test 3: `TestHandlePlayerDeath_SecondKillNoOps`

Simpler than the concurrent test — call `handlePlayerDeath` (or equivalent) twice in sequence on the same player. The second call should be a no-op. This tests the CAS guard directly without needing goroutine timing.

## Files

| File | Change |
|---|---|
| `pkg/game/death_test.go` | Add 2-3 death concurrency/idempotency tests |

## Existing Test Infrastructure

Check these files for patterns:
- `pkg/game/mobact_test.go` — shows how to create test World, MobInstance, Player
- `pkg/game/spec_procs_test.go` — shows mock CombatEngine pattern
- `pkg/game/combat_golden_test.go` — shows combat test setup

## Build Gate

```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

**Critical:** These tests MUST pass with `-race`. That's the whole point.

## Constraints

1. **Must use `go test -race`**. The race detector is the primary validation tool here.
2. **Do NOT add sleeps longer than 1ms** to force collision. Use `sync.WaitGroup` + `Gosched()` if needed. The CAS guard is deterministic, not timing-dependent — the second caller always finds `dying == true`.
3. **Follow existing test patterns** in pkg/game/ for World/Player/MobInstance creation.
4. If `HandleDeath` requires a non-nil CombatEngine, use a mock (see `spec_procs_test.go` for the pattern).
5. Single PR.

## C Fidelity

C's death path had the same double-death problem — the `die()` function was not called under a lock. The Go CAS guard is an improvement over C. The test validates the improvement.
