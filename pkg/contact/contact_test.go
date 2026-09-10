package contact

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeVerifier struct{ err error }

func (v fakeVerifier) Verify(context.Context, string, string) error { return v.err }

type fakeSender struct {
	form submission
	err  error
}

func (s *fakeSender) Send(form submission) error {
	s.form = form
	return s.err
}

func TestHandlerDeliversValidSubmission(t *testing.T) {
	delivery := &fakeSender{}
	handler := &Handler{verify: fakeVerifier{}, send: delivery, limit: newRateLimiter(5, time.Hour)}
	body := `{"character":"Aidan","years":"2001 to 2005","email":"player@example.com","message":"I played Dark Pawns and remember the old crew.","website":"","turnstile":"valid"}`
	request := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if delivery.form.Email != "player@example.com" {
		t.Fatalf("delivered email = %q", delivery.form.Email)
	}
}

func TestHandlerRejectsFailures(t *testing.T) {
	tests := []struct {
		name     string
		verify   error
		send     error
		body     string
		wantCode int
	}{
		{"invalid email", nil, nil, `{"email":"nope","message":"This message is long enough to pass.","turnstile":"valid"}`, http.StatusBadRequest},
		{"turnstile failure", errors.New("no"), nil, `{"email":"player@example.com","message":"This message is long enough to pass.","turnstile":"bad"}`, http.StatusBadRequest},
		{"delivery failure", nil, errors.New("smtp failed"), `{"email":"player@example.com","message":"This message is long enough to pass.","turnstile":"valid"}`, http.StatusBadGateway},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := &Handler{verify: fakeVerifier{err: test.verify}, send: &fakeSender{err: test.send}, limit: newRateLimiter(5, time.Hour)}
			request := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantCode {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.wantCode, response.Body.String())
			}
		})
	}
}

func TestHandlerTreatsHoneypotAsSuccess(t *testing.T) {
	delivery := &fakeSender{}
	handler := &Handler{verify: fakeVerifier{err: errors.New("must not run")}, send: delivery, limit: newRateLimiter(1, time.Hour)}
	body := `{"email":"bot@example.com","message":"This message is long enough to pass.","website":"https://spam.example"}`
	request := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || delivery.form.Email != "" {
		t.Fatal("honeypot submission was not silently discarded")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := newRateLimiter(2, time.Hour)
	now := time.Unix(1000, 0)
	first := limiter.Allow("ip", now)
	second := limiter.Allow("ip", now)
	third := limiter.Allow("ip", now)
	if !first || !second || third {
		t.Fatal("rate limit did not stop the third request")
	}
	if !limiter.Allow("ip", now.Add(2*time.Hour)) {
		t.Fatal("rate limit did not expire")
	}
}

func TestTurnstileResultRequiresSuccessActionAndHostname(t *testing.T) {
	verifier := &turnstileVerifier{
		expectedAction:    "contact",
		expectedHostnames: map[string]struct{}{"darkpawns.org": {}},
	}
	tests := []struct {
		name   string
		result turnstileResult
		want   bool
	}{
		{"valid", turnstileResult{Success: true, Action: "contact", Hostname: "darkpawns.org"}, true},
		{"failed", turnstileResult{Success: false, Action: "contact", Hostname: "darkpawns.org"}, false},
		{"wrong action", turnstileResult{Success: true, Action: "login", Hostname: "darkpawns.org"}, false},
		{"wrong hostname", turnstileResult{Success: true, Action: "contact", Hostname: "localhost"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := verifier.accepts(test.result); got != test.want {
				t.Fatalf("accepts() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestRateLimiterReclaimsExpiredKeysAtCapacity(t *testing.T) {
	const cap = 5
	limiter := newRateLimiterWithCapacity(1, time.Hour, cap)
	oldTime := time.Now().Add(-2 * time.Hour)

	// Fill table to capacity with expired timestamps
	for i := 0; i < cap; i++ {
		key := fmt.Sprintf("ip-%d", i)
		if !limiter.Allow(key, oldTime) {
			t.Fatalf("failed to insert initial key %s", key)
		}
	}

	if len(limiter.entries) != cap {
		t.Fatalf("expected %d entries, got %d", cap, len(limiter.entries))
	}

	// Brand new key at now should succeed by sweeping the expired entries
	now := time.Now()
	if !limiter.Allow("new-ip", now) {
		t.Fatal("Allow on brand new key failed after capacity reached with expired keys")
	}

	// After sweep, old expired keys are gone and only new-ip is tracked
	if len(limiter.entries) != 1 {
		t.Fatalf("expected 1 entry after sweep, got %d", len(limiter.entries))
	}

	// Second request within limit should be blocked
	if limiter.Allow("new-ip", now) {
		t.Fatal("second request for new-ip should have been rate limited")
	}

	// If capacity is filled with active keys, brand new key should be rejected
	for i := 0; i < cap; i++ {
		key := fmt.Sprintf("active-ip-%d", i)
		limiter.Allow(key, now)
	}
	if limiter.Allow("overflow-ip", now) {
		t.Fatal("new key should be rejected when active keys fill capacity")
	}
}

func TestTurnstileFailureDoesNotConsumeRateLimit(t *testing.T) {
	delivery := &fakeSender{}
	limiter := newRateLimiter(1, time.Hour)
	handler := &Handler{
		verify: fakeVerifier{err: errors.New("spam check failed")},
		send:   delivery,
		limit:  limiter,
	}

	body := `{"email":"player@example.com","message":"This message is long enough to pass.","turnstile":"bad"}`
	request := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	if len(limiter.entries) != 0 {
		t.Fatalf("rate limiter entries = %d, want 0 after failed Turnstile", len(limiter.entries))
	}
}
