# BRIEF: Atomic Fixes — PoolMetrics race + Cache TOCTOU

**Date:** 2026-07-05
**Issues:** DP-708, DP-730
**Priority:** Medium
**Files:** `pkg/optimization/advanced_pool.go`, `pkg/optimization/cache.go`
**Cite:** No C equivalent — Go-only optimization layer. No corresponding code in `src/`.

---

## Fix 1: DP-708 — Data race on PoolMetrics fields (MEDIUM)

**File:** `pkg/optimization/advanced_pool.go` — `GetMetrics()` (line 227)

**Problem:**
`TasksCompleted`, `TasksFailed`, and `TasksSubmitted` are written via `atomic.AddInt64` (e.g. lines 136, 131, 178) but read via direct struct copy at line 232:

```go
metrics := p.metrics  // line 232: non-atomic copy of TasksSubmitted/Completed/Failed
metrics.QueueLength = atomic.LoadInt64(&p.metrics.QueueLength)        // atomic
metrics.PriorityLength = atomic.LoadInt64(&p.metrics.PriorityLength) // atomic
```

`QueueLength` and `PriorityLength` are correctly read atomically, but the three task-count fields are not. This is a data race per the Go memory model — concurrent `atomic.AddInt64` writers and non-atomic struct-copy readers accessing the same `int64` fields.

**Fix:**
Add `atomic.LoadInt64` for the three task-count fields:

```go
func (p *AdvancedWorkerPool) GetMetrics() PoolMetrics {
    p.mu.RLock()
    defer p.mu.RUnlock()

    metrics := p.metrics
    metrics.TasksSubmitted = atomic.LoadInt64(&p.metrics.TasksSubmitted)
    metrics.TasksCompleted = atomic.LoadInt64(&p.metrics.TasksCompleted)
    metrics.TasksFailed    = atomic.LoadInt64(&p.metrics.TasksFailed)
    metrics.QueueLength     = atomic.LoadInt64(&p.metrics.QueueLength)
    metrics.PriorityLength  = atomic.LoadInt64(&p.metrics.PriorityLength)

    if metrics.TasksSubmitted > 0 {
        metrics.WorkerUtilization = float64(metrics.TasksCompleted) / float64(metrics.TasksSubmitted)
    }

    return metrics
}
```

**Regression Test:** `pkg/optimization/advanced_pool_test.go`
- Add `TestGetMetrics_AtomicRead` — submit several tasks, then call `GetMetrics()` in a loop while concurrently submitting more tasks. Verify no `go test -race` failures.
- Existing tests may already cover this — run with `-race` to confirm.

**Verification:** `go build ./... && go vet ./... && go test -race ./...`

---

## Fix 2: DP-730 — TOCTOU race in Cache Get() (MEDIUM)

**File:** `pkg/optimization/cache.go` — `Get()` (line 55)

**Problem:**
`Get()` has a classic TOCTOU (time-of-check-time-of-use) race:

```go
func (c *Cache) Get(key string) (interface{}, bool) {
    c.mu.RLock()              // line 56
    item, exists := c.items[key]  // line 57
    c.mu.RUnlock()            // line 58  <-- lock released

    if !exists { return nil, false }

    if time.Now().After(item.expiresAt) {  // line 64 -- stale item ptr
        c.mu.Lock()                         // line 65  <-- TOCTOU gap
        delete(c.items, key)               // line 66  -- key may have been replaced
        c.mu.Unlock()
        return nil, false
    }

    c.mu.Lock()                         // line 71  <-- another TOCTOU gap
    item.accessCount++                   // line 72  -- item may be stale
    c.mu.Unlock()

    return item.value, true
}
```

Between `RUnlock` (line 58) and the subsequent `Lock` (lines 65 or 71), a concurrent writer can `Delete` the key or `Set` a new value. The `item` pointer captured at line 57 could be dangling or stale.

**Fix:**
Restructure `Get()` to either hold the write lock for the entire operation, or re-validate after acquiring the write lock. The simplest correct approach:

```go
func (c *Cache) Get(key string) (interface{}, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()

    item, exists := c.items[key]
    if !exists {
        return nil, false
    }

    if time.Now().After(item.expiresAt) {
        delete(c.items, key)
        return nil, false
    }

    item.accessCount++
    return item.value, true
}
```

Trade-off: This promotes all Gets to write locks, which reduces read concurrency. For the MUD server's usage pattern (moderate cache size, infrequent writes), this is acceptable and eliminates the race entirely. If performance becomes a concern, a `sharded mutex` or `sync.Map` replacement would be better, but that's an architectural change beyond this bug fix.

**Regression Test:** `pkg/optimization/cache_test.go`
- Add `TestCache_GetConcurrent` — create a cache, populate it, then concurrently call Get/Delete/Set. Use `-race` flag to verify no data races.
- Add `TestCache_GetExpired` — set an item with very short TTL, sleep past expiry, verify Get returns false.

**Verification:** `go build ./... && go vet ./... && go test -race ./...`

---

## Execution Order

1. Fix DP-708 first (atomic reads) — mechanical, no behavior change
2. Fix DP-730 second (Cache TOCTOU) — changes lock semantics, needs -race testing

## After All Fixes

- Run `go build ./... && go vet ./... && go test -race ./...`
- Create feature branch: `fix/dp-708-730-atomic-fixes`
- Commit: `fix: atomic PoolMetrics reads + Cache TOCTOU elimination (DP-708, DP-730)`
- Open PR against `main`
- Mark DP-708 and DP-730 as Done in Linear
