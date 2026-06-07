# Brief: SECURITY-001 — Admin Login Brute-Force Lockout

**Issues:** DP-547 (HIGH)
**Priority:** HIGH — security
**Files:** `pkg/admin/login.go`, `pkg/admin/router.go`

## Problem

The admin login endpoint (`handleLogin` in `pkg/admin/login.go:38`) calls `bcrypt.CompareHashAndPassword()` directly without checking `LoginAttemptTracker.IsLocked()` first. The telnet login path (`pkg/session/session_login.go:36,176`) correctly uses the tracker, but admin login doesn't.

This means an attacker can brute-force admin credentials at the rate limiter's allowance (5 requests/sec) with no lockout after 10,000+ failures. Each attempt also burns a full bcrypt computation, which is a CPU exhaustion vector.

**Additional issue — username enumeration:** `handleLogin` returns different responses for "player not found" (line 54) vs "invalid password" (line 64). An attacker can enumerate valid usernames without ever triggering the password check or the lockout counter. At minimum, `RecordFailure` must be called on BOTH paths.

## Required Fix

### Step 1: Add tracker parameter to handleLogin

Change `handleLogin` signature to accept a tracker:

```go
func handleLogin(database *db.DB, loginAttempts *auth.LoginAttemptTracker) http.HandlerFunc {
```

### Step 2: Wire tracker in NewRouter

In `NewRouter`, create a tracker and pass it to `handleLogin`. Follow the existing `rateLimiter` pattern in `router.go:20`:

```go
loginAttempts := auth.NewLoginAttemptTracker(auth.LoginAttemptConfig{
    Threshold: 10,
    Lockout:   15 * time.Minute,
})
// ... later:
mux.HandleFunc("/admin/login", wrap(handleLogin(database, loginAttempts)))
```

Note: use a SEPARATE tracker instance from the session manager's. This is intentional — an IP locked out on telnet should still be able to attempt admin login (and vice versa). The trackers are independent lockout domains.

### Step 3: Add lockout check BEFORE JSON decode

Parse the IP first, check lockout before burning the JSON decode:

```go
func handleLogin(database *db.DB, loginAttempts *auth.LoginAttemptTracker) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
            return
        }

        ip := auth.GetIPFromRequest(r)

        // Lock out BEFORE parsing body — don't burn JSON decode on locked IPs
        if locked, remaining := loginAttempts.IsLocked(ip); locked {
            mins := int(remaining.Minutes()) + 1
            http.Error(w, fmt.Sprintf(`{"error":"too many failed attempts, try again in %d minutes"}`, mins), http.StatusTooManyRequests)
            audit.LogSecurityEvent("login_locked_out", "Admin login locked out", "", ip)
            return
        }

        var req loginRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
            return
        }
        // ... rest of handler
```

### Step 4: RecordFailure on ALL failure paths

Call `RecordFailure(ip)` on BOTH "player not found" AND "invalid password" — prevents username enumeration:

```go
// Look up the player
rec, err := database.GetPlayer(req.PlayerName)
if err != nil {
    loginAttempts.RecordFailure(ip)  // ← NEW
    http.Error(w, `{"error":"player not found"}`, http.StatusUnauthorized)
    return
}

// Verify password
if rec.Password == "" {
    loginAttempts.RecordFailure(ip)  // ← NEW
    http.Error(w, `{"error":"no password set for this player"}`, http.StatusUnauthorized)
    return
}
if err := bcrypt.CompareHashAndPassword([]byte(rec.Password), []byte(req.Password)); err != nil {
    loginAttempts.RecordFailure(ip)  // ← NEW
    http.Error(w, `{"error":"invalid password"}`, http.StatusUnauthorized)
    return
}
```

### Step 5: RecordSuccess on successful login

After password verification succeeds:

```go
loginAttempts.RecordSuccess(ip)  // ← NEW — reset failure counter
```

### Step 6: Add missing imports

`login.go` currently imports: `encoding/json`, `log/slog`, `net/http`, `auth`, `db`, `bcrypt`. Add:

```go
import (
    "fmt"       // for Sprintf in lockout message
    "github.com/zax0rz/darkpawns/pkg/audit"  // for LogSecurityEvent
)
```

## Verification

1. `go build ./...`
2. `go vet ./...`
3. `go test ./pkg/admin/...` — existing tests should pass
4. Add a unit test for lockout (follow pattern in `handlers_test.go:1499`):
   - Pre-populate tracker with 10 failures for test IP
   - Assert `handleLogin` returns 429 with lockout message
   - Assert bcrypt is NOT called (no CPU burn on locked IP)
5. Verify telnet login still works (separate tracker instance, unchanged)
6. Verify "player not found" and "invalid password" both increment failure counter

## Context

The `LoginAttemptTracker` config is `Threshold: 10, Lockout: 15 * time.Minute` (matching session manager). Use a separate instance — independent lockout domains are intentional. An attacker can't brute-force both telnet and admin simultaneously from the same IP, but a legitimate user locked out on one can still use the other.
