package session

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// BUG-R2-C1-A3-H2: finishWebSocketTransport left a non-retained session's send
// channel open, so writePump (whose only exits are <-s.send, <-TransportDone()
// and the 54 s ping ticker) parked for up to 54 s per abruptly-closed pre-auth
// connection, pinning a goroutine, the 256-slot send buffer and the retained
// *http.Request. CloseSend is the waker; the retained linkdead arm must keep
// the channel for reattach.
func TestFinishWebSocketTransport_NonRetainedSessionClosesSend(t *testing.T) {
	m := makeTestManager(t)
	// Unauthenticated: HandleTransportDisconnect refuses to retain it, so
	// teardown takes the UnregisterSession arm.
	s := makeTestSession(t, m, "PreAuth", 1001, false)

	// Stand in for writePump parked on the send channel: the loop ends only
	// when the channel closes. Pre-fix this goroutine never wakes.
	woken := make(chan struct{})
	go func() {
		defer close(woken)
		for range s.send {
		}
	}()

	s.finishWebSocketTransport()

	if !s.SendClosed() {
		t.Fatal("finishWebSocketTransport left the send channel open for a non-retained session")
	}
	select {
	case <-woken:
	case <-time.After(2 * time.Second):
		t.Fatal("a writer parked on the send channel was not woken by teardown")
	}
}

// The other half of BUG-R2-C1-A3-H2: a playing character retained as linkdead
// (DP-1323) must NOT have its send channel closed by transport teardown — the
// character is reattached to it, and CloseSend would end the session instead.
func TestFinishWebSocketTransport_RetainedLinkdeadSessionKeepsSend(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Linkdead", 1001, true)
	s.transportDone = make(chan struct{}) // DetachTransport closes this
	registerTestSession(t, m, s, "Linkdead")

	s.finishWebSocketTransport()

	if s.SendClosed() {
		t.Fatal("finishWebSocketTransport closed the send channel of a retained linkdead session")
	}
	if got, ok := m.GetSession("Linkdead"); !ok || got != s {
		t.Fatal("retained linkdead session was unregistered by transport teardown")
	}
	// Reattach needs the channel; it must still accept a message.
	select {
	case s.send <- []byte("{}"):
	default:
		t.Fatal("retained linkdead session's send channel is unusable")
	}
}

// BUG-R2-C1-A3-H1: HandleWebSocket gated only on webSocketMaxConnsPerIP, which
// rotating source addresses defeat (no IPv6 /64 aggregation). The process-wide
// cap must reject an over-cap connection with a policy-violation close and
// admit one again once the accepted connections release their slots.
func TestHandleWebSocket_TotalConnectionCap(t *testing.T) {
	origMax := webSocketMaxTotalConns
	webSocketMaxTotalConns = 2
	t.Cleanup(func() { webSocketMaxTotalConns = origMax })

	m := makeTestManager(t)
	srv := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	headers := http.Header{}
	headers.Set("Origin", "https://darkpawns.org")
	dial := func() *websocket.Conn {
		t.Helper()
		c, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
		if err != nil {
			t.Fatalf("WebSocket dial: %v", err)
		}
		return c
	}

	// Under the cap: both connections are accepted. Wait for each admission so
	// the counter is settled before the next dial (the handler increments after
	// the handshake reply the dialer has already received).
	first := dial()
	waitForTotalConns(t, m, 1)
	second := dial()
	waitForTotalConns(t, m, 2)

	// Over the cap: a third connection from the same address is refused. The
	// per-IP bucket holds 2 of webSocketMaxConnsPerIP, so only the total cap
	// can be the reason.
	third := dial()
	t.Cleanup(func() { _ = third.Close() })
	_ = third.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := third.ReadMessage()
	var closeErr *websocket.CloseError
	if !errors.As(err, &closeErr) || closeErr.Code != websocket.ClosePolicyViolation {
		t.Fatalf("over-cap connection: err = %v, want close code %d", err, websocket.ClosePolicyViolation)
	}
	if closeErr.Text != "server is at capacity" {
		t.Fatalf("over-cap close reason = %q, want %q", closeErr.Text, "server is at capacity")
	}

	m.ipConnMu.Lock()
	perIP := make(map[string]int, len(m.ipConnCount))
	for ip, n := range m.ipConnCount {
		perIP[ip] = n
	}
	total := m.totalConnCount
	m.ipConnMu.Unlock()
	for ip, n := range perIP {
		if n >= webSocketMaxConnsPerIP {
			t.Fatalf("test invalid: per-IP bucket %s = %d is already at its own cap", ip, n)
		}
	}
	if total != 2 {
		t.Fatalf("rejected connection changed totalConnCount: got %d, want 2", total)
	}

	// Releasing the accepted connections frees their slots: the cap bounds
	// simultaneous connections, it does not latch shut.
	_ = first.Close()
	_ = second.Close()
	waitForTotalConns(t, m, 0)

	fourth := dial()
	t.Cleanup(func() { _ = fourth.Close() })
	waitForTotalConns(t, m, 1)
}

// waitForTotalConns blocks until the manager's process-wide connection count
// reaches want, or fails the test after 2 s.
func waitForTotalConns(t *testing.T, m *Manager, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		m.ipConnMu.Lock()
		got := m.totalConnCount
		m.ipConnMu.Unlock()
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("totalConnCount = %d, want %d", got, want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
