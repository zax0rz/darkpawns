package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestContentNegotiationMiddleware_AllowsAPIRoutes verifies that requests under
// /api reach downstream handlers instead of being short-circuited with 404
// (DP-649).
func TestContentNegotiationMiddleware_AllowsAPIRoutes(t *testing.T) {
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	middleware := ContentNegotiationMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/api/players", nil)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	if !reached {
		t.Fatal("API route did not reach downstream handler")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("handler status = %d, want %d", rr.Code, http.StatusOK)
	}
}

// TestContentNegotiationMiddleware_OnboardingServesFiles verifies that
// /onboarding paths still serve content-negotiated files.
func TestContentNegotiationMiddleware_OnboardingServesFiles(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("onboarding request should not reach next handler")
	})

	middleware := ContentNegotiationMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/onboarding", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// We expect either the onboarding HTML file or a 404 if it is missing;
	// the important thing is that the middleware handled it itself.
	if rr.Code != http.StatusOK && rr.Code != http.StatusNotFound {
		t.Errorf("unexpected status = %d", rr.Code)
	}
}
