package session

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func setAuthReadDeadline(t *testing.T, d time.Duration) {
	t.Helper()
	old := authReadDeadline
	authReadDeadline = d
	t.Cleanup(func() { authReadDeadline = old })
}

// waitForPlayingSession waits until the manager reports one playing session.
func waitForPlayingSession(t *testing.T, m *Manager) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, playing := m.CountSessions(); playing > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no authenticated playing session was established")
}

// TestWebSocket_AuthenticatedIdlePlayerSurvivesLivenessClock is the DP-1385
// proof on the browser transport. An authenticated, in-world session that sends
// no data frames is not dropped while it keeps answering the server's pings:
// its pongs refresh the liveness deadline, exactly as every browser does
// automatically. Remove that pong refresh and the session dies at
// authReadDeadline, so the test fails against the old behaviour (R5h). The
// clock is the injectable authReadDeadline, not 60 s, so it runs in well under
// a second.
func TestWebSocket_AuthenticatedIdlePlayerSurvivesLivenessClock(t *testing.T) {
	t.Setenv("JWT_SECRET", "ws-auth-idle-test-secret-at-least-32-bytes")
	setAuthReadDeadline(t, 250*time.Millisecond)
	// The guest path enters the world directly and needs MortalStartRoom.
	m := entryTransportManager(t, nil)
	c := dialTestWebSocket(t, m)

	wsWrite(t, c, MsgLogin, map[string]interface{}{"player_name": "guest"})
	waitForPlayingSession(t, m)

	// One blocking reader: any error on it is the server closing the socket,
	// which is the drop DP-1385 is about. (A gorilla read deadline is fatal to
	// the connection, so the reader must not poll with short deadlines.)
	readErr := make(chan error, 1)
	go func() {
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				readErr <- err
				return
			}
		}
	}()

	// Answer pings only -- no data frames -- for several deadline periods.
	watchUntil := time.Now().Add(3 * authReadDeadline)
	for time.Now().Before(watchUntil) {
		if err := c.WriteControl(websocket.PongMessage, nil, time.Now().Add(time.Second)); err != nil {
			t.Fatalf("pong write failed: %v", err)
		}
		select {
		case err := <-readErr:
			t.Fatalf("authenticated idle session was dropped while answering pings: %v", err)
		case <-time.After(40 * time.Millisecond):
		}
	}

	select {
	case err := <-readErr:
		t.Fatalf("authenticated idle session was dropped while answering pings: %v", err)
	default:
	}
	if _, playing := m.CountSessions(); playing == 0 {
		t.Fatal("authenticated idle session was dropped")
	}
}
