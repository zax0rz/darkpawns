package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersConnectSrcSameOriginOnly(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header is missing")
	}

	var connectSrc string
	for directive := range strings.SplitSeq(csp, ";") {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(directive, "connect-src ") {
			connectSrc = directive
			break
		}
	}
	if connectSrc != "connect-src 'self'" {
		t.Fatalf("connect-src directive = %q, want %q (full CSP: %q)", connectSrc, "connect-src 'self'", csp)
	}
}
