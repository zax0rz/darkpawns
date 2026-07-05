# Brief: Reek HIGH Batch — Two Fixes

Owner: Kimi (has Linear access)
Scope: Fix 2 real bugs, close 1 already-fixed issue.

## Issue 1: DP-782 — Container Cycle Prevention

**Problem:** Moving container A into container B (when B is already inside A) creates a reference cycle A→B→A. This corrupts `GetTotalWeight` and any recursive Contains walker.

**File:** `pkg/game/object_movement.go`

**Current state:** A test exists (`TestContainerCyclePrevention` in `pkg/game/object_movement_test.go:558`) but the test **documents the bug** — it asserts the cycle IS created (line 582: `t.Error("BUG: MoveObjectToContainer(A into B) succeeded, creating a container cycle")`).

**Fix:**
1. In `attachObjectLocked` (or `MoveObjectToContainer`), before attaching an object to `ObjInContainer`, walk the destination container's ancestry:
   - Start at the destination container
   - Follow `ObjInContainer` links upward
   - If we encounter the object being moved, it's a cycle — reject
2. The ancestry walk function: start at destination, check if destination == object, then follow destination's Location chain while Location.Kind == ObjInContainer
3. Return an error like `"cannot move container into itself (cycle detected)"`

**Acceptance:**
- `TestContainerCyclePrevention` passes (flip the assertion from "BUG:" to expected behavior)
- `go build ./... && go vet ./...` passes
- Add test for deeper cycles: A→B→C, moving A into C must also fail

---

## Issue 2: DP-781 — WorkerPool Backoff Goroutine Panic on Close

**Problem:** In `pkg/optimization/advanced_pool.go:80-105`, when the task queue is full, a backoff goroutine sleeps then attempts to send on `p.taskQueue`. Meanwhile `Close()` (line 288-295) calls `close(p.stop)` then `close(p.taskQueue)`. If the goroutine wakes after both are closed, Go's select picks uniformly at random between ready cases — the send-to-closed-channel case panics.

**File:** `pkg/optimization/advanced_pool.go`

**Current code (lines 80-105):**
```go
go func(pt priorityTask) {
    defer p.producerWG.Done()
    timer := time.NewTimer(time.Millisecond * time.Duration(pt.priority*10))
    defer timer.Stop()
    select {
    case <-timer.C:
    case <-p.stop:
        atomic.AddInt64(&p.metrics.TasksFailed, 1)
        return
    }
    select {
    case p.taskQueue <- pt.task:  // PANICS if taskQueue is closed
        ...
    case <-p.stop:
        ...
    }
}(pt)
```

**Fix:** After the timer fires, check `p.stop` with a non-blocking receive BEFORE attempting the send:
```go
select {
case <-timer.C:
case <-p.stop:
    atomic.AddInt64(&p.metrics.TasksFailed, 1)
    return
}
// Check stop again before attempting send — channel may have closed during timer
select {
case <-p.stop:
    atomic.AddInt64(&p.metrics.TasksFailed, 1)
    return
default:
}
select {
case p.taskQueue <- pt.task:
    waitTime := time.Since(pt.submitted)
    p.updateWaitTime(waitTime)
case <-p.stop:
    atomic.AddInt64(&p.metrics.TasksFailed, 1)
}
```

**Alternative (simpler):** Use a recover to catch the closed-channel panic:
```go
defer func() {
    if r := recover(); r != nil {
        atomic.AddInt64(&p.metrics.TasksFailed, 1)
    }
}()
```
But the explicit check is cleaner.

**Acceptance:**
- `go build ./... && go vet ./...` passes
- Add test: submit tasks until queue is full, immediately Close, loop under `-race` detector

---

## Issue 3: DP-783 — Already Fixed (close it)

**What happened:** Reek flagged that `HTTPMiddleware` and `WebSocketLogger` called `GetGlobalLogger().SetClient(client)` replacing the global singleton. The current code at `middleware.go:74-76` creates a request-local `NewPrivacyLogger(client, ...)` instead. Bug is already resolved.

**Action:** Move DP-783 to Done on Linear with comment: "Already fixed — middleware creates request-local PrivacyLogger via NewPrivacyLogger(client) instead of mutating global singleton. See middleware.go:74-76."

---

## Workflow

1. Fix DP-782 (container cycle) — add ancestry check
2. Fix DP-781 (worker pool race) — add stop check before send
3. Update test for DP-782
4. Run: `go build ./... && go vet ./...`
5. Commit per fix
6. Update Linear:
   - DP-782: Done with commit hash
   - DP-781: Done with commit hash
   - DP-783: Done with "already fixed" comment

## Non-goals

- Do not refactor the worker pool architecture
- Do not add container cycle tests beyond what's needed
- Do not touch DP-783 code (already fixed)
