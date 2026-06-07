# Brief: CLEANUP-001 — Dead Code, Duplicates, and Cosmetic Cleanup

**Issues:** DP-542 (LOW), DP-544 (LOW), DP-552 (LOW), DP-538 (LOW), DP-537 (LOW)
**Priority:** LOW — cleanup
**Multiple files**

## Problem

Five low-priority cleanup items that don't affect runtime behavior but pollute the codebase.

### DP-542: packWeightLabel dead ratio computation (LOW)

`pkg/session/cmd_info.go:117-131` — `packWeightLabel()` has 15 lines of dead code at the top: computes `ratio`, clamps it, then calls `ratio = max(-1, ratio)` (result discarded). The function then starts over with fresh `weight` logic. The dead code is unreachable and the function works correctly without it.

**Fix:** Delete lines 117-131 (the first `ratio` computation block). Keep the `weight`-based logic that follows. Verify the function still returns the same values for test inputs.

### DP-544: Duplicate packWeightLabel finding (CLOSE AS DUPLICATE)

DP-544 and DP-542 describe the same dead code. Close DP-544 as duplicate of DP-542.

### DP-552: Admin UI cache headers (LOW)

`pkg/admin/router.go:52-67` — Admin SPA files served via `http.FileServer` and `http.ServeFile` have no cache-control headers. Browser may cache stale admin UI after deployments.

**Fix:** Add cache-control headers to admin static file serving:
```go
w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
w.Header().Set("Pragma", "no-cache")
w.Header().Set("Expires", "0")
```
Apply this to the `/admin/assets/`, `/admin/favicon.svg`, `/admin/icons.svg`, and `/admin/index.html` handlers.

### DP-538: Duplicate ShopManager interface (LOW)

`pkg/common/common.go:103-107` and `pkg/engine/shop.go:60-64` declare identical `ShopManager` interfaces. Duplicates cause divergent drift risk.

**Fix:** Keep the one in `pkg/common/common.go` (the shared interfaces package). Remove the duplicate from `pkg/engine/shop.go`. Update any imports in `pkg/engine/` to use `common.ShopManager`. Verify `go build ./...` passes.

### DP-537: No test coverage for common package (LOW)

`pkg/common/common.go` has zero tests. Every interface, constant, helper function, and behavioral contract is untested.

**Fix:** Create `pkg/common/common_test.go` with basic tests:
- Test `RaceDefs` map has entries for all expected races
- Test `CharClassDefs` map has entries for all expected classes
- Test attribute constants are non-zero and non-overlapping
- Test `SessionTracker` basic operations (add/remove/count)
- This is defensive — interfaces don't need unit tests, but the data maps do.

## Verification

- `go build ./...`
- `go vet ./...`
- `go test ./...` (including new common tests)
- Visual review: no `packWeightLabel` dead code, no duplicate `ShopManager`, admin UI assets have cache headers

## Context

These are all safe, low-risk changes. The dead code removal (DP-542) and duplicate closure (DP-544) can be done together. The cache headers (DP-552) are a one-liner per handler. The ShopManager dedup (DP-538) requires checking that no code imports the engine copy exclusively.
