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

// ---------------------------------------------------------------------------
// IP extraction (H-12)
// ---------------------------------------------------------------------------

// trustedProxies holds CIDRs of proxies whose X-Forwarded-For we trust.
// Populated once at init via SetTrustedProxies; never mutated after that.
var (
	trustedProxies     []*net.IPNet
	trustedProxiesOnce sync.Once
)

// SetTrustedProxies parses a slice of CIDR strings and stores them as trusted
// proxy networks.  Must be called before any request handling (typically in
// main / server setup).  An empty or nil slice means "trust nothing" and
// effectively disables X-Forwarded-For processing.
func SetTrustedProxies(cidrs []string) error {
	trustedProxiesOnce.Do(func() {
		if len(cidrs) == 0 {
			return
		}
		trustedProxies = make([]*net.IPNet, 0, len(cidrs))
		for _, c := range cidrs {
			_, network, err := net.ParseCIDR(c)
			if err != nil {
				// skip bad entries, log in real init
				continue
			}
			trustedProxies = append(trustedProxies, network)
		}
	})
	return nil
}

// isTrustedProxy returns true when the IP belongs to a configured trusted
// proxy network.
func isTrustedProxy(ip net.IP) bool {
	for _, network := range trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// GetIPFromRequest extracts the client IP from an HTTP request.
//
// SECURITY (H-12): By default this uses the TCP RemoteAddr (r.RemoteAddr)
// and does NOT trust X-Forwarded-For, preventing clients from spoofing their
// source IP to bypass rate limits.  X-Forwarded-For is only consulted when
// the direct connection comes from a configured trusted proxy (via
// SetTrustedProxies).
func GetIPFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	// Parse RemoteAddr to a net.IP for proxy check.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	remoteIP := net.ParseIP(host)
	if remoteIP == nil {
		return host // best effort
	}

	// Only trust X-Forwarded-For if the direct connection is from a trusted proxy.
	if len(trustedProxies) > 0 && isTrustedProxy(remoteIP) {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			// Take the first (leftmost) IP — the original client.
			if ips := strings.Split(forwarded, ","); len(ips) > 0 {
				clientIP := strings.TrimSpace(ips[0])
				if clientIP != "" {
					return clientIP
				}
			}
		}
	}

	return host
}

// ---------------------------------------------------------------------------
// IP rate limiter
// ---------------------------------------------------------------------------

// IPRateLimiter provides per-IP token-bucket rate limiting.
type IPRateLimiter struct {
	ips    sync.Map // string -> *rate.Limiter
	stopCh chan struct{}
	once   sync.Once
}

func NewIPRateLimiter() *IPRateLimiter {
	rl := &IPRateLimiter{
		stopCh: make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

// Stop terminates the background cleanup goroutine. Safe to call multiple times.
func (i *IPRateLimiter) Stop() {
	i.once.Do(func() { close(i.stopCh) })
}

func (i *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// Remove stale entries individually rather than nuking the entire map.
			// The old approach (replace with empty map) let any IP with a full
			// burst get a fresh limiter and bypass the rate limit window.
			count := 0
			i.ips.Range(func(key, value any) bool {
				count++
				limiter := value.(*rate.Limiter)
				if count > 10000 && limiter.Tokens() >= float64(limiter.Burst()) {
					i.ips.Delete(key)
				}
				return true
			})
		case <-i.stopCh:
			return
		}
	}
}

func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	limiter := rate.NewLimiter(rate.Limit(5), 10) // 5 requests per second, burst of 10
	actual, _ := i.ips.LoadOrStore(ip, limiter)
	return actual.(*rate.Limiter)
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	actual, ok := i.ips.Load(ip)
	if ok {
		return actual.(*rate.Limiter)
	}
	return i.AddIP(ip)
}

// ---------------------------------------------------------------------------
// Login attempt lockout (H-15)
// ---------------------------------------------------------------------------

// LoginAttemptTracker tracks failed login attempts per IP and locks out
// IPs that exceed a configurable failure threshold.
type LoginAttemptTracker struct {
	mu        sync.Mutex
	attempts  map[string]*loginAttempts
	threshold int           // failures before lockout
	lockout   time.Duration // how long the lockout lasts
	stop      chan struct{}
	once      sync.Once
}

// loginAttempts holds the failure count and the time the last failure was
// recorded (used for calculating remaining lockout duration).
type loginAttempts struct {
	failures   int
	lastFailAt time.Time
}

// LoginAttemptConfig holds configuration for the tracker.
type LoginAttemptConfig struct {
	Threshold int           // failures before lockout (default 10)
	Lockout   time.Duration // lockout duration (default 15 minutes)
}

// NewLoginAttemptTracker creates a tracker and starts a background goroutine
// that periodically purges expired entries.
func NewLoginAttemptTracker(cfg LoginAttemptConfig) *LoginAttemptTracker {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 10
	}
	if cfg.Lockout <= 0 {
		cfg.Lockout = 15 * time.Minute
	}

	t := &LoginAttemptTracker{
		attempts:  make(map[string]*loginAttempts),
		threshold: cfg.Threshold,
		lockout:   cfg.Lockout,
		stop:      make(chan struct{}),
	}
	go t.cleanupLoop()
	return t
}

// Stop terminates the background cleanup goroutine.
func (t *LoginAttemptTracker) Stop() {
	t.once.Do(func() { close(t.stop) })
}

func (t *LoginAttemptTracker) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.purgeExpired()
		case <-t.stop:
			return
		}
	}
}

func (t *LoginAttemptTracker) purgeExpired() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	for ip, a := range t.attempts {
		if now.Sub(a.lastFailAt) > t.lockout {
			delete(t.attempts, ip)
		}
	}
}

// IsLocked reports whether the given IP is currently locked out.
// If locked, it returns the remaining lockout duration.
func (t *LoginAttemptTracker) IsLocked(ip string) (bool, time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	a, ok := t.attempts[ip]
	if !ok || a.failures < t.threshold {
		return false, 0
	}

	remaining := t.lockout - time.Since(a.lastFailAt)
	if remaining <= 0 {
		// Lockout expired, reset
		delete(t.attempts, ip)
		return false, 0
	}
	return true, remaining
}

// RecordFailure increments the failure counter for an IP.
func (t *LoginAttemptTracker) RecordFailure(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	a, ok := t.attempts[ip]
	if !ok {
		a = &loginAttempts{}
		t.attempts[ip] = a
	}
	a.failures++
	a.lastFailAt = time.Now()
}

// RecordSuccess resets the failure counter for an IP.
func (t *LoginAttemptTracker) RecordSuccess(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.attempts, ip)
}

// ---------------------------------------------------------------------------
// Account-level lockout (DP-592)
// ---------------------------------------------------------------------------

// AccountLockoutStore is the persistence interface required by
// AccountLockoutTracker. It is satisfied by db.Database.
type AccountLockoutStore interface {
	GetAccountLockout(name string) (failedAttempts int, lockedUntil *time.Time, err error)
	RecordLoginFailure(name string, threshold int, lockoutDuration time.Duration) (bool, error)
	RecordLoginSuccess(name string) error
}

// AccountLockoutTracker tracks failed login attempts per account and locks
// accounts that exceed a configurable threshold. State is persisted via the
// supplied store so lockouts survive process restarts.
type AccountLockoutTracker struct {
	mu        sync.Mutex
	store     AccountLockoutStore
	threshold int
	lockout   time.Duration
	cache     map[string]*accountLockoutEntry
}

// accountLockoutEntry holds cached lockout state for a single account.
type accountLockoutEntry struct {
	failedAttempts int
	lockedUntil    time.Time
}

// AccountLockoutConfig holds configuration for the tracker.
type AccountLockoutConfig struct {
	Threshold int
	Lockout   time.Duration
}

// NewAccountLockoutTracker creates a tracker backed by store.
func NewAccountLockoutTracker(store AccountLockoutStore, cfg AccountLockoutConfig) *AccountLockoutTracker {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 10
	}
	if cfg.Lockout <= 0 {
		cfg.Lockout = 15 * time.Minute
	}
	return &AccountLockoutTracker{
		store:     store,
		threshold: cfg.Threshold,
		lockout:   cfg.Lockout,
		cache:     make(map[string]*accountLockoutEntry),
	}
}

// IsLocked reports whether the account is currently locked out. If locked, it
// returns the remaining lockout duration.
func (t *AccountLockoutTracker) IsLocked(name string) (bool, time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	entry, ok := t.cache[name]
	if !ok {
		// Cache miss: load from persistent store.
		attempts, lockedUntil, err := t.store.GetAccountLockout(name)
		if err != nil {
			// On store error, fail secure: treat as locked for the configured
			// duration so a broken DB does not open the door to brute force.
			slog.Error("account lockout store error", "name", name, "error", err)
			return true, t.lockout
		}
		if lockedUntil == nil {
			return false, 0
		}
		entry = &accountLockoutEntry{
			failedAttempts: attempts,
			lockedUntil:    *lockedUntil,
		}
		t.cache[name] = entry
	}

	remaining := time.Until(entry.lockedUntil)
	if remaining <= 0 {
		delete(t.cache, name)
		return false, 0
	}
	return true, remaining
}

// RecordFailure records a failed login attempt for the account. It returns
// true when the account becomes locked as a result.
func (t *AccountLockoutTracker) RecordFailure(name string) bool {
	locked, err := t.store.RecordLoginFailure(name, t.threshold, t.lockout)
	if err != nil {
		slog.Error("failed to record login failure", "name", name, "error", err)
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	// Refresh cache from store on next IsLocked; avoid duplicating threshold
	// logic here by invalidating the cached entry.
	delete(t.cache, name)

	return locked
}

// Lockout returns the configured lockout duration.
func (t *AccountLockoutTracker) Lockout() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lockout
}

// RecordSuccess clears the lockout state for the account.
func (t *AccountLockoutTracker) RecordSuccess(name string) {
	if err := t.store.RecordLoginSuccess(name); err != nil {
		slog.Error("failed to record login success", "name", name, "error", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.cache, name)
}
