package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

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
		// Clear old entries - in a simple implementation, just keep recent ones
		// Since we don't track timestamps per entry, a simple approach is
		// to not clean up aggressively (entries naturally age via rate.Limiter)
		// or keep a simple max map size check
		if len(i.ips) > 10000 {
			// If map gets very large, rebuild with recent entries
			// For now, just log and keep - rate.Limiter is lightweight
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
