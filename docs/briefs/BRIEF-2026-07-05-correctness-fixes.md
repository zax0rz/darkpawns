# BRIEF: Correctness Fixes — RateLimiter cleanup + BatchFilter partials

**Date:** 2026-07-05
**Issues:** DP-703, DP-731
**Priority:** Medium
**Files:** `pkg/auth/ratelimit.go`, `pkg/privacy/client.go`
**Cite:** No C equivalent — Go-only optimization/administrative layer. No corresponding code in `src/`.

---

## Fix 1: DP-703 — IPRateLimiter cleanup loop skips earliest entries (MEDIUM)

**File:** `pkg/auth/ratelimit.go` — `cleanupLoop()` (line 119)

**Problem:**
The cleanup loop has a `count > 10000` guard that makes only entries visited after position 10000 eligible for deletion:

```go
i.ips.Range(func(key, value any) bool {
    count++
    limiter := value.(*rate.Limiter)
    if count > 10000 && limiter.Tokens() >= float64(limiter.Burst()) {
        i.ips.Delete(key)
    }
    return true
})
```

Since `sync.Map.Range` iteration order is non-deterministic, whichever entries happen to be visited in the first 10000 positions are permanently immune to cleanup on that pass. The intent of the guard (avoid scanning the entire map in one pass) is reasonable, but the implementation is wrong — it should use a maximum-deletions-per-pass counter instead of a minimum-position guard.

**Fix:**
Replace the position guard with a max-deletions-per-pass counter:

```go
func (i *IPRateLimiter) cleanupLoop() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            const maxDeletions = 500
            deleted := 0
            i.ips.Range(func(key, value any) bool {
                if deleted >= maxDeletions {
                    return false // stop iterating
                }
                limiter := value.(*rate.Limiter)
                if limiter.Tokens() >= float64(limiter.Burst()) {
                    i.ips.Delete(key)
                    deleted++
                }
                return true
            })
        case <-i.stopCh:
            return
        }
    }
}
```

This ensures all entries are visited each pass, but only up to 500 are deleted. On the next tick, the iteration starts fresh and can delete more stale entries. No entry is permanently immune.

**Regression Test:** `pkg/auth/ratelimit_test.go`
- Add `TestIPRateLimiter_CleanupDeletesAllStale` — add >10000 IPs with full-burst limiters (all stale), run cleanup, verify they are all eventually deleted within a few cleanup cycles.
- Add `TestIPRateLimiter_CleanupPreservesActive` — add IPs with active (non-full) limiters, run cleanup, verify they are NOT deleted.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 2: DP-731 — BatchFilter drops all partial results on single-element error (MEDIUM)

**File:** `pkg/privacy/client.go` — `BatchFilter()` (line 139)

**Problem:**
`BatchFilter` immediately returns `nil, nil, err` on the first element error (line 149):

```go
func (c *Client) BatchFilter(texts []string) ([]string, [][]string, error) {
    var filteredTexts []string
    var allDetected [][]string

    for _, text := range texts {
        filtered, detected, err := c.FilterText(text)
        if err != nil {
            return nil, nil, err  // drops all accumulated results
        }
        filteredTexts = append(filteredTexts, filtered)
        allDetected = append(allDetected, detected)
    }

    return filteredTexts, allDetected, nil
}
```

While `FilterText` now falls back to a local filter for most errors (network failure, non-200 status), it still returns an error for JSON decode failure on a 200 response (line 123: `fmt.Errorf("failed to decode response: %w", err)`). If that one edge case hits, ALL accumulated results are lost.

**Fix:**
Collect partial results and return them alongside the error:

```go
func (c *Client) BatchFilter(texts []string) ([]string, [][]string, error) {
    var filteredTexts []string
    var allDetected [][]string
    var firstErr error

    for _, text := range texts {
        filtered, detected, err := c.FilterText(text)
        if err != nil {
            // Use fallback for the failed element; record first error.
            filtered = c.fallbackFilter(text)
            detected = []string{"fallback"}
            if firstErr == nil {
                firstErr = fmt.Errorf("FilterText failed for element %d: %w", len(filteredTexts), err)
            }
        }
        filteredTexts = append(filteredTexts, filtered)
        allDetected = append(allDetected, detected)
    }

    return filteredTexts, allDetected, firstErr
}
```

This preserves all accumulated results (using fallback for failed elements) and returns the first error so callers know something went wrong.

**Regression Test:** `pkg/privacy/client_test.go`
- Add `TestBatchFilter_PartialError` — use a mock server that returns a malformed JSON response for one element. Verify the function returns all results (with fallback for the failed element) AND a non-nil error.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. Fix DP-703 first (ratelimit cleanup) — self-contained, easy to test
2. Fix DP-731 second (BatchFilter) — different package, independent

## After All Fixes

- Run `go build ./... && go vet ./... && go test ./...`
- Create feature branch: `fix/dp-703-731-correctness`
- Commit: `fix: ratelimit cleanup immunity + BatchFilter partial loss (DP-703, DP-731)`
- Open PR against `main`
- Mark DP-703 and DP-731 as Done in Linear
