package auth

import (
	"net/http"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// H-12: GetIPFromRequest — must use TCP RemoteAddr unless from trusted proxy
// ---------------------------------------------------------------------------

func TestGetIPFromRequest_DirectConnection_IgnoresForwardedFor(t *testing.T) {
	// No trusted proxies configured — X-Forwarded-For must be ignored
	SetTrustedProxies(nil)

	req := &http.Request{
		RemoteAddr: "203.0.113.50:43211",
		Header:     http.Header{"X-Forwarded-For": {"1.2.3.4, 10.0.0.1"}},
	}

	ip := GetIPFromRequest(req)
	if ip != "203.0.113.50" {
		t.Errorf("expected TCP remote addr, got %s", ip)
	}
}

func TestGetIPFromRequest_TrustedProxy_ExtractsClientIP(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.0/8"})

	req := &http.Request{
		RemoteAddr: "10.0.0.1:12345",
		Header:     http.Header{"X-Forwarded-For": {"203.0.113.50, 10.0.0.1"}},
	}

	ip := GetIPFromRequest(req)
	if ip != "203.0.113.50" {
		t.Errorf("expected client IP from X-Forwarded-For, got %s", ip)
	}
}

func TestGetIPFromRequest_TrustedProxy_ChainedProxies(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12"})

	req := &http.Request{
		RemoteAddr: "10.0.0.2:54321",
		Header:     http.Header{"X-Forwarded-For": {"203.0.113.50, 172.16.0.1, 10.0.0.2"}},
	}

	ip := GetIPFromRequest(req)
	if ip != "203.0.113.50" {
		t.Errorf("expected original client IP, got %s", ip)
	}
}

func TestGetIPFromRequest_TrustedProxy_AllTrusted(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.0/8"})

	req := &http.Request{
		RemoteAddr: "10.0.0.3:44444",
		Header:     http.Header{"X-Forwarded-For": {"10.0.0.1, 10.0.0.2, 10.0.0.3"}},
	}

	ip := GetIPFromRequest(req)
	if ip != "10.0.0.1" {
		t.Errorf("expected leftmost IP when all are trusted, got %s", ip)
	}
}

func TestGetIPFromRequest_TrustedProxy_NoForwardedHeader(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.0/8"})

	req := &http.Request{
		RemoteAddr: "10.0.0.1:12345",
	}

	ip := GetIPFromRequest(req)
	if ip != "10.0.0.1" {
		t.Errorf("expected proxy IP when no X-Forwarded-For, got %s", ip)
	}
}

func TestGetIPFromRequest_UntrustedProxy_IgnoresForwarded(t *testing.T) {
	SetTrustedProxies([]string{"10.0.0.1"})

	req := &http.Request{
		RemoteAddr: "203.0.113.50:9999",
		Header:     http.Header{"X-Forwarded-For": {"192.168.1.1"}},
	}

	ip := GetIPFromRequest(req)
	if ip != "203.0.113.50" {
		t.Errorf("expected TCP remote addr (attacker IP), got %s", ip)
	}
}

func TestGetIPFromRequest_SpoofDifferentIPsPerRequest(t *testing.T) {
	SetTrustedProxies(nil)

	attackerRealIP := "203.0.113.50"

	for _, spoofed := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"} {
		req := &http.Request{
			RemoteAddr: attackerRealIP + ":12345",
			Header:     http.Header{"X-Forwarded-For": {spoofed}},
		}

		ip := GetIPFromRequest(req)
		if ip != attackerRealIP {
			t.Errorf("spoof bypass: expected %s, got %s", attackerRealIP, ip)
		}
	}
}

func TestIsTrustedProxy(t *testing.T) {
	tests := []struct {
		ip       string
		proxies  []string
		expected bool
	}{
		{"10.0.0.1", []string{"10.0.0.1"}, true},
		{"10.0.0.5", []string{"10.0.0.0/24"}, true},
		{"10.0.1.1", []string{"10.0.0.0/24"}, false},
		{"192.168.1.1", []string{"10.0.0.0/8"}, false},
		{"1.2.3.4", nil, false},
		{"1.2.3.4", []string{}, false},
		{"172.16.0.5", []string{"10.0.0.0/8", "172.16.0.0/12"}, true},
	}

	for _, tt := range tests {
		SetTrustedProxies(tt.proxies)
		got := isTrustedProxy(tt.ip)
		if got != tt.expected {
			t.Errorf("isTrustedProxy(%q, %v) = %v, want %v", tt.ip, tt.proxies, got, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// H-15: LoginAttemptTracker — lockout after N failed attempts
// ---------------------------------------------------------------------------

func TestLoginAttemptTracker_NoLockout_UnderThreshold(t *testing.T) {
	tracker := NewLoginAttemptTracker(LoginAttemptConfig{
		MaxAttempts:     5,
		LockoutDuration: 15 * time.Minute,
		WindowDuration:  15 * time.Minute,
	})

	for i := 0; i < 4; i++ {
		locked, _ := tracker.IsLocked("1.2.3.4")
		if locked {
			t.Fatalf("should not be locked after %d failures", i+1)
		}
		tracker.RecordFailure("1.2.3.4")
	}

	locked, _ := tracker.IsLocked("1.2.3.4")
	if locked {
		t.Error("should not be locked at 4 failures with threshold 5")
	}

	tracker.RecordFailure("1.2.3.4")
	locked, remaining := tracker.IsLocked("1.2.3.4")
	if !locked {
		t.Error("should be locked after 5 failures")
	}
	if remaining <= 0 {
		t.Error("remaining duration should be positive")
	}
}

func TestLoginAttemptTracker_LockoutExpires(t *testing.T) {
	cfg := LoginAttemptConfig{
		MaxAttempts:     3,
		LockoutDuration: 100 * time.Millisecond,
		WindowDuration:  1 * time.Second,
	}
	tracker := NewLoginAttemptTracker(cfg)

	for i := 0; i < 3; i++ {
		tracker.RecordFailure("1.2.3.4")
	}

	locked, _ := tracker.IsLocked("1.2.3.4")
	if !locked {
		t.Error("should be locked")
	}

	time.Sleep(150 * time.Millisecond)

	locked, _ = tracker.IsLocked("1.2.3.4")
	if locked {
		t.Error("lockout should have expired")
	}
}

func TestLoginAttemptTracker_SuccessClearsFailures(t *testing.T) {
	tracker := NewLoginAttemptTracker(LoginAttemptConfig{
		MaxAttempts:     5,
		LockoutDuration: 15 * time.Minute,
		WindowDuration:  15 * time.Minute,
	})

	for i := 0; i < 3; i++ {
		tracker.RecordFailure("1.2.3.4")
	}

	if tracker.RemainingAttempts("1.2.3.4") != 2 {
		t.Error("should have 2 remaining attempts")
	}

	tracker.RecordSuccess("1.2.3.4")

	if tracker.RemainingAttempts("1.2.3.4") != 5 {
		t.Error("should have 5 remaining after success")
	}

	locked, _ := tracker.IsLocked("1.2.3.4")
	if locked {
		t.Error("should not be locked after success clears history")
	}
}

func TestLoginAttemptTracker_DifferentIPsAreIndependent(t *testing.T) {
	tracker := NewLoginAttemptTracker(LoginAttemptConfig{
		MaxAttempts:     3,
		LockoutDuration: 15 * time.Minute,
		WindowDuration:  15 * time.Minute,
	})

	for i := 0; i < 3; i++ {
		tracker.RecordFailure("1.1.1.1")
	}

	locked, _ := tracker.IsLocked("2.2.2.2")
	if locked {
		t.Error("IP2 should not be locked")
	}

	locked, _ = tracker.IsLocked("1.1.1.1")
	if !locked {
		t.Error("IP1 should be locked")
	}
}

func TestLoginAttemptTracker_RemainingAttempts(t *testing.T) {
	tracker := NewLoginAttemptTracker(LoginAttemptConfig{
		MaxAttempts:     5,
		LockoutDuration: 15 * time.Minute,
		WindowDuration:  15 * time.Minute,
	})

	if tracker.RemainingAttempts("1.2.3.4") != 5 {
		t.Error("new IP should have max remaining attempts")
	}

	tracker.RecordFailure("1.2.3.4")
	if tracker.RemainingAttempts("1.2.3.4") != 4 {
		t.Error("should have 4 remaining after 1 failure")
	}

	for i := 0; i < 4; i++ {
		tracker.RecordFailure("1.2.3.4")
	}
	if tracker.RemainingAttempts("1.2.3.4") != 0 {
		t.Error("should have 0 remaining when locked out")
	}
}

// ---------------------------------------------------------------------------
// remoteAddrIP — edge cases
// ---------------------------------------------------------------------------

func TestRemoteAddrIP(t *testing.T) {
	tests := []struct {
		remoteAddr string
		expected   string
	}{
		{"192.168.1.1:12345", "192.168.1.1"},
		{"[::1]:8080", "::1"},
		{"10.0.0.1", "10.0.0.1"},
		{"bad input", "bad input"},
	}

	for _, tt := range tests {
		req := &http.Request{RemoteAddr: tt.remoteAddr}
		got := remoteAddrIP(req)
		if got != tt.expected {
			t.Errorf("remoteAddrIP(%q) = %q, want %q", tt.remoteAddr, got, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// CIDR edge cases
// ---------------------------------------------------------------------------

func TestIsTrustedProxy_CIDR_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		cidr     string
		expected bool
	}{
		{"exact match", "10.0.0.1", "10.0.0.1/32", true},
		{"in range", "10.0.0.127", "10.0.0.0/24", true},
		{"just outside", "10.0.1.0", "10.0.0.0/24", false},
		{"single host", "192.168.1.1", "192.168.1.1", true},
		{"different host", "192.168.1.2", "192.168.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetTrustedProxies([]string{tt.cidr})
			got := isTrustedProxy(tt.ip)
			if got != tt.expected {
				t.Errorf("isTrustedProxy(%s in %s) = %v, want %v", tt.ip, tt.cidr, got, tt.expected)
			}
		})
	}
}
