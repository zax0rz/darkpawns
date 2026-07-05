# BRIEF: Errcheck Sweep 2 — database.go missing error checks

**Date:** 2026-07-05
**Issues:** DP-701, DP-722
**Priority:** Medium
**Files:** `pkg/optimization/database.go`
**Cite:** No C equivalent — Go-only optimization layer (PostgreSQL index analysis, batch processor). No corresponding code in `src/`.

---

## Fix 1: DP-701 — AnalyzeTable ignores rows.Err() after iteration (MEDIUM)

**File:** `pkg/optimization/database.go` — `AnalyzeTable()` (line ~169-171)

**Problem:**
After the `for rows.Next()` loop at line 169, the function immediately returns `recommendations, nil` at line 171 without checking `rows.Err()`. Per Go's `database/sql` documentation, `rows.Err()` must always be checked after iteration — it returns any error that occurred during row iteration (network failure mid-scan, context cancellation, etc.). Without this check, partial results are returned as if successful, and the admin gets an incomplete index analysis silently.

**Fix:**
Add a `rows.Err()` check between the loop and the return:

```go
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating pg_stats rows: %w", err)
	}

	return recommendations, nil
```

**Regression Test:** `pkg/optimization/database_test.go`
- Add `TestAnalyzeTable_RowsError` — use a mock DB that returns rows where `rows.Err()` returns an error after iteration. Verify the function returns an error wrapping "iterating pg_stats rows".
- If no mock framework exists, this can be tested with a real SQLite/Postgres connection that is killed mid-iteration, or deferred to integration tests.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 2: DP-722 — flushLoop discards errors from flushLocked (MEDIUM)

**File:** `pkg/optimization/database.go` — `flushLoop()` (line ~293)

**Problem:**
Line 293 discards the error returned by `flushLocked()` with `_ =`:
```go
_ = bp.flushLocked()
```

While `flushLocked()` has retry logic (3 attempts with exponential backoff) and logs warnings on each retry and an error on final failure, the returned error after all retries is thrown away by the background goroutine. If the DB is completely down, the batch is permanently lost with no mechanism to notify any caller or trigger alerting. Compare with `Close()` at line 315 which correctly returns the error.

**Fix:**
Log the error at the same level as `flushLocked()`'s final failure (Error), and optionally add a metric counter:

```go
case <-ticker.C:
    bp.mu.Lock()
    if len(bp.operations) > 0 {
        if err := bp.flushLocked(); err != nil {
            // flushLocked already logged the error with details;
            // this is the final propagated failure from the background loop.
            slog.Error("background flush failed, data may be lost",
                "error", err)
        }
    }
    bp.mu.Unlock()
```

**Regression Test:** `pkg/optimization/database_test.go`
- Add `TestBatchProcessor_FlushLoopError` — create a BatchProcessor with a flushFunc that always returns an error. Trigger a flush via resetCh or by waiting for the interval. Verify the error is logged (use `slog.Default()` with a test handler or check that the operations buffer is cleared).
- Add `TestBatchProcessor_CloseReturnsError` — verify Close() propagates the error (this should already work but confirms the path).

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. Fix DP-701 first (rows.Err) — single-line addition, no behavior change for happy path
2. Fix DP-722 second (flushLoop error) — logging only, no behavior change

## After All Fixes

- Run `go build ./... && go vet ./... && go test ./...`
- Create feature branch: `fix/dp-701-722-errcheck-sweep-2`
- Commit: `fix: errcheck sweep 2 — rows.Err() + flushLoop error propagation (DP-701, DP-722)`
- Open PR against `main`
- Mark DP-701 and DP-722 as Done in Linear
