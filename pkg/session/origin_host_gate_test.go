package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// VULN-007: a localhost Origin must only pass when the request's Host is
// also local (same-host development). A localhost Origin against the
// production Host is a page on the victim's own machine — a malicious local
// dev server or extension page — driving their logged-in session.
func TestCheckOriginLocalhostOriginRequiresLocalhostHost(t *testing.T) {
	m := makeTestManager(t)

	// Rejected: localhost Origin against the production Host.
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Host = "darkpawns.org"
	req.Header.Set("Origin", "http://localhost:5173")
	if m.checkOrigin(req) {
		t.Error("localhost Origin against production Host accepted")
	}

	// Rejected: 127.0.0.1 Origin against the production Host, any port/scheme.
	req = httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Host = "darkpawns.org"
	req.Header.Set("Origin", "http://127.0.0.1:3000")
	if m.checkOrigin(req) {
		t.Error("127.0.0.1 Origin against production Host accepted")
	}

	// Accepted: localhost Origin with a localhost Host (same-host development).
	req = httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Host = "localhost:4350"
	req.Header.Set("Origin", "http://localhost:5173")
	if !m.checkOrigin(req) {
		t.Error("localhost Origin with localhost Host rejected — same-host development broken")
	}

	// Accepted: 127.0.0.1 on both sides, ports may differ.
	req = httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Host = "127.0.0.1:4350"
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	if !m.checkOrigin(req) {
		t.Error("127.0.0.1 Origin with 127.0.0.1 Host rejected")
	}

	// The missing-Origin path is unchanged (its own gate handles it).
	req = httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Host = "localhost:4350"
	req.RemoteAddr = "192.168.1.10:12345"
	if m.checkOrigin(req) {
		t.Error("missing Origin from a private IP slipped through")
	}
}
