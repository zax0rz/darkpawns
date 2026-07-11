package auth

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestIPRateLimiter_CleanupDeletesAllStale(t *testing.T) {
	rl := NewIPRateLimiter()
	t.Cleanup(func() { rl.Stop() })

	const totalIPs = 10050
	for i := 0; i < totalIPs; i++ {
		rl.AddIP(fmt.Sprintf("192.168.0.%d", i))
	}

	// All limiters are stale (full burst, never used). Each cleanup pass deletes
	// at most maxDeletions entries, so keep running until the map is empty.
	for passes := 0; passes < 25; passes++ {
		rl.cleanupOnce()
		empty := true
		rl.ips.Range(func(key, value any) bool {
			empty = false
			return false
		})
		if empty {
			return
		}
	}

	t.Fatalf("expected all %d stale limiters to be deleted within cleanup passes", totalIPs)
}

func TestIPRateLimiter_CleanupPreservesActive(t *testing.T) {
	rl := NewIPRateLimiter()
	t.Cleanup(func() { rl.Stop() })

	activeIP := "10.0.0.1"
	staleIP := "10.0.0.2"

	active := rl.GetLimiter(activeIP)
	for i := 0; i < 15; i++ {
		active.Allow()
	}

	rl.AddIP(staleIP)

	rl.cleanupOnce()

	if _, ok := rl.ips.Load(activeIP); !ok {
		t.Error("expected active limiter to be preserved")
	}
	if _, ok := rl.ips.Load(staleIP); ok {
		t.Error("expected stale limiter to be deleted")
	}
}

func TestSetTrustedProxies_SkipsInvalidCIDR(t *testing.T) {
	resetTrustedProxies()

	// Mix of valid and invalid CIDRs; valid ones are applied even when some
	// are invalid, and the invalid ones are reported via the returned error.
	err := SetTrustedProxies([]string{"not-a-cidr", "10.0.0.0/8", "bad"})
	if err == nil {
		t.Fatal("expected a non-nil error reporting the invalid CIDRs")
	}
	if !strings.Contains(err.Error(), "not-a-cidr") || !strings.Contains(err.Error(), "bad") {
		t.Errorf("error %q should name both invalid CIDRs", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:43210"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")

	ip := GetIPFromRequest(req)
	if ip != "203.0.113.50" {
		t.Errorf("expected valid CIDR to be trusted, got %q", ip)
	}
}

func TestSetTrustedProxies_AllValidNoError(t *testing.T) {
	resetTrustedProxies()

	// All-valid input must return nil and apply the CIDRs.
	err := SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12"})
	if err != nil {
		t.Fatalf("expected nil error for all-valid CIDRs, got %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.16.5.5:43210"
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	if ip := GetIPFromRequest(req); ip != "203.0.113.99" {
		t.Errorf("expected valid CIDR to be trusted, got %q", ip)
	}
}

func TestSetTrustedProxies_OnlyOnce(t *testing.T) {
	resetTrustedProxies()

	// First call applies; second call is a no-op and returns nil (init model).
	if err := SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
		t.Fatalf("first call: %v", err)
	}
	// A second call with invalid CIDRs must NOT error (it never runs).
	if err := SetTrustedProxies([]string{"not-a-cidr"}); err != nil {
		t.Errorf("second SetTrustedProxies call should be a no-op, got error: %v", err)
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

func TestLoginAttemptTracker_StopTwice(t *testing.T) {
	tracker := NewLoginAttemptTracker(LoginAttemptConfig{})

	// Calling Stop() multiple times should not panic
	tracker.Stop()
	tracker.Stop()
}

// ---------------------------------------------------------------------------
// AccountLockoutTracker (DP-592)
// ---------------------------------------------------------------------------

type mockAccountLockoutStore struct {
	mu       sync.Mutex
	attempts map[string]int
	locked   map[string]time.Time
}

func newMockAccountLockoutStore() *mockAccountLockoutStore {
	return &mockAccountLockoutStore{
		attempts: make(map[string]int),
		locked:   make(map[string]time.Time),
	}
}

func (m *mockAccountLockoutStore) GetAccountLockout(name string) (int, *time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	until, ok := m.locked[name]
	if !ok {
		return m.attempts[name], nil, nil
	}
	return m.attempts[name], &until, nil
}

func (m *mockAccountLockoutStore) RecordLoginFailure(name string, threshold int, lockoutDuration time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts[name]++
	if m.attempts[name] >= threshold {
		until := time.Now().Add(lockoutDuration)
		m.locked[name] = until
		return true, nil
	}
	return false, nil
}

func (m *mockAccountLockoutStore) RecordLoginSuccess(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.attempts, name)
	delete(m.locked, name)
	return nil
}

func TestAccountLockoutTracker_NotLockedInitially(t *testing.T) {
	store := newMockAccountLockoutStore()
	tracker := NewAccountLockoutTracker(store, AccountLockoutConfig{
		Threshold: 3,
		Lockout:   1 * time.Minute,
	})

	locked, _ := tracker.IsLocked("alice")
	if locked {
		t.Fatal("account should not be locked initially")
	}
}

func TestAccountLockoutTracker_LocksAtThreshold(t *testing.T) {
	store := newMockAccountLockoutStore()
	tracker := NewAccountLockoutTracker(store, AccountLockoutConfig{
		Threshold: 3,
		Lockout:   1 * time.Minute,
	})

	tracker.RecordFailure("alice")
	tracker.RecordFailure("alice")
	if locked, _ := tracker.IsLocked("alice"); locked {
		t.Fatal("account should not be locked below threshold")
	}

	newlyLocked := tracker.RecordFailure("alice")
	if !newlyLocked {
		t.Fatal("expected account to become locked at threshold")
	}

	locked, remaining := tracker.IsLocked("alice")
	if !locked {
		t.Fatal("account should be locked after threshold")
	}
	if remaining <= 0 || remaining > 1*time.Minute {
		t.Fatalf("unexpected remaining lockout: %v", remaining)
	}
}

func TestAccountLockoutTracker_SuccessResets(t *testing.T) {
	store := newMockAccountLockoutStore()
	tracker := NewAccountLockoutTracker(store, AccountLockoutConfig{
		Threshold: 3,
		Lockout:   1 * time.Minute,
	})

	tracker.RecordFailure("alice")
	tracker.RecordFailure("alice")
	tracker.RecordSuccess("alice")

	if locked, _ := tracker.IsLocked("alice"); locked {
		t.Fatal("account should not be locked after success")
	}
}

func TestAccountLockoutTracker_DefaultConfig(t *testing.T) {
	tracker := NewAccountLockoutTracker(newMockAccountLockoutStore(), AccountLockoutConfig{})
	if tracker.threshold != 10 {
		t.Errorf("default threshold should be 10, got %d", tracker.threshold)
	}
	if tracker.lockout != 15*time.Minute {
		t.Errorf("default lockout should be 15m, got %v", tracker.lockout)
	}
}

func TestAccountLockoutTracker_StoreErrorFailsSecure(t *testing.T) {
	store := &failingAccountLockoutStore{}
	tracker := NewAccountLockoutTracker(store, AccountLockoutConfig{
		Threshold: 3,
		Lockout:   1 * time.Minute,
	})

	locked, _ := tracker.IsLocked("alice")
	if !locked {
		t.Fatal("expected fail-secure behavior on store error")
	}
}

type failingAccountLockoutStore struct{}

func (f *failingAccountLockoutStore) GetAccountLockout(string) (int, *time.Time, error) {
	return 0, nil, errors.New("store unavailable")
}

func (f *failingAccountLockoutStore) RecordLoginFailure(string, int, time.Duration) (bool, error) {
	return false, errors.New("store unavailable")
}

func (f *failingAccountLockoutStore) RecordLoginSuccess(string) error {
	return errors.New("store unavailable")
}

// Note: resetTrustedProxies requires sync import
var _ sync.Once
