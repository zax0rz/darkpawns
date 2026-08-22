package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/auth"
)

func setTestJWTSecret(t *testing.T) {
	t.Helper()
	os.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-characters-long")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })
}

func TestAuthMiddleware_AcceptsCaseInsensitiveBearer(t *testing.T) {
	setTestJWTSecret(t)

	token, err := auth.GenerateJWT("Hero", false, 0, "player")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []string{
		"Bearer " + token,
		"bearer " + token,
		"BEARER " + token,
		"BeArEr " + token,
	}

	for _, authHeader := range cases {
		t.Run(authHeader[:6], func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", authHeader)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}
		})
	}
}

func TestAuthMiddleware_RejectsMalformedBearer(t *testing.T) {
	setTestJWTSecret(t)

	token, err := auth.GenerateJWT("Hero", false, 0, "player")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for unauthorized request")
	}))

	cases := []string{
		"",
		"BearerX " + token,
		"Basic " + token,
		"Bearer" + token,
		"Token " + token,
	}

	for _, authHeader := range cases {
		t.Run(authHeader, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d for %q, got %d", http.StatusUnauthorized, authHeader, rec.Code)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q, want application/json; charset=utf-8", got)
			}
			var response ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}
			if response.Error.Code != "AUTH_REQUIRED" || response.Error.Resolution == "" {
				t.Errorf("unexpected error response: %+v", response)
			}
		})
	}
}

func TestAuthMiddleware_RejectsInvalidTokenAsJSON(t *testing.T) {
	setTestJWTSecret(t)
	handler := AuthMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be called for an invalid token")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/example", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	var response ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if response.Error.Code != "INVALID_TOKEN" {
		t.Errorf("error code = %q, want INVALID_TOKEN", response.Error.Code)
	}
}
