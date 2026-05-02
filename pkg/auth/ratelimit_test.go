package auth

import (
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// H-12: GetIPFromRequest
// ---------------------------------------------------------------------------

func resetTrustedProxies() {
	trustedProxiesOnce = sync.Once{}
	trustedProxies = nil
}

func TestGetIPFromRequest_RemoteAddr(t *testing.T) {
	resetTrustedProxies()

	// No trusted proxies configured — should always use RemoteAddr
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.50:12345"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	ip := GetIPFromRequest(req)
	if ip != "203.0.113.50" {
		t.Errorf("expected RemoteAddr IP, got %q", ip)
	}
}

func TestGetIPFromRequest_TrustedProxy(t *testing.T) {
	resetTrustedProxies()

	// Configure the server's own network (e.g. Docker bridge) as trusted
	_ = SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:43210"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")

	ip := GetIPFromRequest(req)
	if ip != "203.0.113.50" {
		t.Errorf("expected X-Forwarded-For IP from trusted proxy, got %q", ip)
	}
}

func TestGetIPFromRequest_TrustedProxy_MultipleForwards(t *testing.T) {
	resetTrustedProxies()
	_ = SetTrustedProxies([]string{"10.0.0.0/8"})

	// Multiple proxies: first entry is the original client
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:43210"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.3, 172.16.0.1")

	ip := GetIPFromRequest(req)
	if ip != "203.0.113.50" {
		t.Errorf("expected leftmost X-Forwarded-For IP, got %q", ip)
	}
}

func TestGetIPFromRequest_UntrustedProxy_IgnoresHeader(t *testing.T) {
	resetTrustedProxies()
	_ = SetTrustedProxies([]string{"10.0.0.0/8"})

	// Connection from an untrusted IP — should ignore X-Forwarded-For
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "198.51.100.75:9999"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	ip := GetIPFromRequest(req)
	if ip != "198.51.100.75" {
		t.Errorf("expected RemoteAddr from untrusted proxy, got %q", ip)
	}
}

func TestGetIPFromRequest_NoHeader(t *testing.T) {
	resetTrustedProxies()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.50:12345"

	ip := GetIPFromRequest(req)
	if ip != "203.0.113.50" {
		t.Errorf("expected RemoteAddr when no X-Forwarded-For, got %q", ip)
	}
}

func TestIsTrustedProxy(t *testing.T) {
	resetTrustedProxies()
	_ = SetTrustedProxies([]string{"10.0.0.0/8", "192.168.1.0/24"})

	tests := []struct {
		ip   string
		want bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"192.168.1.100", true},
		{"192.168.1.0", true},
		{"192.168.2.1", false},
		{"203.0.113.50", false},
		{"8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			got := isTrustedProxy(ip)
			if got != tt.want {
				t.Errorf("isTrustedProxy(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// H-15: LoginAttemptTracker
// ---------------------------------------------------------------------------

func TestLoginAttemptTracker_NotLockedInitially(t *testing.T) {
	tracker := NewLoginAttemptTracker(LoginAttemptConfig{
		Threshold: 3,
		Lockout:   1 * time.Minute,
	})
	defer tracker.Stop()

	locked, remaining := tracker.IsLocked("1.2.3.4")
	if locked {
		t.Error("new IP should not be locked")
	}
	if remaining != 0 {
		t.Errorf("expected 0 remaining, got %v", remaining)
	}
}

func TestLoginAttemptTracker_LockoutAfterThreshold(t *testing.T) {
	tracker := NewLoginAttemptTracker(LoginAttemptConfig{
		Threshold: 3,
		Lockout:   10 * time.Minute,
	})
	defer tracker.Stop()

	ip := "10.0.0.1"
	for i := 0; i < 3; i++ {
		tracker.RecordFailure(ip)
	}

	locked, remaining := tracker.IsLocked(ip)
	if !locked {
		t.Error("should be locked after 3 failures")
	}
	if remaining <= 0 {
		t.Error("remaining should be positive")
	}
	if remaining > 10*time.Minute {
		t.Errorf("remaining %v exceeds lockout duration", remaining)
	}
}

func TestLoginAttemptTracker_NotLockedBelowThreshold(t *testing.T) {
	tracker := NewLoginAttemptTracker(LoginAttemptConfig{
		Threshold: 5,
		Lockout:   10 * time.Minute,
	})
	defer tracker.Stop()

	ip := "10.0.0.2"
	for i := 0; i < 4; i++ {
		tracker.RecordFailure(ip)
	}

	locked, _ := tracker.IsLocked(ip)
	if locked {
		t.Error("should not be locked below threshold")
	}
}

func TestLoginAttemptTracker_RecordSuccessResets(t *testing.T) {
	tracker := NewLoginAttemptTracker(LoginAttemptConfig{
		Threshold: 3,
		Lockout:   10 * time.Minute,
	})
	defer tracker.Stop()

	ip := "10.0.0.3"
	tracker.RecordFailure(ip)
	tracker.RecordFailure(ip)
	tracker.RecordSuccess(ip)

	locked, _ := tracker.IsLocked(ip)
	if locked {
		t.Error("should not be locked after success resets counter")
	}
}

func TestLoginAttemptTracker_LockoutExpires(t *testing.T) {
	tracker := NewLoginAttemptTracker(LoginAttemptConfig{
		Threshold: 2,
		Lockout:   50 * time.Millisecond,
	})
	defer tracker.Stop()

	ip := "10.0.0.4"
	tracker.RecordFailure(ip)
	tracker.RecordFailure(ip)

	locked, _ := tracker.IsLocked(ip)
	if !locked {
		t.Error("should be locked immediately")
	}

	// Wait for lockout to expire
	time.Sleep(100 * time.Millisecond)

	locked, remaining := tracker.IsLocked(ip)
	if locked {
		t.Errorf("should be unlocked after expiry, remaining=%v", remaining)
	}
}

func TestLoginAttemptTracker_IndependentIPs(t *testing.T) {
	tracker := NewLoginAttemptTracker(LoginAttemptConfig{
		Threshold: 2,
		Lockout:   10 * time.Minute,
	})
	defer tracker.Stop()

	tracker.RecordFailure("10.0.0.1")
	tracker.RecordFailure("10.0.0.1")

	locked, _ := tracker.IsLocked("10.0.0.1")
	if !locked {
		t.Error("10.0.0.1 should be locked")
	}

	locked, _ = tracker.IsLocked("10.0.0.2")
	if locked {
		t.Error("10.0.0.2 should not be locked")
	}
}

func TestLoginAttemptTracker_DefaultConfig(t *testing.T) {
	// Ensure defaults are applied when config is zero-value
	tracker := NewLoginAttemptTracker(LoginAttemptConfig{})
	defer tracker.Stop()

	if tracker.threshold != 10 {
		t.Errorf("default threshold should be 10, got %d", tracker.threshold)
	}
	if tracker.lockout != 15*time.Minute {
		t.Errorf("default lockout should be 15m, got %v", tracker.lockout)
	}
}

// ---------------------------------------------------------------------------
// IPRateLimiter (existing coverage)
// ---------------------------------------------------------------------------

func TestIPRateLimiter_BasicRateLimit(t *testing.T) {
	rl := NewIPRateLimiter()
	t.Cleanup(func() { rl.Stop() })

	limiter := rl.GetLimiter("192.168.1.1")

	allowed := 0
	for i := 0; i < 15; i++ {
		if limiter.Allow() {
			allowed++
		}
	}

	if allowed < 10 {
		t.Errorf("expected at least burst=10 allowed, got %d", allowed)
	}
	if allowed > 10 {
		// Some may leak through due to token refill, but shouldn't be >> burst
		t.Logf("allowed %d (burst=10, tokens refill over time)", allowed)
	}
}

func TestIPRateLimiter_DifferentIPs(t *testing.T) {
	rl := NewIPRateLimiter()
	t.Cleanup(func() { rl.Stop() })

	l1 := rl.GetLimiter("10.0.0.1")
	l2 := rl.GetLimiter("10.0.0.2")

	if l1 == nil || l2 == nil {
		t.Error("limiters should not be nil")
	}

	// Drain one limiter; the other should be unaffected
	for i := 0; i < 15; i++ {
		l1.Allow()
	}

	allowed := 0
	for i := 0; i < 10; i++ {
		if l2.Allow() {
			allowed++
		}
	}

	if allowed < 10 {
		t.Errorf("separate IP limiter should have full burst, got %d", allowed)
	}
}

// Benchmark for GetIPFromRequest with trusted proxy
func BenchmarkGetIPFromRequest_TrustedProxy(b *testing.B) {
	resetTrustedProxies()
	_ = SetTrustedProxies([]string{"10.0.0.0/8"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:43210"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetIPFromRequest(req)
	}
}

// Benchmark for GetIPFromRequest without trusted proxy (default path)
func BenchmarkGetIPFromRequest_NoTrustedProxy(b *testing.B) {
	resetTrustedProxies()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.50:12345"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetIPFromRequest(req)
	}
}

// Note: resetTrustedProxies requires sync import
var _ sync.Once
