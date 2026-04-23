# Dark Pawns Security Hardening Guide

## Immediate Critical Fixes

### 1. Fix CORS/WebSocket Origin Validation

**File:** `pkg/session/manager.go`

**Replace lines 23-25:**
```go
CheckOrigin: func(r *http.Request) bool {
    return true // Allow all origins for development
}
```

**With:**
```go
CheckOrigin: func(r *http.Request) bool {
    // Development: allow all origins
    if os.Getenv("ENVIRONMENT") == "development" {
        return true
    }
    
    // Production: validate against allowed origins
    allowedOrigins := []string{
        "https://darkpawns.example.com",
        "https://game.darkpawns.example.com",
        // Add your production domains
    }
    
    origin := r.Header.Get("Origin")
    if origin == "" {
        // No Origin header, could be direct WebSocket connection
        // Allow but log for monitoring
        log.Printf("WebSocket connection without Origin header from %s", r.RemoteAddr)
        return true
    }
    
    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    
    log.Printf("Rejected WebSocket connection from unauthorized origin: %s", origin)
    return false
}
```

### 2. Remove Hardcoded Default API Key

**File:** `.env.example`

**Change line 8 from:**
```bash
AI_API_KEY=br3nd4-69-ag3nt-k3y-d3f4ult
```

**To:**
```bash
AI_API_KEY=REPLACE_WITH_SECURE_RANDOM_KEY
```

**File:** `deployment/deploy-local.sh`

**Change line 22 from:**
```bash
AI_API_KEY=br3nd4-69-ag3nt-k3y-d3f4ult
```

**To:**
```bash
# Generate a random key if not set
if [ -z "$AI_API_KEY" ]; then
    AI_API_KEY=$(openssl rand -hex 32)
    echo "Generated AI_API_KEY: $AI_API_KEY"
fi
```

**Add validation in `pkg/db/player.go` in `ValidateAgentKey` function:**
```go
func (db *DB) ValidateAgentKey(rawKey string) (characterName string, keyID int64, valid bool) {
    // Reject default/example keys
    if rawKey == "br3nd4-69-ag3nt-k3y-d3f4ult" || 
       strings.Contains(rawKey, "example") || 
       strings.Contains(rawKey, "test") {
        return "", 0, false
    }
    
    h := sha256.Sum256([]byte(rawKey))
    keyHash := hex.EncodeToString(h[:])

    err := db.conn.QueryRow(
        `SELECT id, character_name FROM agent_keys WHERE key_hash = $1 AND revoked = FALSE`,
        keyHash,
    ).Scan(&keyID, &characterName)
    if err != nil {
        return "", 0, false
    }
    return characterName, keyID, true
}
```

### 3. Implement Player Name Validation

**Create new file:** `pkg/validation/validation.go`

```go
package validation

import (
    "regexp"
    "strings"
    "unicode/utf8"
)

var (
    playerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\. ]+$`)
    maxPlayerNameLength = 32
    minPlayerNameLength = 2
)

func IsValidPlayerName(name string) bool {
    // Check length
    if utf8.RuneCountInString(name) < minPlayerNameLength || 
       utf8.RuneCountInString(name) > maxPlayerNameLength {
        return false
    }
    
    // Check character set
    if !playerNameRegex.MatchString(name) {
        return false
    }
    
    // Check for reserved names
    reservedNames := []string{"admin", "system", "root", "server", "null", "undefined"}
    lowerName := strings.ToLower(name)
    for _, reserved := range reservedNames {
        if lowerName == reserved {
            return false
        }
    }
    
    return true
}

func SanitizePlayerName(name string) string {
    // Remove invalid characters
    runes := []rune(name)
    var result []rune
    for _, r := range runes {
        if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
           (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' || r == ' ' {
            result = append(result, r)
        }
    }
    
    // Trim and limit length
    sanitized := string(result)
    if utf8.RuneCountInString(sanitized) > maxPlayerNameLength {
        sanitized = string([]rune(sanitized)[:maxPlayerNameLength])
    }
    
    return strings.TrimSpace(sanitized)
}
```

**Update `pkg/session/manager.go` handleLogin function:**

Add near line 318:
```go
import "github.com/zax0rz/darkpawns/pkg/validation"

// In handleLogin function, after parsing login data:
if !validation.IsValidPlayerName(login.PlayerName) {
    s.sendError("Invalid player name. Names must be 2-32 characters and contain only letters, numbers, spaces, dots, dashes, and underscores.")
    s.conn.Close()
    return nil
}
```

### 4. Implement Lua Sandboxing

**File:** `pkg/scripting/engine.go`

**Update NewEngine function (around line 30):**
```go
func NewEngine(scriptsDir string, world ScriptableWorld) *Engine {
    L := lua.NewState()
    engine := &Engine{
        scriptsDir:  scriptsDir,
        L:           L,
        world:       world,
        transitItems: make(map[int]ScriptableObject),
    }

    // Open safe libraries only
    L.OpenLibs()
    
    // Remove dangerous functions for security
    // Remove file system access
    L.SetGlobal("dofile", lua.LNil)
    L.SetGlobal("loadfile", lua.LNil)
    L.SetGlobal("load", lua.LNil)
    
    // Remove OS access
    osTable := L.GetGlobal("os").(*lua.LTable)
    osTable.RawSetString("execute", lua.LNil)
    osTable.RawSetString("exit", lua.LNil)
    osTable.RawSetString("remove", lua.LNil)
    osTable.RawSetString("rename", lua.LNil)
    osTable.RawSetString("setlocale", lua.LNil)
    osTable.RawSetString("tmpname", lua.LNil)
    
    // Remove package library (can load arbitrary code)
    L.SetGlobal("package", lua.LNil)
    
    // Remove debug library
    L.SetGlobal("debug", lua.LNil)
    
    // Remove io library
    L.SetGlobal("io", lua.LNil)
    
    // Set memory limit
    L.SetMx(1000) // Limit memory allocation
    
    // Register our custom functions
    engine.registerFunctions()

    // Load globals.lua
    engine.loadGlobals()

    return engine
}
```

### 5. Add Security Headers Middleware

**Create new file:** `pkg/web/security.go`

```go
package web

import (
    "net/http"
    "os"
)

func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Content Security Policy
        // Adjust these directives based on your actual requirements
        csp := "default-src 'self'; " +
               "script-src 'self' 'unsafe-inline'; " +
               "style-src 'self' 'unsafe-inline'; " +
               "img-src 'self' data:; " +
               "connect-src 'self' ws: wss:; " +
               "font-src 'self'; " +
               "object-src 'none'; " +
               "media-src 'self'; " +
               "frame-src 'none'; " +
               "frame-ancestors 'none';"
        
        w.Header().Set("Content-Security-Policy", csp)
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        // HSTS - Only in production with HTTPS
        if os.Getenv("ENVIRONMENT") == "production" {
            w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        }
        
        // Permissions Policy
        w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
        
        next.ServeHTTP(w, r)
    })
}
```

**Update `cmd/server/main.go` to use middleware:**

```go
import (
    // ... existing imports
    "github.com/zax0rz/darkpawns/pkg/web"
)

// In main function, replace:
// if err := http.ListenAndServe(addr, nil); err != nil {

// With:
handler := web.SecurityHeaders(http.DefaultServeMux)
if err := http.ListenAndServe(addr, handler); err != nil {
```

### 6. Implement TLS/HTTPS Support

**Update `cmd/server/main.go`:**

```go
// Add TLS configuration
func startServer(addr string, useTLS bool) error {
    if useTLS {
        certFile := os.Getenv("TLS_CERT_FILE")
        keyFile := os.Getenv("TLS_KEY_FILE")
        
        if certFile == "" || keyFile == "" {
            log.Fatal("TLS_CERT_FILE and TLS_KEY_FILE environment variables must be set for TLS")
        }
        
        log.Printf("Starting HTTPS server on %s", addr)
        return http.ListenAndServeTLS(addr, certFile, keyFile, nil)
    } else {
        log.Printf("Starting HTTP server on %s (WARNING: Not secure for production)", addr)
        return http.ListenAndServe(addr, nil)
    }
}

// In main function, replace the server startup:
useTLS := os.Getenv("USE_TLS") == "true"
if err := startServer(addr, useTLS); err != nil {
    log.Fatalf("Server error: %v", err)
}
```

### 7. Add Rate Limiting for Login Attempts

**Create new file:** `pkg/auth/ratelimit.go`

```go
package auth

import (
    "net"
    "sync"
    "time"
    
    "golang.org/x/time/rate"
)

type IPRateLimiter struct {
    ips map[string]*rate.Limiter
    mu  sync.RWMutex
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
    return &IPRateLimiter{
        ips: make(map[string]*rate.Limiter),
    }
}

func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
    i.mu.Lock()
    defer i.mu.Unlock()
    
    limiter := rate.NewLimiter(rate.Limit(5), 10) // 5 requests per second, burst of 10
    i.ips[ip] = limiter
    
    // Cleanup old entries (optional)
    go func() {
        time.Sleep(5 * time.Minute)
        i.mu.Lock()
        delete(i.ips, ip)
        i.mu.Unlock()
    }()
    
    return limiter
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
    i.mu.Lock()
    defer i.mu.Unlock()
    
    limiter, exists := i.ips[ip]
    if !exists {
        return i.AddIP(ip)
    }
    
    return limiter
}

func GetIPFromRequest(r *http.Request) string {
    // Get IP from X-Forwarded-For header if behind proxy
    if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
        // Take the first IP in the list
        if ips := strings.Split(forwarded, ","); len(ips) > 0 {
            return strings.TrimSpace(ips[0])
        }
    }
    
    // Fall back to RemoteAddr
    host, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return host
}
```

**Update login handler to use IP rate limiting:**

```go
// Add to session manager
type Manager struct {
    mu            sync.RWMutex
    sessions      map[string]*Session
    world         *game.World
    combatEngine  *combat.CombatEngine
    db            db.DB
    hasDB         bool
    loginLimiter  *auth.IPRateLimiter // Add this
}

// In NewManager:
m := &Manager{
    sessions:     make(map[string]*Session),
    world:        world,
    combatEngine: ce,
    loginLimiter: auth.NewIPRateLimiter(rate.Limit(5), 10), // 5 logins per second per IP
}

// In handleLogin, add at beginning:
ip := auth.GetIPFromRequest(s.conn.Request())
if !s.manager.loginLimiter.GetLimiter(ip).Allow() {
    s.sendError("Too many login attempts. Please try again later.")
    s.conn.Close()
    return nil
}
```

### 8. Add Audit Logging

**Create new file:** `pkg/audit/logger.go`

```go
package audit

import (
    "encoding/json"
    "log"
    "os"
    "time"
)

type AuditEvent struct {
    Timestamp   time.Time `json:"timestamp"`
    EventType   string    `json:"event_type"`
    User        string    `json:"user,omitempty"`
    IPAddress   string    `json:"ip_address,omitempty"`
    Action      string    `json:"action"`
    Details     string    `json:"details,omitempty"`
    Success     bool      `json:"success"`
}

type AuditLogger struct {
    file *os.File
}

func NewAuditLogger(filename string) (*AuditLogger, error) {
    file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    
    return &AuditLogger{file: file}, nil
}

func (a *AuditLogger) Log(event AuditEvent) {
    event.Timestamp = time.Now()
    
    data, err := json.Marshal(event)
    if err != nil {
        log.Printf("Failed to marshal audit event: %v", err)
        return
    }
    
    a.file.Write(append(data, '\n'))
    
    // Also log to console for important events
    if !event.Success || event.EventType == "security" {
        log.Printf("[AUDIT] %s: %s (User: %s, IP: %s)", 
            event.EventType, event.Action, event.User, event.IPAddress)
    }
}

func (a *AuditLogger) Close() {
    a.file.Close()
}

// Convenience functions
func LogLoginAttempt(user, ip string, success bool) {
    event := AuditEvent{
        EventType: "authentication",
        User:      user,
        IPAddress: ip,
        Action:    "login_attempt",
        Success:   success,
    }
    
    if !success {
        event.Details = "Failed login attempt"
    }
    
    // Get global audit logger and log event
    // Implementation depends on how you structure your app
}
```

### 9. Update Deployment Scripts for Security

**Update `deployment/deploy-k8s.sh`:**

Add password strength validation:
```bash
validate_password() {
    local password="$1"
    local min_length=12
    
    if [ ${#password} -lt $min_length ]; then
        echo "Password must be at least $min_length characters"
        return 1
    fi
    
    # Check for complexity
    if ! [[ "$password" =~ [A-Z] ]] || ! [[ "$password" =~ [a-z] ]] || ! [[ "$password" =~ [0-9] ]]; then
        echo "Password must contain uppercase, lowercase, and numbers"
        return 1
    fi
    
    return 0
}

# In the password prompt section:
read -sp "PostgreSQL Password: " postgres_password
echo
if ! validate_password "$postgres_password"; then
    echo "Invalid password. Please try again."
    exit 1
fi
```

### 10. Create Security Configuration File

**Create new file:** `config/security.yaml`

```yaml
# Security Configuration
security:
  # WebSocket/CORS
  allowed_origins:
    - "https://darkpawns.example.com"
    - "https://game.darkpawns.example.com"
  
  # Rate limiting
  rate_limits:
    commands_per_second: 10
    login_attempts_per_minute: 5
    ip_ban_after_failed_logins: 10
    ip_ban_duration_minutes: 15
  
  # Session management
  session:
    timeout_minutes: 60
    renewal_threshold_minutes: 15
    max_sessions_per_user: 3
  
  # Password policy (if implemented)
  password_policy:
    min_length: 12
    require_uppercase: true
    require_lowercase: true
    require_numbers: true
    require_special_chars: true
    max_age_days: 90
  
  # Audit logging
  audit:
    enabled: true
    log_file: "/var/log/darkpawns/audit.log"
    retention_days: 90
    log_level: "info"
  
  # TLS/SSL
  tls:
    enabled: true
    min_version: "TLS1.2"
    cipher_suites:
      - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
      - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
  
  # Headers
  headers:
    hsts_max_age: 31536000
    csp_enabled: true
    x_frame_options: "DENY"
```

## Testing Your Security Fixes

### 1. Run Security Tests

Create `tests/security/test_security.py`:

```python
#!/usr/bin/env python3
import requests
import websocket
import json
import time

def test_cors_origin():
    """Test that unauthorized origins are rejected"""
    print("Testing CORS origin validation...")
    
    # Try to connect from unauthorized origin
    ws = websocket.WebSocket()
    try:
        ws.connect("ws://localhost:8080/ws", 
                  header={"Origin": "https://evil.com"})
        print("❌ FAIL: Connected from unauthorized origin")
        return False
    except:
        print("✅ PASS: Rejected unauthorized origin")
        return True

def test_rate_limiting():
    """Test command rate limiting"""
    print("\nTesting rate limiting...")
    
    # Connect as normal user
    ws = websocket.WebSocket()
    ws.connect("ws://localhost:8080/ws")
    
    # Login
    ws.send(json.dumps({
        "type": "login",
        "data": {"player_name": "testuser"}
    }))
    
    # Send commands rapidly
    commands_sent = 0
    rate_limited = False
    
    for i in range(15):  # More than rate limit
        ws.send(json.dumps({
            "type": "command",
            "data": {"command": "look"}
        }))
        commands_sent += 1
        
        # Check response
        try:
            response = ws.recv()
            data = json.loads(response)
            if data.get("type") == "error" and "rate limit" in data.get("data", {}).get("message", ""):
                rate_limited = True
                print(f"✅ Rate limited after {commands_sent} commands")
                break
        except:
            pass
        
        time.sleep(0.05)  # 20 commands per second
    
    ws.close()
    
    if not rate_limited:
        print("❌ FAIL: No rate limiting detected")
        return False
    
    return True

def test_sql_injection():
    """Test for SQL injection vulnerabilities"""
    print("\nTesting SQL injection...")
    
    # Test player names with SQL injection attempts
    test_names = [
        "test'; DROP TABLE players; --",
        "admin' OR '1'='1",
        "test\" OR 1=1 --",
    ]
    
    for name in test_names:
        ws = websocket.WebSocket()
        ws.connect("ws://localhost:8080/ws")
        
        ws.send(json.dumps({
            "type": "login",
            "data": {"player_name": name}
        }))
        
        try:
            response = ws.recv()
            data = json.loads(response)
            if data.get("type") == "error":
                print(f"✅ Rejected SQL injection attempt: {name[:20]}...")
            else:
                print(f"❌ FAIL: Accepted potentially dangerous name: {name[:20]}...")
                ws.close()
                return False
        except:
            print(f"✅ Connection closed for dangerous name: {name[:20]}...")
        
        ws.close()
    
    return True

if __name__ == "__main__":
    print("Running security tests...")
    
    tests = [
        test_cors_origin,
        test_rate_limiting,
        test_sql_injection,
    ]
    
    passed = 0
    total = len(tests)
    
    for test in tests:
        try:
            if test():
                passed += 1
        except Exception as e:
            print(f"❌ Test failed with error: {e}")
    
    print(f"\nSecurity tests: {passed}/{total} passed")
    exit(0 if passed == total else 1)
```

### 2. Add Security Tests to CI/CD

**Update `.github/workflows/ci.yml`:**

```yaml
name: Security Tests
on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      
      - name: Run security scanner
        run: |
          # Install security tools
          go install github.com/securego/gosec/v2/cmd/gosec@latest
          go install golang.org/x/vuln/cmd/govulncheck@latest
          
          # Run gosec
          gosec ./...
          
          # Run vulnerability check
          govulncheck ./...
      
      - name: Run custom security tests
        run: |
          cd tests/security
          python3 test_security.py
```

## Monitoring and Maintenance

### 1. Regular Security Updates

Create `scripts/security-update.sh`:

```bash
#!/bin/bash
# Weekly security update script

echo "Running security updates..."

# Update Go dependencies
go get -u ./...
go mod tidy

# Run security scanners
gosec ./...
govulncheck ./...

# Check for leaked secrets
if command -v trufflehog &> /dev/null; then
    trufflehog filesystem . --no-verification
fi

# Update audit log retention
find /var/log/darkpawns/audit.log* -mtime +90 -delete

echo "Security updates complete"
```

### 2. Security Incident Response Plan

Create `docs/SECURITY_INCIDENT_RESPONSE.md`:

```markdown
# Security Incident Response Plan

## 1. Identification
- Monitor audit logs for suspicious activity
- Watch for failed login attempts
- Monitor rate limit triggers
- Check for unusual Lua script behavior

## 2. Containment
- Immediately block offending IP addresses
- Revoke compromised API keys
- Temporarily disable affected features
- Increase logging level

## 3. Eradication
- Identify root cause
- Apply security patches
- Rotate all credentials
- Update firewall rules

## 4. Recovery
- Restore from clean backup if needed
- Verify system integrity
- Gradually re-enable features
- Monitor for recurrence

## 5. Lessons Learned
- Document incident
- Update security procedures
- Train team members
- Improve monitoring
```

## Conclusion

Implementing these security hardening measures will significantly improve the security posture of Dark Pawns. Start with the critical fixes (CORS, hardcoded credentials, input validation), then progressively implement the other recommendations based on your deployment timeline and risk assessment.

Remember to:
1. **Test all changes** in a staging environment first
2. **Monitor logs** after deployment
3. **Keep dependencies updated** regularly
4. **Conduct periodic security reviews** (quarterly recommended)
5. **Stay informed** about new security threats and best practices