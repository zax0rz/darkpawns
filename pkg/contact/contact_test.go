package contact

import (
	"context"
	"errors"
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
