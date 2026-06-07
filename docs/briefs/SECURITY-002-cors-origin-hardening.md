# Brief: SECURITY-002 — CORS & WebSocket Origin Hardening

**Issues:** DP-548 (MEDIUM), DP-549 (MEDIUM), DP-551 (LOW)
**Priority:** MEDIUM — security
**Files:** `pkg/session/manager.go`, `web/cors.go`, `pkg/admin/router.go`, `k8s/server.yaml`

## Problem

Three related issues share the same root cause: placeholder/development CORS and WebSocket origin configuration is still in production code.

1. **DP-548 (MEDIUM):** `pkg/admin/router.go:167-168` — `corsMiddleware()` hardcodes `http://localhost:5173` and `http://localhost:4350`. `ADMIN_CORS_ORIGIN` env var exists but isn't set in k8s deployment. **This is a hard failure** — admin panel accessed through `darkpawns.labz0rz.com` will be CORS-rejected before auth headers are sent.

2. **DP-549 (MEDIUM):** `pkg/session/manager.go:44` — `CheckOrigin` returns `true` for all origins when `ENVIRONMENT=development`. The k8s manifest (`k8s/server.yaml`) doesn't set `ENVIRONMENT` at all. If the container default is unset or "development", WebSocket connections from any origin are accepted.

3. **DP-551 (LOW):** `pkg/session/manager.go:51-52` and `web/cors.go:50-51` — `darkpawns.example.com` and `game.darkpawns.example.com` appear as allowed origins. These are placeholders. If someone registers `darkpawns.example.com`, it would be in the allowlist.

## Required Fixes (order matters)

### Fix 1: k8s/server.yaml — set ENVIRONMENT=production (FIRST)

Add to the `env` section:
```yaml
- name: ENVIRONMENT
  value: "production"
```

This closes the dev-mode backdoor in both `pkg/session/manager.go` (WebSocket accepts all origins) AND `web/cors.go` (`isDevMode()` returns true). **Must be done before domain cleanup** — Fix 2 without Fix 1 leaves production CORS on placeholder domains while dev mode is still active.

### Fix 2: web/cors.go — replace placeholder domains (SECOND)

In `getAllowedOrigins()`, replace the production defaults:
```go
// Before:
return []string{
    "https://darkpawns.example.com",
    "https://game.darkpawns.example.com",
}

// After:
return []string{
    "https://darkpawns.labz0rz.com",
}
```

**Critical:** Also remove the `allowedSubdomains` map entry for `darkpawns.example.com`:
```go
// Before (lines ~90-93):
var allowedSubdomains = map[string][]string{
    "darkpawns.example.com": {"game", "www", "api"},
}

// After: delete entirely or replace with:
var allowedSubdomains = map[string][]string{}
```

The `allowedSubdomains` map is a separate structure from the explicit origins list. It would still match `game.darkpawns.example.com` even after the explicit origins are cleaned up.

### Fix 3: pkg/session/manager.go — remove placeholder domains in CheckOrigin (THIRD)

In the `allowedOrigins` slice inside `CheckOrigin`, remove the dead entries:
```go
// Before:
allowedOrigins := []string{
    "https://darkpawns.labz0rz.com",
    "https://darkpawns.example.com",
    "https://game.darkpawns.example.com",
}

// After:
allowedOrigins := []string{
    "https://darkpawns.labz0rz.com",
}
```

Note: `darkpawns.labz0rz.com` was already correct — this fix just removes the two dead entries.

### Fix 4: k8s/server.yaml + pkg/admin/router.go — admin panel CORS (NOT OPTIONAL)

The admin panel CORS is a **hard failure** if someone accesses through the production domain. CORS blocks the preflight before auth headers are sent — "behind authentication" doesn't help.

**Option A (preferred):** Add `ADMIN_CORS_ORIGIN` to k8s/server.yaml:
```yaml
- name: ADMIN_CORS_ORIGIN
  value: "https://darkpawns.labz0rz.com"
```

**Option B:** Add `darkpawns.labz0rz.com` as a default in the corsMiddleware alongside the localhost entries:
```go
allowed := origin == "http://localhost:5173" || origin == "http://localhost:4350" ||
    origin == "https://localhost:5173" || origin == "https://localhost:4350" ||
    origin == "https://darkpawns.labz0rz.com"
```

Option A is cleaner — keeps the env-var escape hatch pattern.

## Verification

1. `go build ./...`
2. `go vet ./...`
3. Confirm `ENVIRONMENT=production` is set in k8s deployment
4. Confirm `darkpawns.example.com` no longer appears in:
   - `web/cors.go` `getAllowedOrigins()` return value
   - `web/cors.go` `allowedSubdomains` map
   - `pkg/session/manager.go` `CheckOrigin` allowlist
5. Confirm `ADMIN_CORS_ORIGIN` is set to `https://darkpawns.labz0rz.com` in k8s
6. Test WebSocket connection from browser at `darkpawns.labz0rz.com` → should succeed
7. Test WebSocket connection from random origin → should be rejected
8. Test admin panel access from `darkpawns.labz0rz.com` → should load without CORS error

## Context

The `isDevMode()` check in `web/cors.go` logs a warning when dev mode allows all origins — this is intentional for local development but must never activate in production. Setting `ENVIRONMENT=production` in k8s is the critical first fix. Domain cleanup follows. Admin panel CORS is not optional — it's a deployment blocker for production access.
