# Dark Pawns Security Audit Report

**Date:** 2026-04-22  
**Auditor:** Agent 63 (Security Audit Subagent)  
**Scope:** Full codebase security review  
**Location:** `/home/zach/.openclaw/workspace/darkpawns_repo/`

## Executive Summary

The Dark Pawns codebase demonstrates generally good security practices with proper use of parameterized SQL queries, rate limiting, and secure password handling. However, several critical issues were identified requiring immediate attention, particularly around CORS configuration, hardcoded credentials, and input validation.

## Critical Findings (Severity: High)

### 1. **CORS Misconfiguration - WebSocket Origin Check**
**Location:** `pkg/session/manager.go:23-25`
```go
CheckOrigin: func(r *http.Request) bool {
    return true // Allow all origins for development
}
```
**Risk:** Allows WebSocket connections from any origin, enabling Cross-Site WebSocket Hijacking (CSWSH) attacks.
**Impact:** Malicious websites could establish WebSocket connections to the game server on behalf of authenticated users.
**Fix:** Implement proper origin validation in production:
```go
CheckOrigin: func(r *http.Request) bool {
    allowedOrigins := []string{"https://yourdomain.com", "https://game.yourdomain.com"}
    origin := r.Header.Get("Origin")
    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    return false
}
```

### 2. **Hardcoded Default API Key**
**Location:** `.env.example:8`, `deployment/deploy-local.sh:22`
```bash
AI_API_KEY=br3nd4-69-ag3nt-k3y-d3f4ult
```
**Risk:** Default API key is publicly visible in repository. If not changed in production, provides unauthorized access.
**Impact:** Unauthorized agent authentication and game access.
**Fix:** 
- Remove default key from repository
- Generate unique keys per deployment
- Add validation to reject default key in production

### 3. **Missing Input Validation on Player Names**
**Location:** `pkg/session/manager.go:318-389` (handleLogin function)
**Risk:** No validation on player name length or character set.
**Impact:** Potential for extremely long names causing buffer issues or special characters breaking display/logging.
**Fix:** Add validation:
```go
func isValidPlayerName(name string) bool {
    if len(name) < 2 || len(name) > 32 {
        return false
    }
    // Allow alphanumeric and basic punctuation
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-\. ]+$`, name)
    return matched
}
```

## High Severity Findings

### 4. **No Password Hashing Implementation**
**Location:** `pkg/db/player.go`
**Observation:** The `PlayerRecord` struct has a `Password` field but no actual password authentication is implemented in the login flow.
**Risk:** If password authentication is added later without proper hashing, credentials could be exposed.
**Fix:** If implementing passwords, use bcrypt or Argon2 with salt:
```go
import "golang.org/x/crypto/bcrypt"

func hashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

### 5. **Lua Script Execution Without Sandboxing**
**Location:** `pkg/scripting/engine.go`
**Risk:** Lua scripts have access to full Lua standard libraries which could be abused.
**Impact:** Malicious scripts could cause denial of service or attempt to access filesystem.
**Fix:** Implement Lua sandboxing by restricting available libraries:
```go
// Instead of L.OpenLibs(), selectively open libraries
L.OpenLibs()
// Then remove dangerous functions
L.SetGlobal("dofile", lua.LNil)
L.SetGlobal("loadfile", lua.LNil)
L.SetGlobal("os", lua.LNil)
L.SetGlobal("io", lua.LNil)
L.SetGlobal("debug", lua.LNil)
```

## Medium Severity Findings

### 6. **Missing HTTPS Enforcement**
**Location:** `cmd/server/main.go:108` - Uses `http.ListenAndServe`
**Risk:** All traffic including authentication is transmitted in plaintext.
**Impact:** Credential interception, session hijacking.
**Fix:** Use TLS in production:
```go
err := http.ListenAndServeTLS(addr, "server.crt", "server.key", nil)
```

### 7. **No CSRF Protection for Web Endpoints**
**Location:** `web/middleware.go` - Web interface endpoints
**Risk:** Cross-Site Request Forgery attacks against web interface.
**Impact:** Unauthorized actions if web interface has admin functionality.
**Fix:** Implement CSRF tokens for state-changing operations.

### 8. **Insecure Default Database Credentials**
**Location:** `.env.example:5`, deployment scripts
```bash
POSTGRES_PASSWORD=postgres
```
**Risk:** Default database password is weak and publicly known.
**Impact:** Database compromise if not changed.
**Fix:** Generate strong random passwords for each deployment.

### 9. **Missing File Upload Validation**
**Location:** Web interface file serving (`web/middleware.go`)
**Risk:** Path traversal attacks if user input influences file paths.
**Observation:** Current implementation uses `filepath.Join` which is relatively safe, but no additional validation.
**Fix:** Add path validation:
```go
func safeServeFile(w http.ResponseWriter, r *http.Request, baseDir, filePath string) {
    // Clean and validate path
    cleanPath := filepath.Clean(filePath)
    if strings.Contains(cleanPath, "..") {
        http.Error(w, "Invalid path", http.StatusBadRequest)
        return
    }
    fullPath := filepath.Join(baseDir, cleanPath)
    http.ServeFile(w, r, fullPath)
}
```

## Low Severity Findings

### 10. **Verbose Error Messages**
**Location:** Various error logging throughout codebase
**Risk:** Potential information disclosure through error messages.
**Fix:** Sanitize error messages in production:
```go
// Instead of:
log.Printf("DB load error for %s: %v", login.PlayerName, err)
// Use:
log.Printf("DB load error for user: %v", sanitizeError(err))
```

### 11. **Missing Security Headers**
**Location:** Web endpoints lack security headers
**Risk:** Various web vulnerabilities (XSS, clickjacking, etc.)
**Fix:** Add security middleware:
```go
func securityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        next.ServeHTTP(w, r)
    })
}
```

### 12. **No Dependency Vulnerability Scanning**
**Risk:** Using dependencies with known vulnerabilities.
**Fix:** Integrate vulnerability scanning into CI/CD:
```bash
# Add to CI pipeline
go list -json -m all | nancy sleuth
# or
govulncheck ./...
```

## Positive Security Findings

### ✅ **SQL Injection Protection**
**Status:** GOOD - All database queries use parameterized queries
**Location:** `pkg/db/player.go` - Uses `QueryRow(query, params...)`

### ✅ **Rate Limiting Implemented**
**Status:** GOOD - Command rate limiting (10/sec) per session
**Location:** `pkg/session/manager.go:433-440`

### ✅ **Secure Agent Key Generation**
**Status:** GOOD - Uses crypto/rand for key generation, stores only SHA-256 hash
**Location:** `pkg/db/player.go:143-159`

### ✅ **WebSocket Ping/Pong Keepalive**
**Status:** GOOD - Proper connection management with timeouts
**Location:** `pkg/session/manager.go:274-292`

### ✅ **JSON Input Validation**
**Status:** GOOD - Proper JSON unmarshaling with error handling
**Location:** `pkg/session/manager.go:296-302`

## Immediate Action Items (Critical)

1. **Fix CORS configuration** - Restrict WebSocket origins
2. **Remove hardcoded credentials** - Generate unique keys per deployment
3. **Implement input validation** - Validate player names and commands
4. **Add password hashing** - If implementing password auth

## Recommended Security Improvements

1. **Implement TLS/HTTPS** for all production traffic
2. **Add security headers** to web endpoints
3. **Implement Lua sandboxing** for script execution
4. **Add dependency scanning** to CI pipeline
5. **Implement audit logging** for security events
6. **Add IP-based rate limiting** for connection attempts
7. **Implement session expiration** and reauthentication

## Testing Recommendations

1. **Penetration Testing:** Conduct focused testing on:
   - WebSocket endpoint security
   - Lua script injection attempts
   - Database query injection attempts
   - Rate limit bypass attempts

2. **Automated Security Scanning:**
   ```bash
   # Static analysis
   gosec ./...
   # Dependency scanning
   govulncheck ./...
   # SAST tools
   semgrep --config auto
   ```

3. **Manual Testing:**
   - Test with malformed JSON inputs
   - Attempt path traversal in file requests
   - Test command injection via Lua scripts
   - Verify rate limiting effectiveness

## Conclusion

The Dark Pawns codebase has a solid security foundation with proper use of parameterized queries, secure random number generation, and rate limiting. However, the critical CORS misconfiguration and hardcoded credentials pose significant risks that must be addressed before production deployment. Implementing the recommended fixes will significantly improve the security posture of the application.

**Overall Security Rating: 6.5/10** (With critical fixes applied: 8.5/10)