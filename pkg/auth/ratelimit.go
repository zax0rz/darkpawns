package auth

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// LoginAttemptConfig controls lockout thresholds.
type LoginAttemptConfig struct {
	// MaxAttempts is the number of consecutive failed login attempts before
	// the IP is temporarily locked out.
	MaxAttempts int
	// LockoutDuration is how long the IP remains locked out after MaxAttempts failures.
	LockoutDuration time.Duration
	// WindowDuration is the sliding window over which failed attempts are counted.
	// Attempts older than this are pruned.
	WindowDuration time.Duration
}

// DefaultLoginAttemptConfig returns sensible production defaults.
func DefaultLoginAttemptConfig() LoginAttemptConfig {
	return LoginAttemptConfig{
		MaxAttempts:     10,
		LockoutDuration: 15 * time.Minute,
		WindowDuration:  15 * time.Minute,
	}
}

// LoginAttemptTracker tracks failed login attempts per IP and enforces
// temporary lockouts after too many consecutive failures (H-15).
type LoginAttemptTracker struct {
	mu      sync.Mutex
	attempts map[string]*loginAttempts // keyed by IP
	cfg     LoginAttemptConfig
}

type loginAttempts struct {
	failures   []time.Time
	lockedUntil time.Time
}

// NewLoginAttemptTracker creates a tracker with the given config.
func NewLoginAttemptTracker(cfg LoginAttemptConfig) *LoginAttemptTracker {
	t := &LoginAttemptTracker{
		attempts: make(map[string]*loginAttempts),
		cfg:     cfg,
	}
	go t.cleanupLoop()
	return t
}

// cleanupLoop periodically prunes expired entries to prevent unbounded growth.
func (t *LoginAttemptTracker) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		t.mu.Lock()
		cutoff := time.Now().Add(-t.cfg.WindowDuration)
		for ip, a := range t.attempts {
			// Remove if no recent failures and not currently locked
			if a.lockedUntil.Before(cutoff) && len(a.failures) == 0 {
				delete(t.attempts, ip)
			}
		}
		t.mu.Unlock()
	}
}

// RecordFailure records a failed login attempt for the given IP and returns
// the current failure count (including this one). If the count exceeds
// MaxAttempts, the IP is locked out.
func (t *LoginAttemptTracker) RecordFailure(ip string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	a, exists := t.attempts[ip]
	if !exists {
		a = &loginAttempts{}
		t.attempts[ip] = a
	}

	// If currently locked out, don't add new failures (they're irrelevant)
	if now.Before(a.lockedUntil) {
		return len(a.failures)
	}

	// Prune old failures outside the window
	cutoff := now.Add(-t.cfg.WindowDuration)
	pruned := a.failures[:0]
	for _, ts := range a.failures {
		if ts.After(cutoff) {
			pruned = append(pruned, ts)
		}
	}
	a.failures = pruned

	// Add the new failure
	a.failures = append(a.failures, now)

	// Check if lockout threshold is reached
	if len(a.failures) >= t.cfg.MaxAttempts {
		a.lockedUntil = now.Add(t.cfg.LockoutDuration)
		slog.Warn("IP locked out: too many failed login attempts",
			"ip", ip,
			"failures", len(a.failures),
			"locked_until", a.lockedUntil,
		)
	}

	return len(a.failures)
}

// RecordSuccess clears the failure history for the given IP on a successful login.
func (t *LoginAttemptTracker) RecordSuccess(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempts, ip)
}

// IsLocked returns true if the IP is currently locked out, and the remaining duration.
func (t *LoginAttemptTracker) IsLocked(ip string) (bool, time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	a, exists := t.attempts[ip]
	if !exists {
		return false, 0
	}

	if time.Now().Before(a.lockedUntil) {
		return true, time.Until(a.lockedUntil)
	}

	return false, 0
}

// RemainingAttempts returns how many more failures are allowed before lockout.
// Returns 0 if already locked.
func (t *LoginAttemptTracker) RemainingAttempts(ip string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	a, exists := t.attempts[ip]
	if !exists {
		return t.cfg.MaxAttempts
	}

	// Prune old failures
	now := time.Now()
	cutoff := now.Add(-t.cfg.WindowDuration)
	pruned := a.failures[:0]
	for _, ts := range a.failures {
		if ts.After(cutoff) {
			pruned = append(pruned, ts)
		}
	}
	a.failures = pruned

	if now.Before(a.lockedUntil) {
		return 0
	}

	remaining := t.cfg.MaxAttempts - len(a.failures)
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// IPRateLimiter provides per-IP rate limiting using token buckets.
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.RWMutex
}

func NewIPRateLimiter() *IPRateLimiter {
	rl := &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
	}
	go rl.cleanupLoop()
	return rl
}

func (i *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		i.mu.Lock()
		if len(i.ips) > 10000 {
			i.ips = make(map[string]*rate.Limiter)
		}
		i.mu.Unlock()
	}
}

func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := rate.NewLimiter(rate.Limit(5), 10) // 5 requests per second, burst of 10
	i.ips[ip] = limiter

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

// ---------------------------------------------------------------------------
// IP extraction — H-12 fix: use TCP RemoteAddr, not spoofable headers
// ---------------------------------------------------------------------------

// trustedProxies contains IPs (or CIDRs) that are allowed to set
// X-Forwarded-For. If the direct connection does not come from one of
// these, the header is ignored and the TCP remote address is used.
var trustedProxies []string

// SetTrustedProxies sets the list of trusted proxy IPs/CIDRs. Only
// connections from these IPs will have X-Forwarded-For respected.
// Call this during server initialization (before serving requests).
func SetTrustedProxies(proxies []string) {
	trustedProxies = proxies
}

// GetIPFromRequest returns the client IP from the request. It uses the
// TCP RemoteAddr by default and only trusts X-Forwarded-For when the
// direct connection originates from a configured trusted proxy (H-12).
func GetIPFromRequest(r *http.Request) string {
	remoteIP := remoteAddrIP(r)

	// Only consider X-Forwarded-For if the request came from a trusted proxy
	if !isTrustedProxy(remoteIP) {
		return remoteIP
	}

	// Trusted proxy — use the rightmost non-trusted IP in X-Forwarded-For
	// (the standard practice: each proxy appends on the right, so the
	// leftmost is the original client, but we want the rightmost untrusted)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ips := strings.Split(forwarded, ",")
		// Walk right-to-left, skip trusted proxies
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			if ip == "" {
				continue
			}
			if !isTrustedProxy(ip) {
				return ip
			}
		}
		// All entries are trusted proxies — use the leftmost (original client)
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	return remoteIP
}

// remoteAddrIP extracts the IP from r.RemoteAddr (the actual TCP connection).
func remoteAddrIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isTrustedProxy checks if an IP is in the trusted proxy list.
// Supports plain IPs and CIDR notation.
func isTrustedProxy(ip string) bool {
	if len(trustedProxies) == 0 {
		return false
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	for _, proxy := range trustedProxies {
		if strings.Contains(proxy, "/") {
			// CIDR notation
			_, network, err := net.ParseCIDR(proxy)
			if err != nil {
				continue
			}
			if network.Contains(parsedIP) {
				return true
			}
		} else {
			// Plain IP
			if proxy == ip {
				return true
			}
		}
	}
	return false
}
