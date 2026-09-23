package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/zax0rz/darkpawns/pkg/auth"
)

// TestWebSocketPerIPLimitCountsRealClientsBehindProxy reproduces the
// production topology: every connection arrives from the reverse proxy on
// loopback, naming the real client in X-Forwarded-For. The per-address cap
// must count clients, not the proxy: one household stops at the cap, while a
// different client behind the same proxy still gets in. Before the proxy was
// trusted, the whole site shared one 127.0.0.1 bucket of five.
func TestWebSocketPerIPLimitCountsRealClientsBehindProxy(t *testing.T) {
	if err := auth.SetTrustedProxies(auth.DefaultTrustedProxies); err != nil {
		t.Fatal(err)
	}
	m := makeTestManager(t)
	server := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	t.Cleanup(server.Close)
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	dial := func(client string) (*websocket.Conn, bool) {
		t.Helper()
		headers := http.Header{"X-Forwarded-For": {client}, "Origin": {"https://darkpawns.org"}}
		conn, _, err := websocket.DefaultDialer.Dial(url, headers)
		if err != nil {
			t.Fatalf("dial as %s: %v", client, err)
		}
		// A refused connection is closed with a policy-violation frame.
		_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		_, _, readErr := conn.ReadMessage()
		var closeErr *websocket.CloseError
		if ok := asCloseError(readErr, &closeErr); ok && closeErr.Code == websocket.ClosePolicyViolation {
			return conn, false
		}
		return conn, true
	}

	const household = "203.0.113.20"
	for i := 0; i < webSocketMaxConnsPerIP; i++ {
		conn, admitted := dial(household)
		t.Cleanup(func() { _ = conn.Close() })
		if !admitted {
			t.Fatalf("connection %d from one household refused below the cap of %d", i+1, webSocketMaxConnsPerIP)
		}
	}
	conn, admitted := dial(household)
	t.Cleanup(func() { _ = conn.Close() })
	if admitted {
		t.Fatalf("connection %d from one household admitted past the cap", webSocketMaxConnsPerIP+1)
	}
	other, admitted := dial("198.51.100.44")
	t.Cleanup(func() { _ = other.Close() })
	if !admitted {
		t.Fatal("a different client behind the same proxy was refused: the cap is counting the proxy")
	}
}

func asCloseError(err error, target **websocket.CloseError) bool {
	ce, ok := err.(*websocket.CloseError)
	if ok {
		*target = ce
	}
	return ok
}

// TestMultiplayLimitsFitTwoPlayers pins the sizing rationale: two players on
// one connection, each at the three-character allowance.
func TestMultiplayLimitsFitTwoPlayers(t *testing.T) {
	if webSocketMaxConnsPerIP < 2*3 {
		t.Fatalf("webSocketMaxConnsPerIP = %d, below two players' multiplay allowance", webSocketMaxConnsPerIP)
	}
}
