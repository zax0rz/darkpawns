package auth

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
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
//
// Valid CIDRs are always applied even when some entries are invalid; any
// invalid CIDR strings are collected and reported via the returned error so a
// misconfiguration is not silently swallowed. Callers that only want to log
// can ignore the error — the valid entries still take effect.
//
// Only the first call takes effect (sync.Once): subsequent calls are no-ops
// and return nil. This matches an init-time configuration model; to reconfigure
// in a test, reset trustedProxiesOnce directly.
func SetTrustedProxies(cidrs []string) error {
	var invalid []string
	trustedProxiesOnce.Do(func() {
		if len(cidrs) == 0 {
			return
		}
		trustedProxies = make([]*net.IPNet, 0, len(cidrs))
		for _, c := range cidrs {
			_, network, err := net.ParseCIDR(c)
			if err != nil {
				slog.Warn("skipping invalid trusted proxy CIDR", "cidr", c, "error", err)
				invalid = append(invalid, c)
				continue
			}
			trustedProxies = append(trustedProxies, network)
		}
	})
	if len(invalid) > 0 {
		return fmt.Errorf("invalid trusted proxy CIDR(s): %s", strings.Join(invalid, ", "))
	}
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

// DefaultTrustedProxies are trusted when the deployment names none: the
// loopback networks, where a reverse proxy on the same machine (Caddy in the
// production topology) connects from. Only processes on the machine itself
// can connect from loopback, so trusting their X-Forwarded-For lets nobody
// else choose an address.
var DefaultTrustedProxies = []string{"127.0.0.0/8", "::1/128"}

// GetIPFromRequest extracts the client IP from an HTTP request.
//
// SECURITY (H-12): X-Forwarded-For is only consulted when the direct
// connection comes from a configured trusted proxy (via SetTrustedProxies);
// otherwise the TCP RemoteAddr is the client, so a client cannot choose its
// own address to slip past bans and rate limits.
//
// Behind trusted proxies the header is read from the right. Each trusted
// proxy vouches for the address to its left, so the client is the rightmost
// address that is not a trusted proxy. The leftmost entry is whatever the
// client sent: a proxy that appends to a client-supplied header would let
// anyone pick their address if that entry were believed.
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
	if len(trustedProxies) == 0 || !isTrustedProxy(remoteIP) {
		return host
	}
	forwarded := r.Header.Values("X-Forwarded-For")
	hops := strings.Split(strings.Join(forwarded, ","), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(hops[i])
		if hop == "" {
			continue
		}
		ip := net.ParseIP(hop)
		if ip == nil {
			// A malformed entry breaks the chain of vouching: nothing to its
			// left can be believed, so stop at the last proxy that spoke.
			return host
		}
		if !isTrustedProxy(ip) {
			return ip.String()
		}
		host = ip.String()
	}
	// Every hop was a trusted proxy: the innermost one is as far as the
	// header can be followed.
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
			i.cleanupOnce()
		case <-i.stopCh:
			return
		}
	}
}

// cleanupOnce removes up to maxDeletions stale limiters. A limiter is stale
// when it has regained its full burst, meaning it has not been used recently.
func (i *IPRateLimiter) cleanupOnce() {
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
}

func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	// L2-020: distinct addresses grow the map without bound; once past the
	// cap, evict arbitrary entries until back under it (sync.Map has no
	// recency; re-added IPs simply get fresh budgets).
	size := 0
	i.ips.Range(func(_, _ any) bool { size++; return true })
	if size > maxTrackedIPs {
		i.ips.Range(func(key, _ any) bool {
			if size <= maxTrackedIPs {
				return false
			}
			i.ips.Delete(key)
			size--
			return true
		})
	}
	limiter := rate.NewLimiter(rate.Limit(5), 10) // 5 requests per second, burst of 10
	actual, _ := i.ips.LoadOrStore(ip, limiter)
	return actual.(*rate.Limiter)
}

// maxTrackedIPs caps the per-IP limiter map (L2-020). Eviction is arbitrary
// (sync.Map has no recency); re-added IPs simply get fresh budgets.
const maxTrackedIPs = 50000

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
		// L2-029: distinct failing addresses grow the map without bound;
		// beyond the cap, drop the stalest entries (lowest failure counts
		// first are wrong — oldest lastFailAt is the safe eviction).
		if len(t.attempts) >= maxTrackedLoginIPs {
			t.evictStaleLoginIPs()
		}
		a = &loginAttempts{}
		t.attempts[ip] = a
	}
	a.failures++
	a.lastFailAt = time.Now()
}

// maxTrackedLoginIPs caps the failed-login map (L2-029).
const maxTrackedLoginIPs = 10000

func (t *LoginAttemptTracker) evictStaleLoginIPs() {
	type aged struct {
		ip       string
		lastFail time.Time
	}
	entries := make([]aged, 0, len(t.attempts))
	for ip, a := range t.attempts {
		entries = append(entries, aged{ip, a.lastFailAt})
	}
	sort.Slice(entries, func(x, y int) bool { return entries[x].lastFail.Before(entries[y].lastFail) })
	for _, e := range entries {
		if len(t.attempts) < maxTrackedLoginIPs {
			return
		}
		delete(t.attempts, e.ip)
	}
}

// FailAndCheck records a failure and reports atomically whether the attempt
// may still be answered (L2-024): checking IsLocked and recording in
// separate critical sections let concurrent guesses all pass the check
// before any of them was counted. A failed return means the source is now
// locked and the response must be the locked refusal, not the wrong-password
// one.
func (t *LoginAttemptTracker) FailAndCheck(ip string) (allowed bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	a, ok := t.attempts[ip]
	if !ok {
		a = &loginAttempts{}
		t.attempts[ip] = a
	}
	if a.failures >= t.threshold && t.lockout > time.Since(a.lastFailAt) {
		a.lastFailAt = time.Now()
		return false
	}
	a.failures++
	a.lastFailAt = time.Now()
	return true
}

// accountBudget is the configurable per-account failure budget (L2-004).
// Distributed guesses rotate source IPs to escape the per-IP budget, but a
// hard per-name lockout is itself an attack on known accounts — the
// renewable per-account lockout was removed for exactly that (2c3e269e4,
// Zach's Option D), and C has no account lockout at all. The budget is
// therefore OFF by default: set DP_ACCOUNT_FAILURE_BUDGET to a failure
// count (and optionally DP_ACCOUNT_FAILURE_WINDOW as a duration, default
// 1h) to bound total wrong passwords against one name across all sources.
type accountBudget struct {
	failures  int
	firstSeen time.Time
}

// AccountBudgetTracker bounds failures per account name across all source
// IPs. A zero threshold disables it entirely.
type AccountBudgetTracker struct {
	mu        sync.Mutex
	threshold int
	window    time.Duration
	accounts  map[string]*accountBudget
}

// NewAccountBudgetTracker builds the tracker; threshold <= 0 disables it.
func NewAccountBudgetTracker(threshold int, window time.Duration) *AccountBudgetTracker {
	if window <= 0 {
		window = time.Hour
	}
	return &AccountBudgetTracker{threshold: threshold, window: window, accounts: map[string]*accountBudget{}}
}

// AccountBudgetFromEnvironment reads the opt-in configuration.
func AccountBudgetFromEnvironment() *AccountBudgetTracker {
	count, _ := strconv.Atoi(os.Getenv("DP_ACCOUNT_FAILURE_BUDGET"))
	if count <= 0 {
		return NewAccountBudgetTracker(0, 0)
	}
	window, err := time.ParseDuration(os.Getenv("DP_ACCOUNT_FAILURE_WINDOW"))
	if err != nil || window <= 0 {
		window = time.Hour
	}
	return NewAccountBudgetTracker(count, window)
}

// Fail records a failure against an account and reports whether the attempt
// may still be answered. Disabled trackers always allow.
func (t *AccountBudgetTracker) Fail(account string) (allowed bool) {
	if t.threshold <= 0 {
		return true
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	a, ok := t.accounts[account]
	if !ok || time.Since(a.firstSeen) > t.window {
		a = &accountBudget{firstSeen: time.Now()}
		t.accounts[account] = a
	}
	a.failures++
	return a.failures < t.threshold
}

// RecordSuccess clears an account's budget usage after a successful login.
func (t *AccountBudgetTracker) RecordSuccess(account string) {
	if t.threshold <= 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.accounts, account)
}

// RecordSuccess resets the failure counter for an IP.
func (t *LoginAttemptTracker) RecordSuccess(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.attempts, ip)
}
