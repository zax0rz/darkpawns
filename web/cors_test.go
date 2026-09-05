package web

import (
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetAllowedOrigins_TrimsWhitespaceAndSkipsEmpty(t *testing.T) {
	orig := os.Getenv("CORS_ALLOWED_ORIGINS")
	defer os.Setenv("CORS_ALLOWED_ORIGINS", orig)

	os.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example, https://b.example,, https://c.example ")
	got := getAllowedOrigins()
	want := []string{"https://a.example", "https://b.example", "https://c.example"}
	if len(got) != len(want) {
		t.Fatalf("getAllowedOrigins() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("origin[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsOriginAllowed_WithTrimmedOrigins(t *testing.T) {
	origins := []string{"https://a.example", "https://b.example"}
	if !isOriginAllowed("https://b.example", origins, nil) {
		t.Error("isOriginAllowed should accept origin after trimming")
	}
}

func TestCORS_DevModeNonLocalRejected(t *testing.T) {
	origEnv := os.Getenv("ENVIRONMENT")
	defer os.Setenv("ENVIRONMENT", origEnv)
	os.Setenv("ENVIRONMENT", "development")

	req := httptest.NewRequest("GET", "http://1.2.3.4:3000/something", nil)
	req.RemoteAddr = "5.6.7.8:53000"
	req.Header.Set("Origin", "http://evil.example")

	if isOriginAllowed("http://evil.example", nil, req) {
		t.Error("expected non-local dev mode request to be rejected")
	}
}

func TestCORS_DevModeLocalAllowed(t *testing.T) {
	origEnv := os.Getenv("ENVIRONMENT")
	defer os.Setenv("ENVIRONMENT", origEnv)
	os.Setenv("ENVIRONMENT", "development")

	for _, host := range []string{"localhost:3000", "127.0.0.1:3000", "[::1]:3000"} {
		req := httptest.NewRequest("GET", "http://"+host+"/something", nil)
		req.RemoteAddr = "127.0.0.1:54321"
		req.Header.Set("Origin", "http://example.com")
		if !isOriginAllowed("http://example.com", nil, req) {
			t.Errorf("expected local host %q to be allowed in dev mode", host)
		}
	}
}

func TestCORS_DevModeSpoofedLocalHostFromNonLocalPeerRejected(t *testing.T) {
	origEnv := os.Getenv("ENVIRONMENT")
	defer os.Setenv("ENVIRONMENT", origEnv)
	os.Setenv("ENVIRONMENT", "development")

	req := httptest.NewRequest("GET", "http://localhost:3000/something", nil)
	req.RemoteAddr = "1.2.3.4:53000"
	req.Header.Set("Origin", "http://evil.example")

	if isOriginAllowed("http://evil.example", nil, req) {
		t.Error("expected spoofed local Host from a non-local peer to be rejected")
	}
}
