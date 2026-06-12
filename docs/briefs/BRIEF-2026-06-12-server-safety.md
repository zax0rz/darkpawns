# Brief: Server Safety Fixes — 2026-06-12

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-562 — Door race condition (HIGH)

**File:** `pkg/game/systems/door_manager.go` — all action methods (OpenDoor, CloseDoor, LockDoor, etc.)

**Problem:** Every action method follows the same pattern:
1. `GetDoor()` acquires RLock, looks up `*Door` pointer, releases lock, returns pointer
2. Method calls `door.Open()` / `door.Close()` / etc. on the returned pointer
3. No lock is held during the mutation

If two players try to open the same door concurrently, both see `door.Closed == true`, both set `door.Closed = false`. Or one opens while another locks — interleaved reads and writes on the same struct fields. Data race.

**Fix:** Hold the manager's lock for the entire operation. Change each action method to acquire the lock once and operate on the door while holding it:

```go
func (dm *DoorManager) OpenDoor(fromRoom int, direction string) (bool, string) {
    dm.mu.Lock()
    defer dm.mu.Unlock()

    key := dm.key(fromRoom, direction)
    door, ok := dm.doors[key]
    if !ok {
        return false, "There is no door there."
    }
    if !door.CanSee() {
        return false, "There is no door there."
    }
    return door.Open()
}
```

Apply the same pattern to: `CloseDoor`, `LockDoor`, `UnlockDoor`, `PickDoor`, `BashDoor`, `CanPass`, `GetDoorStatus`.

Also remove `GetDoor()` from being used in action methods — it's still useful for read-only queries, but action methods should do their own lookup under the write lock.

**Caveat:** `BashDoor` modifies `door.Hp` — this is a mutation that needs the write lock. Same for `Open`/`Close`/`Lock`/`Unlock`/`Pick` which all modify `door.Closed` or `door.Locked`.

**Regression Test:** `pkg/game/systems/door_manager_test.go`
- `TestConcurrentOpenClose`: launch 100 goroutines opening and closing the same door concurrently. Assert no race detected (`go test -race`), assert final state is consistent (either open or closed, not corrupted)
- `TestConcurrentBashAndLock`: one goroutine bashes while another locks. Assert no data race.

**Cite:** C source — `src/act.movement.c` and `src/act.other.c`. C was single-threaded (one global game loop), so no locking was needed. The Go port added concurrent sessions but didn't add door locking.

**Verification:** `go build ./... && go vet ./... && go test -race ./pkg/game/systems/...`

---

## Fix 2: DP-566 — os.Exit in server goroutine bypasses graceful shutdown (HIGH)

**File:** `cmd/server/main.go` — lines 343, 349

**Problem:** The HTTP server runs in a goroutine. If `ListenAndServe` fails (line 343) or `ListenAndServeTLS` fails (line 330), it calls `os.Exit(1)`. This kills the process immediately without running deferred functions, without saving player data, without flushing logs.

The signal handler (SIGINT/SIGTERM) correctly calls `srv.Shutdown()` and `gameLoop.Stop()` for graceful shutdown. But the error path in the server goroutine bypasses all of that.

```go
go func() {
    if haveCerts {
        if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
            slog.Error("Server error", "error", err)
            os.Exit(1)  // ← kills everything, no graceful shutdown
        }
    } else {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.Error("Server error", "error", err)
            os.Exit(1)  // ← same problem
        }
    }
}()
```

**Fix:** Instead of `os.Exit(1)`, signal the main goroutine to shut down gracefully. Use a channel:

```go
errChan := make(chan error, 1)

go func() {
    if haveCerts {
        if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
            errChan <- err
        }
    } else {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            errChan <- err
        }
    }
}()

// Select between signal channel and error channel
select {
case <-sigChan:
    slog.Info("Received shutdown signal")
case err := <-errChan:
    slog.Error("Server error, shutting down", "error", err)
}

// Graceful shutdown (already exists below this point)
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := srv.Shutdown(ctx); err != nil {
    slog.Error("Server shutdown error", "error", err)
}
gameLoop.Stop()
```

Also remove the `os.Exit(1)` calls at lines 78, 88, 96 (startup errors before the server starts) — these are fine to keep since the server hasn't started yet, but consider using `log.Fatal` for consistency.

**Regression Test:** Hard to unit test (requires starting an actual server). Manual verification:
1. Start the server, kill the process with SIGTERM — verify graceful shutdown messages appear
2. Start the server on a port that's already in use — verify it shuts down gracefully instead of panicking

**Cite:** No C equivalent — the C code was single-process with no signal handling. This is a Go-specific concurrency issue.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. **DP-566** (os.Exit) — smaller change, less risk, fixes data loss on errors
2. **DP-562** (door race) — more methods to change, but isolated to one package

## After Both Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
go build ./... && go vet ./... && go test ./...
git add -A
git commit -m "fix: server safety — door race condition + graceful shutdown (DP-562, DP-566)"
git push -u origin fix/server-safety-2026-06-12
gh pr create --title "fix: server safety (DP-562, DP-566)" --body "See docs/briefs/BRIEF-2026-06-12-server-safety.md for details."
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-562: Add comment "Fixed — DoorManager action methods now hold write lock for entire operation", commit <hash>, move to Done
- DP-566: Add commit hash comment, move to Done
