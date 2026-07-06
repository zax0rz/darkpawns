# BRIEF — COV-3: Shutdown drain test (DP-964)

**Linear:** DP-964 (COV-3: shutdown drain test — assert no ticker mutates world during SaveWorld)
**Effort:** S
**Agent:** Reek (DeepSeek)
**Source of truth:** docs/reports/REVIEW-2026-07-05-full-audit.md — §3C item 3

## Goal

Write a test that boots a World with all ticker families running, triggers the shutdown sequence, and asserts no goroutine mutates world state after `SaveWorld` begins. Locks in F3/F4 fixes.

## Background

F3 (DP-944) rewrote the shutdown sequence: `StopAITicker()` + `StopPeriodicResets()` → `telnet.Stop()` → `srv.Shutdown()` → `manager.ShutdownGracefully()` → `SaveWorld()`. F4 removed the `ZoneDispatcher` and consolidated into a single 30s point ticker.

The current shutdown order in `cmd/server/main.go:420-455` is:
1. `gameLoop.Stop()` — stop the main game loop
2. `gameWorld.StopAITicker()` — stop AI + point tickers (closes `done` channel)
3. `gameWorld.StopPeriodicResets()` — stop zone reset goroutine
4. `telnet.Stop()` — stop accepting telnet connections
5. `srv.Shutdown()` — drain HTTP/WebSocket connections
6. `manager.ShutdownGracefully(5s)` — drain player sessions
7. `game.SaveWorld(gameWorld)` — persist state

We have no test that verifies step 2-3 actually stop mutations before step 7 runs.

## Fix

### Test: `TestShutdown_NoMutationAfterSaveBegins` (pkg/game/world_test.go or new file)

**Approach:** Add a mutation counter to World that increments on every state-mutating operation (mob movement, point tick, zone reset). Boot the world with tickers running, then simulate the shutdown sequence. After calling `StopAITicker` + `StopPeriodicResets`, wait briefly, then check that the counter has stopped incrementing.

```go
func TestShutdown_NoMutationAfterSaveBegins(t *testing.T) {
    // 1. Create a world with rooms and mobs
    // 2. Add a mutation counter: atomic int64 on World
    //    (or use a channel that receives on each mutation)
    // 3. Start AI ticker (which calls mobileActivity → wander → SetRoom → mutation)
    // 4. Let it run for 200ms (accumulate mutations)
    // 5. Call StopAITicker() + StopPeriodicResets()
    // 6. Wait 200ms
    // 7. Assert counter has not changed since step 6
}
```

**Simpler alternative (no World modification):** Instead of a counter, use a channel-based approach:
1. Start a goroutine that reads `w.GetMobsInRoom(someRoom)` every 10ms into a result channel
2. After shutdown, assert the goroutine has stopped (close a done channel)

But the counter approach is cleaner. If adding a field to World is too invasive for an S-effort PR, use a different strategy:

**Simplest approach (no World modification):**
1. Boot world with wandering mobs
2. Record all mob room VNums
3. Start AI ticker
4. Sleep 200ms (mobs should wander)
5. Record all mob room VNums again — verify at least one moved
6. Call `StopAITicker()`
7. Take a snapshot of all mob rooms
8. Sleep 200ms
9. Take another snapshot — verify NO mob rooms changed
10. Assert `w.done` channel is closed (ticker stopped)

This tests the shutdown ordering without any World changes. It verifies that `StopAITicker()` actually stops the ticker goroutine from calling `mobileActivity()`.

### Test: `TestStopPeriodicResets_StopsZoneResets`

Simpler test — verify `StopPeriodicResets()` actually stops the zone reset goroutine.

```go
func TestStopPeriodicResets_StopsZoneResets(t *testing.T) {
    w := setupWorld(t)
    w.StartPeriodicResets() // if this is the public API
    time.Sleep(100ms)
    w.StopPeriodicResets()
    // If we can verify the goroutine stopped, assert it
}
```

Check what `StopPeriodicResets()` does — it may just close a channel. Verify the goroutine exits.

## Files

| File | Change |
|---|---|
| `pkg/game/world_test.go` or new `pkg/game/shutdown_test.go` | Add shutdown drain tests |

## Key References

- Shutdown sequence: `cmd/server/main.go:420-455`
- `StopAITicker`: `pkg/game/world.go:996-1005` — closes `w.done` channel
- `StopPeriodicResets`: find the implementation, check what it closes
- AI ticker loop: find the `for { select { case <-w.done: return; case <-ticker: mobileActivity() } }` pattern
- Point ticker: same loop pattern
- Zone reset goroutine: find `StartPeriodicResets` / zone reset loop

## Build Gate

```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

## Constraints

1. **Do NOT add fields to World for the counter** if possible. Use the snapshot approach instead. If a counter is truly needed, make it `MutationCount atomic.Int64` and only use it in tests (or behind a build tag).
2. **Do NOT change the shutdown sequence** — we're testing the existing sequence, not modifying it.
3. **Use `time.Sleep` sparingly** — 100-200ms is fine for verifying ticker shutdown. Don't use seconds-long sleeps.
4. Follow existing test patterns in `pkg/game/world_test.go`.
5. Single PR.

## C Fidelity

C's shutdown (`bones()`) was simpler — it called `close_socket()` on all descriptors and `save_world()`. No tickers to drain because the MUD was single-threaded. The Go version is more complex (multiple goroutines) and needs this test to prevent regressions.
