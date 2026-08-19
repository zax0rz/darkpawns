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

func TestGrapevineOutboundGossip(t *testing.T) {
	upgrader := websocket.Upgrader{}
	var receivedMessages []grapevineMessage
	var receivedMu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("failed to upgrade server connection: %v", err)
			return
		}

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

				if gMsg.Event == "authenticate" {
					_ = conn.WriteJSON(grapevineMessage{
						Event:   "authenticate",
						Payload: json.RawMessage(`{"status":"success"}`),
					})
				}
			}
		}
	}))
	defer srv.Close()

	t.Setenv("GRAPEVINE_CLIENT_ID", "test_client")
	t.Setenv("GRAPEVINE_CLIENT_SECRET", "test_secret")
	t.Setenv("GRAPEVINE_URL", strings.Replace(srv.URL, "http://", "ws://", 1)+"/socket")

	pw := &parser.World{}
	world, err := game.NewWorld(pw)
	if err != nil {
		t.Fatalf("failed to create world: %v", err)
	}

	client := NewClient(world)
	client.Start()
	defer client.Stop()

	// Wait for the handshake (auth + subscribe) to confirm the connection is up.
	deadline := time.Now().Add(2 * time.Second)
	for {
		receivedMu.Lock()
		count := len(receivedMessages)
		receivedMu.Unlock()
		if count >= 2 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	client.sendGossip("tester", "hello grapevine")

	deadline = time.Now().Add(2 * time.Second)
	got := false
	for !got && time.Now().Before(deadline) {
		receivedMu.Lock()
		for _, m := range receivedMessages {
			if m.Event == "channels/send" {
				var payload struct {
					Channel string `json:"channel"`
					Name    string `json:"name"`
					Message string `json:"message"`
				}
				if err := json.Unmarshal(m.Payload, &payload); err != nil {
					receivedMu.Unlock()
					t.Fatalf("failed to unmarshal gossip payload: %v", err)
				}
				if payload.Channel == "gossip" && payload.Name == "tester" && payload.Message == "hello grapevine" {
					got = true
				}
			}
		}
		receivedMu.Unlock()
		if !got {
			time.Sleep(10 * time.Millisecond)
		}
	}

	if !got {
		receivedMu.Lock()
		defer receivedMu.Unlock()
		t.Errorf("expected to receive gossip channels/send message, got %v", receivedMessages)
	}
}
