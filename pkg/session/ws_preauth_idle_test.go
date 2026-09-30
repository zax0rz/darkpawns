package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func setPreAuthIdleTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	old := preAuthIdleTimeout
	preAuthIdleTimeout = d
	t.Cleanup(func() { preAuthIdleTimeout = old })
}

func dialTestWebSocket(t *testing.T, m *Manager) *websocket.Conn {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	t.Cleanup(srv.Close)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	headers := http.Header{}
	headers.Set("Origin", "https://darkpawns.org")
	c, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("WebSocket dial: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// A parked pre-auth socket must be dropped on the DP-912 login idle clock
// even while it answers the server's pings: pongs prove a live socket, not a
// live login attempt, and an unanswered-clock socket otherwise holds one of
// the per-IP connection slots forever. Without the pre-auth deadline the
// pong handler refreshes the 60 s read deadline indefinitely.
func TestWebSocket_PreAuthIdleTimeout_DropsParkedSocketDespitePongs(t *testing.T) {
	setPreAuthIdleTimeout(t, 250*time.Millisecond)
	m := makeManagerWithStartRoom(t)
	c := dialTestWebSocket(t, m)

	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if err := c.WriteControl(websocket.PongMessage, nil, time.Now().Add(time.Second)); err != nil {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	start := time.Now()
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			elapsed := time.Since(start)
			if elapsed >= 1500*time.Millisecond {
				t.Fatalf("connection closed only after %v; want the pre-auth idle deadline (~250ms), not the pong-refreshed 60s clock", elapsed)
			}
			return
		}
	}
}

// Interactive pre-auth traffic (login attempts, char-creation input) keeps
// the session alive: every data frame refreshes the idle deadline.
func TestWebSocket_PreAuthIdleTimeout_DataFramesRefreshDeadline(t *testing.T) {
	setPreAuthIdleTimeout(t, 250*time.Millisecond)
	m := makeManagerWithStartRoom(t)
	c := dialTestWebSocket(t, m)

	// 2.4x the timeout while a data frame arrives every 80 ms. Pre-auth
	// commands answer ErrNotAuthenticated, so each frame also provokes a
	// reply the final read below can observe.
	for i := 0; i < 8; i++ {
		if err := c.WriteJSON(map[string]interface{}{"type": "command", "data": map[string]interface{}{"command": "look"}}); err != nil {
			t.Fatalf("WriteJSON: %v", err)
		}
		time.Sleep(80 * time.Millisecond)
	}

	// Still connected: a read may find silence (timeout) but must not report
	// a close — closing at 250 ms idle while data flowed would mean data
	// frames do not refresh the deadline.
	_ = c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, _, err := c.ReadMessage(); err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.ClosePolicyViolation) {
			t.Fatalf("connection closed while data frames were arriving: %v", err)
		}
	}
}
