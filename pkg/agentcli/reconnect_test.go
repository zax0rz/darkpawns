package agentcli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestRunConnectedReconnectsOnDisconnect(t *testing.T) {
	var connectCount int32
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer c.Close()

		atomic.AddInt32(&connectCount, 1)

		// Consume login message.
		var login map[string]any
		if err := c.ReadJSON(&login); err != nil {
			return
		}
		// Send a non-error response so Connect() succeeds.
		if err := c.WriteJSON(map[string]any{
			"type": "state",
			"data": map[string]any{},
		}); err != nil {
			return
		}
		// Consume subscribe message.
		var sub map[string]any
		_ = c.ReadJSON(&sub)

		// Abruptly drop the connection so the client's read returns an error.
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	host := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("parse server port: %v", err)
	}

	cfg := &AgentConfig{
		Key:        "test-key",
		PlayerName: "TestBot",
		GameHost:   host,
		GamePort:   port,
	}

	client := NewAgentClient(cfg)
	rcfg := ReconnectConfig{
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     200 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         0.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = client.RunConnected(ctx, rcfg)

	if atomic.LoadInt32(&connectCount) < 2 {
		t.Fatalf("expected RunConnected to reconnect at least once, got %d connection(s)", connectCount)
	}
}
