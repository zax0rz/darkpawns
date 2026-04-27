# Pass 3: Security Deep Dive

**Reviewer:** Claude Opus 4 (automated security audit)
**Date:** 2026-04-26
**Scope:** Authentication, session management, WebSocket security, privilege escalation, Lua sandbox, file system access, player isolation, input validation

## Executive Summary

The codebase shows solid security fundamentals: bcrypt password hashing, JWT with enforced 32-byte minimum secret, proper CORS/CSP headers, Lua sandbox with removed dangerous stdlib modules, and parameterized SQL queries throughout. However, several logical vulnerabilities exist that bypass these controls. The most critical are: **telnet login requires no password** (complete auth bypass), the wizard **`idlist` command writes to arbitrary file paths**, the **`force` command doesn't actually execute the forced command** (incomplete but masks a design flaw), and the **`cmdAt` command enables recursive wizard command execution without depth limiting**. The X-Forwarded-For header is trusted without proxy validation, enabling rate limit bypass for login brute force.

---

## CRITICAL

### C1. Telnet Login Bypasses Password Authentication
**Files:** `pkg/telnet/listener.go:129-138`

The telnet `sendLogin()` function constructs a login message with `player_name` only — no password field. When `handleLogin()` in `manager.go:489-499` processes this for an existing player with a password set, it checks `if login.Password == ""` and sends "Password required" then closes. **However**, for any player name not yet in the DB, the flow falls through to the no-DB path (line 530) which creates a new character with zero authentication. More critically, if `hasDB` is false (no database configured), **all logins succeed with no authentication at all** (line 530-535).

Even with a DB, a telnet user can connect with any *new* name and get a fully authenticated session — the password for new characters is only required in the WebSocket path but the telnet `sendLogin()` never sends one, causing the "Password required for new characters" check to close the connection. This is actually a DoS vector — an attacker can repeatedly attempt creation of names, triggering the close-on-empty-password path, consuming rate limiter tokens for legitimate users.

**Impact:** Complete authentication bypass in no-DB mode; DoS via telnet name-squatting attempts in DB mode.

**Fix:**
```go
// telnet/listener.go sendLogin() — must prompt for and include password
func sendLogin(s *session.Session, name, password string) error {
    loginData, _ := json.Marshal(map[string]interface{}{
        "player_name": name,
        "password":    password,
    })
    // ...
}
```
Add password prompt to handleConn() before calling sendLogin(). For no-DB mode, either disable telnet or add a server-level password.

---

### C2. Wizard `idlist` Command — Arbitrary File Write
**Files:** `pkg/session/wizard_cmds.go:1240-1270`

The `cmdIdlist` command accepts a user-supplied filename argument and passes it directly to `os.Create(filename)`:

```go
filename := "idlist.txt"
if len(args) > 0 {
    filename = args[0]
}
f, err := os.Create(filename)
```

A wizard (level 61) can write to **any path the server process can access**: `idlist /etc/cron.d/backdoor`, `idlist ../../../root/.ssh/authorized_keys`, etc. The content is a formatted object dump, but the path is completely attacker-controlled.

**Impact:** Arbitrary file creation/overwrite on the server filesystem. A compromised wizard account leads to full server compromise.

**Fix:**
```go
// Force output to a safe directory
filename := filepath.Base(args[0]) // strip path components
if filename == "" || filename == "." || filename == ".." {
    filename = "idlist.txt"
}
safePath := filepath.Join("data", filename)
f, err := os.Create(safePath)
```

---

### C3. Wizard `sysfile` Command — Partial Path Traversal
**Files:** `pkg/session/wizard_cmds.go:1639-1674`

While `cmdSysfile` uses a switch statement to map section names to hardcoded paths (`data/bugs.txt`, etc.), the `#nosec G304` annotation suppresses the gosec warning. The current implementation is safe because the path is hardcoded. **However**, the function reads file content and sends it directly to the wizard's session with no size limit — a multi-megabyte bugs.txt file could cause memory pressure.

**Impact:** Low (currently hardcoded paths), but flagged for the `#nosec` suppression pattern.

**Fix:** Add a size limit: `data, err := io.ReadAll(io.LimitReader(f, 64*1024))`.

---

## HIGH

### H1. X-Forwarded-For Header Trusted Without Proxy Validation
**Files:** `pkg/auth/ratelimit.go:66-77`

`GetIPFromRequest()` blindly trusts the `X-Forwarded-For` header:

```go
if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
    if ips := strings.Split(forwarded, ","); len(ips) > 0 {
        return strings.TrimSpace(ips[0])
    }
}
```

Any client can set `X-Forwarded-For: 1.2.3.4` to spoof their IP, bypassing:
1. **Login rate limiting** (`loginLimiter` in manager.go:461)
2. **Per-IP connection limits** (ipConnCount in manager.go:192-202)

This makes brute-force attacks trivially achievable — each attempt uses a different spoofed IP.

**Impact:** Complete bypass of rate limiting and connection limits. Enables credential brute forcing.

**Fix:**
```go
func GetIPFromRequest(r *http.Request, trustedProxies []string) string {
    // Only trust X-Forwarded-For from known proxy IPs
    if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
        remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
        if isTrustedProxy(remoteIP, trustedProxies) {
            // Take the rightmost untrusted IP
            ips := strings.Split(forwarded, ",")
            return strings.TrimSpace(ips[len(ips)-1])
        }
    }
    host, _, _ := net.SplitHostPort(r.RemoteAddr)
    return host
}
```

### H2. WebSocket Allows Connections Without Origin Header in Production
**Files:** `pkg/session/manager.go:45-51`

```go
if origin == "" {
    slog.Warn("WebSocket connection without Origin header", ...)
    return true  // ALLOWS connection
}
```

When no `Origin` header is present (e.g., from a raw TCP WebSocket client, curl, or a non-browser tool), the connection is allowed even in production mode. Combined with H1, this means an attacker can connect from any tool without browser-based origin restrictions.

**Impact:** CSRF-like attacks possible if any session state is cookie-based (currently JWT-based, so lower impact). More importantly, removes a defense-in-depth layer.

**Fix:** In production mode, reject connections without an Origin header, or require an explicit configuration flag to allow them.

### H3. `cmdAt` — Recursive Wizard Command Execution Without Depth Limit
**Files:** `pkg/session/wizard_cmds.go:70-85`

The `at` command teleports the wizard, executes a command, then returns:

```go
s.player.SetRoom(dest)
defer s.player.SetRoom(orig)
ExecuteCommand(s, strings.Fields(rest)[0], strings.Fields(rest)[1:])
```

A wizard can run `at 100 at 200 at 300 shutdown` — there's no recursion depth limit. While `shutdown` doesn't actually shut down (stub), the pattern enables:
1. Stack overflow via deep recursion: `at 100 at 100 at 100 ...` (Go goroutine stack grows but is bounded, typically 1GB — still a DoS)
2. Executing any wizard command at any room without audit attribution (the intermediate rooms aren't logged)

**Impact:** Potential DoS via stack exhaustion; audit log evasion for wizard actions.

**Fix:** Add a recursion counter to Session:
```go
if s.atDepth > 3 {
    s.Send("Maximum 'at' nesting depth reached.")
    return nil
}
s.atDepth++
defer func() { s.atDepth-- }()
```

### H4. `force` Command Is a Stub — But Doesn't Execute the Command
**Files:** `pkg/session/wizard_cmds.go:450-477`

The `force` command logs the intent but **never calls `ExecuteCommand()`** on the target:

```go
slog.Info("forced", "target", target.player.Name, "command", forceCmd, "by", s.player.Name)
s.Send(fmt.Sprintf("Forced %s to '%s'.", target.player.Name, forceCmd))
return nil  // Command never executed!
```

The "force all" path is similarly a no-op. This is currently a **security positive** (it can't be exploited), but it's also a functional bug — when it gets implemented, it needs careful attention:
- `force <target> force <other> <cmd>` — transitive force chains
- `force <target> shutdown` — privilege escalation if target is lower-level
- The forceCmd parsing only takes `args[1]`, dropping additional args

**Impact:** Currently none (stub). Flagged as HIGH because the intended functionality is inherently dangerous and the current code structure suggests it will be filled in without security review.

**Fix:** When implementing, ensure:
1. Forced command runs at the **target's** privilege level, not the forcer's
2. Prevent force chains (force someone to force someone else)
3. Parse the full command string, not just args[1]

### H5. JWT Token Exposed in Welcome State Message
**Files:** `pkg/session/manager.go:383-409` (sendWelcome), `pkg/session/protocol.go:61`

The JWT token is sent in the initial `state` message over WebSocket:

```go
state := StateData{
    ...
    Token: token,  // JWT included in state
}
```

The `RoomState` includes `Players []string` — a list of player names visible in the room. This means every player in the room receives a broadcast containing the new player's state... **but** looking at the code more carefully, `sendWelcome` sends only to `s.send` (the authenticated player's own channel), not via broadcast. The JWT is therefore not leaked to other players.

However, the token is sent over the WebSocket which may not be TLS-encrypted in development. The token has a 24-hour lifetime with no refresh/rotation mechanism — if intercepted, it provides API access for a full day.

**Impact:** Medium — token interception over unencrypted WebSocket gives 24-hour API access.

**Fix:** Reduce token lifetime to 1 hour with refresh mechanism. Ensure WSS-only in production.

### H6. No Account Lockout After Failed Login Attempts
**Files:** `pkg/session/manager.go:461-469`, `pkg/auth/ratelimit.go`

The rate limiter allows 5 requests/second with burst of 10, per IP. Combined with the X-Forwarded-For bypass (H1), there is **no account-level lockout**. An attacker can attempt unlimited passwords against a single account by rotating spoofed IPs.

**Impact:** Credential brute forcing against individual accounts.

**Fix:** Add per-account rate limiting in addition to per-IP:
```go
// Track per-account failed attempts
type accountLimiter struct {
    failures  int
    lockedUntil time.Time
}
```
Lock accounts for escalating durations: 5 failures → 30s lockout, 10 → 5min, 20 → 1hr.

---

## MEDIUM

### M1. CORS Wildcard Subdomain Matching Is Overly Permissive
**Files:** `web/cors.go:53-58`

```go
if strings.HasPrefix(allowedOrigin, "*.") {
    domain := strings.TrimPrefix(allowedOrigin, "*.")
    if strings.HasSuffix(origin, domain) {
        return true
    }
}
```

If `allowedOrigin` is `*.darkpawns.example.com`, then `evil-darkpawns.example.com` would also match because `HasSuffix` doesn't check for a dot boundary. An attacker registering `evil-darkpawns.example.com` could make cross-origin requests.

**Impact:** Cross-origin request forgery from malicious subdomains.

**Fix:** Check for dot boundary:
```go
if strings.HasSuffix(origin, "."+domain) || origin == "https://"+domain {
```

### M2. Development Mode CORS Allows All Origins
**Files:** `web/cors.go:44-46`

```go
if os.Getenv("ENVIRONMENT") == "development" {
    return true
}
```

If `ENVIRONMENT` is accidentally left as "development" in production (or unset, which defaults to non-development — but the risk exists), all origins are allowed for both CORS and WebSocket connections.

**Impact:** If deployed with wrong ENVIRONMENT value, all origin restrictions are disabled.

**Fix:** Make this a compile-time flag or require explicit opt-in: `ALLOW_ALL_ORIGINS=true`.

### M3. CSP Allows `unsafe-inline` for Scripts and Styles
**Files:** `web/security.go:14-15`

```go
csp := "default-src 'self'; " +
    "script-src 'self' 'unsafe-inline'; " +
    "style-src 'self' 'unsafe-inline'; " +
```

`unsafe-inline` for scripts defeats the purpose of CSP against XSS. If any user-controlled content is rendered in HTML (player names, room descriptions, chat messages), an XSS payload could execute.

**Impact:** XSS protection from CSP is significantly weakened.

**Fix:** Use nonces or hashes instead of `unsafe-inline` for scripts. Keep `unsafe-inline` only for styles if absolutely necessary.

### M4. `cmdUsers` Exposes Player IP Addresses to All Wizards
**Files:** `pkg/session/act_informative.go:246-256`

The `users` command (LVL_IMMORT = level 50) displays the raw IP address (or X-Forwarded-For value) of every connected player. Any level-50 immortal can see the real IP addresses of all online players.

**Impact:** Privacy violation — player IP addresses disclosed to all wizard-level characters.

**Fix:** Restrict to LVL_GRGOD (61) or redact to show only network prefix (e.g., `192.168.x.x`).

### M5. `cmdSwitch` Doesn't Fully Implement Body Switching
**Files:** `pkg/session/wizard_cmds.go:307-357`

The `switch` command sets `isSwitched = true` and stores references, but **doesn't change `s.player`** to the target. This means:
1. All commands still execute as the wizard, not the switched mob/player
2. The wizard retains their own privileges while "in" another body
3. A switched wizard can modify their own player object while pretending to be someone else

This is a logic bug that becomes a security issue if the wizard expects that actions are attributed to the switched target.

**Impact:** Audit log confusion — actions appear as wizard, not switched entity. Potential for unattributed actions.

**Fix:** Either complete the switch implementation (changing `s.player` to the target) or clearly document that switch is cosmetic only.

### M6. Lua `dofile` Re-Registered After Sandbox Removal
**Files:** `pkg/scripting/engine.go:50, 297`

The sandbox setup in `newSafeLState()` removes `dofile`:
```go
L.SetGlobal("dofile", lua.LNil)
```

But then `registerFunctionsOn()` re-registers it as a custom sandboxed version:
```go
L.SetGlobal("dofile", L.NewFunction(e.luaDofile))
```

The custom `luaDofile` does have path traversal protection:
```go
rel, err := filepath.Rel(e.scriptsDir, fullPath)
if err != nil || strings.HasPrefix(rel, "..") || strings.HasPrefix(rel, "/") {
    slog.Warn("dofile: path traversal blocked", "path", path)
    return 0
}
```

The protection is sound, but the pattern of removing then re-adding `dofile` is confusing and error-prone. A comment explaining the intentional override would help.

**Impact:** Low (properly sandboxed), but the pattern invites accidental removal of the sandbox check.

**Fix:** Add a comment at the removal site: `// Removed here, re-added as sandboxed version in registerFunctionsOn()`.

### M7. Rate Limiter Cleanup Is Crude — Unbounded Memory Growth
**Files:** `pkg/auth/ratelimit.go:30-43`

```go
if len(i.ips) > 10000 {
    i.ips = make(map[string]*rate.Limiter)
}
```

The cleanup strategy simply wipes the entire map when it exceeds 10,000 entries. This means:
1. Up to 10,000 entries accumulate with no eviction
2. When cleared, all rate limits reset simultaneously — burst of previously-limited IPs all become unlimited
3. An attacker can intentionally create 10,000 entries to trigger a reset

**Impact:** Periodic complete bypass of all IP-based rate limiting.

**Fix:** Use an LRU cache with TTL or track last-access time per entry.

### M8. `who` Command Reveals Agent Status
**Files:** `pkg/session/commands.go` (cmdWho function)

```go
tag := "player"
if sess.isAgent {
    tag = "agent"
}
out += fmt.Sprintf("... (%s, %s, %s)\n", raceName, className, tag)
```

All players can see which sessions are agent-controlled bots vs human players. This leaks implementation details about automated players.

**Impact:** Information disclosure — bot detection by any player.

**Fix:** Remove agent tag from public-facing output, or only show to wizards.

---

## LOW

### L1. `math/rand` Used for Game RNG (Expected)
**Files:** Multiple locations

`math/rand` (not `crypto/rand`) is used for game mechanics (combat damage, stat rolls, etc.). This is appropriate for game randomness — flagged only for completeness. The `#nosec G404` annotations are correct.

### L2. Lua `OpenLibs()` Opens All Standard Libraries Before Removal
**Files:** `pkg/scripting/engine.go:41`

`L.OpenLibs()` opens everything, then dangerous modules are removed. This is the standard pattern for gopher-lua but means there's a brief window where all libraries are available. Since no scripts execute between `OpenLibs()` and the removal, this is not exploitable.

### L3. No CSRF Protection on API Endpoints
**Files:** `web/auth.go`, `cmd/server/main.go:124`

API endpoints use JWT Bearer token authentication. Since the token must be in the `Authorization` header (not a cookie), CSRF is not applicable — browsers don't auto-attach custom headers. This is a secure pattern.

### L4. Session `tempData` Has No Type Safety
**Files:** `pkg/session/manager.go` (SetTempData)

`tempData map[string]interface{}` stores arbitrary data without type checking. While not directly exploitable, type confusion bugs could arise if two command handlers use the same key with different types.

### L5. Player Name Validation Allows Dots, Spaces, Dashes
**Files:** `pkg/validation/validation.go:10`

```go
playerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\. ]+$`)
```

While `sanitizeName()` in save.go strips these for filenames, the broader validation allows characters that could cause display issues (names with spaces, dots) or social engineering (names that look like system messages).

### L6. `sendError` Uses Non-Blocking Channel Send
**Files:** `pkg/session/manager.go:422-430`

```go
select {
case s.send <- msg:
default:
    // Channel full, drop message
}
```

Error messages to the client are silently dropped if the send buffer is full. This means security-relevant errors (rate limit, auth failure) may not reach the client. Not exploitable, but could confuse legitimate users.

---

## Prioritized Summary

| # | Severity | Finding | Effort | Priority |
|---|----------|---------|--------|----------|
| C1 | CRITICAL | Telnet login bypasses password auth | Medium | **P0** |
| C2 | CRITICAL | `idlist` arbitrary file write | Low | **P0** |
| H1 | HIGH | X-Forwarded-For trusted without proxy validation | Medium | **P1** |
| H2 | HIGH | WebSocket allows no-Origin connections in production | Low | **P1** |
| H3 | HIGH | `cmdAt` recursive execution without depth limit | Low | **P1** |
| H6 | HIGH | No account-level lockout for failed logins | Medium | **P1** |
| H5 | HIGH | JWT 24h lifetime, no rotation | Medium | **P2** |
| H4 | HIGH | `force` command stub — dangerous when implemented | Low (doc) | **P2** |
| M1 | MEDIUM | CORS wildcard subdomain matching too permissive | Low | **P2** |
| M3 | MEDIUM | CSP allows unsafe-inline for scripts | Medium | **P2** |
| M4 | MEDIUM | `users` command exposes player IPs to wizards | Low | **P3** |
| M5 | MEDIUM | `switch` doesn't change player context | Medium | **P3** |
| M7 | MEDIUM | Rate limiter cleanup enables periodic bypass | Medium | **P3** |
| M8 | MEDIUM | `who` reveals agent status to all players | Low | **P3** |
| M2 | MEDIUM | Development CORS allows all origins | Low | **P3** |
| M6 | MEDIUM | Lua dofile re-registration pattern confusing | Low | **P4** |
| C3 | LOW-MED | `sysfile` no size limit on file reads | Low | **P4** |

**Top 3 immediate actions:**
1. **Fix telnet auth** — add password prompt or disable telnet in production
2. **Fix `idlist` path** — restrict output to a safe directory
3. **Fix X-Forwarded-For** — only trust from configured proxy IPs
