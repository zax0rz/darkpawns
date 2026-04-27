package auth

import (
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
		// If map gets very large, rebuild with recent entries
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
	close(t.stop)
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
		if a.failures >= t.threshold {
			// Keep locked-out entries until lockout expires
			if now.Sub(a.lastFailAt) > t.lockout {
				delete(t.attempts, ip)
			}
		} else {
			// Non-locked entries: expire after lockout duration of inactivity
			if now.Sub(a.lastFailAt) > t.lockout {
				delete(t.attempts, ip)
			}
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
