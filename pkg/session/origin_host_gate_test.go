package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
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

func TestCheckOriginLoopbackChecksOrigin(t *testing.T) {
	m := makeTestManager(t)

	cases := []struct {
		name       string
		remoteAddr string
		host       string
		origin     string
		want       bool
	}{
		{"loopback IPv4, foreign origin", "127.0.0.1:40000", "127.0.0.1:4350", "https://elsewhere.example", false},
		{"loopback IPv6, foreign origin", "[::1]:40000", "[::1]:4350", "https://elsewhere.example", false},
		{"loopback IPv4, no origin", "127.0.0.1:40000", "127.0.0.1:4350", "", true},
		{"loopback IPv6, no origin", "[::1]:40000", "[::1]:4350", "", true},
		{"loopback, localhost origin and host", "127.0.0.1:40000", "localhost:4350", "http://localhost:4350", true},
		{"loopback, 127.0.0.1 origin and host", "127.0.0.1:40000", "127.0.0.1:4350", "http://127.0.0.1:4350", true},
		{"loopback, IPv6 origin and host with port", "[::1]:40000", "[::1]:4350", "http://[::1]:4350", true},
		{"loopback, IPv6 origin and bare IPv6 host", "[::1]:40000", "[::1]", "http://[::1]:5173", true},
		{"loopback, Vite dev origin and local host", "127.0.0.1:40000", "localhost:4350", "http://localhost:5173", true},
		{"loopback, allowlisted origin", "127.0.0.1:40000", "127.0.0.1:4350", "https://darkpawns.org", true},
		{"loopback, local origin with non-local host", "127.0.0.1:40000", "darkpawns.org", "http://localhost:5173", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			req.RemoteAddr = tc.remoteAddr
			req.Host = tc.host
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if got := m.checkOrigin(req); got != tc.want {
				t.Errorf("checkOrigin(remote=%q, host=%q, origin=%q) = %v, want %v", tc.remoteAddr, tc.host, tc.origin, got, tc.want)
			}
		})
	}
}

func TestHandleWebSocketLoopbackForeignOriginNeverReachesCreation(t *testing.T) {
	database := entryDatabase(t)
	m := entryWorldManager(t, database)

	srv := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	headers := http.Header{"Origin": []string{"https://elsewhere.example"}}
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err == nil {
		_ = conn.Close()
		t.Fatal("loopback WebSocket upgrade with foreign Origin succeeded, want rejection")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("upgrade status = %d (err=%v), want %d", status, err, http.StatusForbidden)
	}
	count, err := database.CountPlayers()
	if err != nil {
		t.Fatalf("CountPlayers: %v", err)
	}
	if count != 0 {
		t.Fatalf("player count = %d, want 0", count)
	}
}
