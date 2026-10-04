package session

import (
	"strings"
	"testing"
	"time"
)

// DP-1387 approves these Go-only protections; nanny has no equivalent
// per-IP bucket or shared IP failure counter (src/interpreter.c:1743-1938).
func TestEntryIPRateLimitApproved(t *testing.T) {
	database := entryDatabase(t)
	s := entrySession(t, database)
	s.request = nil
	s.SetRemoteIP("192.0.2.15")
	limiter := s.manager.loginLimiter.GetLimiter(s.RemoteIP())
	if limiter.Limit() != 5 || limiter.Burst() != 10 {
		t.Fatalf("approved bucket changed: rate=%v burst=%d", limiter.Limit(), limiter.Burst())
	}
	now := time.Now()
	if !limiter.AllowN(now, 10) || limiter.AllowN(now, 1) {
		t.Fatal("bucket must admit ten simultaneous requests and refuse the eleventh")
	}
	// Reserve a fresh burst in the future so scheduler delays cannot refill the
	// exhausted bucket before handleLogin's real Allow call. Thresholds stay intact.
	if reservation := limiter.ReserveN(now.Add(time.Hour), 10); !reservation.OK() {
		t.Fatal("could not reserve the approved burst")
	}
	if err := s.handleLogin(loginMsg("Freshhero", "")); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(s); !strings.Contains(got, "Too many login attempts. Please try again later.") || !s.SendClosed() || s.authenticated || s.charCreating {
		t.Fatalf("IP rate gate failed: %q closed=%v", got, s.SendClosed())
	}
	peer := makeCharSession(t, s.manager)
	peer.request = nil
	peer.SetRemoteIP("192.0.2.16")
	if err := peer.handleLogin(loginMsg("Otherhero", "")); err != nil {
		t.Fatal(err)
	}
	if peer.SendClosed() || peer.charStage != "confirm_name" {
		t.Fatal("one IP exhausted another IP's bucket")
	}
}

func TestEntryIPFailureLockoutApproved(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Aiko")
	entrySeed(t, database, "Beatrice")
	s := entrySession(t, database)
	s.request = nil
	s.SetRemoteIP("192.0.2.15")
	tracker := s.manager.loginAttempts
	// Prime nine failures, then require the real password path to record the
	// tenth. This isolates the IP lockout from the independently proven bucket.
	for range 9 {
		tracker.RecordFailure(s.RemoteIP())
	}
	if locked, _ := tracker.IsLocked(s.RemoteIP()); locked {
		t.Fatal("IP locked before ten failures")
	}
	before := time.Now()
	if err := s.handleLogin(loginMsg("Aiko", "wrongpass")); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(s); !strings.Contains(got, "Wrong password.") {
		t.Fatalf("tenth failure did not authenticate: %q", got)
	}
	locked, remaining := tracker.IsLocked(s.RemoteIP())
	if !locked || remaining > 15*time.Minute || remaining < 15*time.Minute-time.Since(before)-time.Second {
		t.Fatalf("approved ten-failure/fifteen-minute lockout changed: locked=%v remaining=%v", locked, remaining)
	}
	sameIP := makeCharSession(t, s.manager)
	sameIP.request = nil
	sameIP.SetRemoteIP(s.RemoteIP())
	// A different account with a valid password must still be refused by this IP.
	if err := sameIP.handleLogin(loginMsg("Beatrice", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(sameIP); !strings.Contains(got, "Too many failed login attempts. Try again in ") || !sameIP.SendClosed() || sameIP.authenticated || sameIP.player != nil {
		t.Fatalf("IP lockout bypassed: %q closed=%v", got, sameIP.SendClosed())
	}
	otherIP := makeCharSession(t, s.manager)
	otherIP.request = nil
	otherIP.SetRemoteIP("192.0.2.16")
	if err := otherIP.handleLogin(loginMsg("Beatrice", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	if !otherIP.authenticated || otherIP.SendClosed() {
		t.Fatal("IP lockout incorrectly locked the account across IPs")
	}
	if locked, _ := tracker.IsLocked(s.RemoteIP()); !locked {
		t.Fatal("another IP's successful login cleared this IP's lockout")
	}
}
