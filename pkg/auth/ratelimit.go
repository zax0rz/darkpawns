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