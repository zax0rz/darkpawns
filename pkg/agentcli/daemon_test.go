package agentcli

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func shortHome(t *testing.T) string {
	t.Helper()
	home, err := os.MkdirTemp("/tmp", "dp")
	if err != nil {
		t.Fatalf("mkdir temp home: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	t.Setenv("HOME", home)
	return home
}

func fakeMUDServer(t *testing.T) (*httptest.Server, chan struct{}) {
	t.Helper()
	closed := make(chan struct{})
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		// Consume login.
		var login map[string]any
		_ = c.ReadJSON(&login)
		// Respond so Connect() succeeds.
		_ = c.WriteJSON(map[string]any{
			"type": "state",
			"data": map[string]any{},
		})
		// Consume subscribe.
		var sub map[string]any
		_ = c.ReadJSON(&sub)

		// Block until the client closes the connection.
		_, _, _ = c.ReadMessage()
		close(closed)
	}))
	t.Cleanup(srv.Close)
	return srv, closed
}

func closedTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

func TestStartConnectFailureLeavesDaemonStopped(t *testing.T) {
	shortHome(t)

	cfg := &AgentConfig{
		Key:        "test-key",
		PlayerName: "FailBot",
		GameHost:   "127.0.0.1",
		GamePort:   closedTCPPort(t),
	}

	d, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("new daemon: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = d.Start(ctx)
	if err == nil {
		t.Fatal("expected Start to fail on connect error")
	}

	d.mu.Lock()
	running := d.running
	d.mu.Unlock()
	if running {
		t.Fatal("daemon left running after failed Start")
	}

	// A subsequent Start should not be rejected as already running.
	err2 := d.Start(ctx)
	if err2 == nil {
		t.Fatal("expected second Start to fail because MUD still unreachable")
	}
	if strings.Contains(err2.Error(), "already running") {
		t.Fatalf("expected fresh start attempt, got: %v", err2)
	}
}

func TestStartListenerFailureClosesMUDConnection(t *testing.T) {
	shortHome(t)

	srv, closed := fakeMUDServer(t)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	host := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("parse server port: %v", err)
	}

	// Use a very long player name so the Unix socket path exceeds the
	// maximum address length and net.Listen fails after Connect succeeds.
	cfg := &AgentConfig{
		Key:        "test-key",
		PlayerName: strings.Repeat("a", 200),
		GameHost:   host,
		GamePort:   port,
	}

	d, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("new daemon: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = d.Start(ctx)
	if err == nil {
		t.Fatal("expected Start to fail on listener error")
	}
	if !strings.Contains(err.Error(), "listen socket") {
		t.Fatalf("expected listener error, got: %v", err)
	}

	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("MUD connection was not closed after listener failure")
	}

	d.mu.Lock()
	running := d.running
	d.mu.Unlock()
	if running {
		t.Fatal("daemon left running after failed Start")
	}
}

func TestStartRecoversAfterFailure(t *testing.T) {
	shortHome(t)

	cfg := &AgentConfig{
		Key:        "test-key",
		PlayerName: "RecoverBot",
		GameHost:   "127.0.0.1",
		GamePort:   closedTCPPort(t),
	}

	d, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("new daemon: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := d.Start(ctx); err == nil {
		t.Fatal("expected first Start to fail")
	}

	srv, _ := fakeMUDServer(t)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	cfg.GameHost = u.Hostname()
	cfg.GamePort, _ = strconv.Atoi(u.Port())

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	done := make(chan error, 1)
	go func() {
		done <- d.Start(ctx2)
	}()

	// Let the daemon start successfully, then shut it down.
	time.Sleep(200 * time.Millisecond)
	cancel2()

	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("expected clean shutdown, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

// TestSendCommandConcurrentReconnect verifies that sendCommand does not race
// with reconnect: the nil-check and WriteJSON must be atomic under d.mu
// (DP-668).
func TestSendCommandConcurrentReconnect(t *testing.T) {
	shortHome(t)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// Server that accepts repeated reconnects and consumes agent messages.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		var login map[string]any
		_ = c.ReadJSON(&login)
		_ = c.WriteJSON(map[string]any{
			"type": "state",
			"data": map[string]any{},
		})
		var sub map[string]any
		_ = c.ReadJSON(&sub)

		// Read messages until the client disconnects.
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				return
			}
		}
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
		PlayerName: "RaceBot",
		GameHost:   host,
		GamePort:   port,
	}

	d, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("new daemon: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := NewAgentClient(cfg)
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	d.client = client

	var wg sync.WaitGroup

	// Writer goroutine: repeatedly send commands.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			resp := d.sendCommand("look", nil)
			if !resp.OK && resp.Error != "not connected" && !strings.Contains(resp.Error, "close sent") {
				t.Errorf("sendCommand error: %s", resp.Error)
			}
		}
	}()

	// Reconnector goroutine: close and reconnect the same AgentClient.
	// Hold d.mu around the close+reconnect so that sendCommand cannot read
	// d.client.conn while it is being replaced (DP-668).
	wg.Add(1)
	go func() {
		defer wg.Done()
		rcfg := ReconnectConfig{
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     50 * time.Millisecond,
			Multiplier:     2.0,
			Jitter:         0.0,
			MaxAttempts:    1,
		}
		for i := 0; i < 30; i++ {
			d.mu.Lock()
			_ = client.Close()
			_ = client.Reconnect(ctx, rcfg)
			d.mu.Unlock()
		}
	}()

	wg.Wait()
	cancel()
}
