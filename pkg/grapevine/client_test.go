package grapevine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestGrapevineClient(t *testing.T) {
	// Spin up a mock websocket server
	upgrader := websocket.Upgrader{}
	var receivedMessages []grapevineMessage
	var receivedMu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("failed to upgrade server connection: %v", err)
			return
		}

		// Read loop on server
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var gMsg grapevineMessage
			if err := json.Unmarshal(msg, &gMsg); err == nil {
				receivedMu.Lock()
				receivedMessages = append(receivedMessages, gMsg)
				receivedMu.Unlock()

				// Automatically reply success to authenticate
				if gMsg.Event == "authenticate" {
					reply := grapevineMessage{
						Event:   "authenticate",
						Payload: json.RawMessage(`{"status":"success"}`),
					}
					_ = conn.WriteJSON(reply)
				}
			}
		}
	}))
	defer srv.Close()

	// Set environment variables to point client to mock server
	t.Setenv("GRAPEVINE_CLIENT_ID", "test_client")
	t.Setenv("GRAPEVINE_CLIENT_SECRET", "test_secret")
	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1) + "/socket"
	t.Setenv("GRAPEVINE_URL", wsURL)

	// Create dummy game world
	pw := &parser.World{}
	world, err := game.NewWorld(pw)
	if err != nil {
		t.Fatalf("failed to create world: %v", err)
	}

	client := NewClient(world)
	client.Start()
	defer client.Stop()

	// Wait for client to connect and handshake
	time.Sleep(100 * time.Millisecond)

	receivedMu.Lock()
	defer receivedMu.Unlock()

	if len(receivedMessages) < 2 {
		t.Errorf("expected at least 2 messages (auth + subscribe), got %d", len(receivedMessages))
		return
	}

	if receivedMessages[0].Event != "authenticate" {
		t.Errorf("expected first message event to be 'authenticate', got %s", receivedMessages[0].Event)
	}
	if receivedMessages[1].Event != "channels/subscribe" {
		t.Errorf("expected second message event to be 'channels/subscribe', got %s", receivedMessages[1].Event)
	}
}
