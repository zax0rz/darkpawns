package session

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeManagerWithStartRoom returns a Manager whose world contains the mortal
// start room (8004) so sendWelcome can build a valid StateData message.
func makeManagerWithStartRoom(t *testing.T) *Manager {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum:        game.MortalStartRoom,
				Name:        "The Adventurers Guild",
				Description: "A grand hall where adventurers gather to seek glory.",
				Zone:        80,
			},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return NewManager(w, nil)
}

// wsReadUntilType drains WebSocket messages and returns the first one whose
// "type" field matches wantType.  All prior messages are silently discarded.
// Fails the test after 5 s with no match.
func wsReadUntilType(t *testing.T, c *websocket.Conn, wantType string) map[string]interface{} {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage (waiting for %q): %v", wantType, err)
		}
		var msg map[string]interface{}
		if jsonErr := json.Unmarshal(raw, &msg); jsonErr != nil {
			t.Fatalf("unmarshal: %v", jsonErr)
		}
		if msg["type"] == wantType {
			_ = c.SetReadDeadline(time.Time{}) // clear after match
			return msg
		}
	}
}

// wsWrite sends a JSON message over the WebSocket connection.
func wsWrite(t *testing.T, c *websocket.Conn, msgType string, data interface{}) {
	t.Helper()
	if err := c.WriteJSON(map[string]interface{}{"type": msgType, "data": data}); err != nil {
		t.Fatalf("WriteJSON(%q): %v", msgType, err)
	}
}

// TestWebSocket_NewCharThenLook is the end-to-end proof that command responses
// reach the client after character creation over a real WebSocket connection.
//
// Flow: dial → login (new_char) → walk all 6 char-create stages → receive
// welcome state → send "look" → receive room state with the right room name.
//
// Why the na ve test client receives nothing (root-cause explanation):
//
//  1. The test sends login with new_char:true but never sends char_input
//     responses to the char_create prompts the server emits.
//
//  2. The server's readPump (session_pump.go:38) holds a 60-second read
//     deadline that is only renewed on pong (line 39-41).  While the client
//     is reading in a loop waiting for "state", the server is blocked in
//     conn.ReadMessage() waiting for the next char_input that never comes.
//
//  3. After 60 seconds the deadline fires, readPump breaks out of its loop,
//     Unregister() closes s.send, writePump receives !ok and calls conn.Close().
//
//  4. Because char creation never completed, sendWelcome was never called and
//     no "state" message was ever sent — the client receives only char_create
//     prompts before the connection closes under it.
//
// Secondary issue for in-process tests:
//
//	net.IP.IsPrivate() returns false for 127.0.0.1 (loopback is not RFC-1918).
//	The upgrader.CheckOrigin rejects loopback connections that carry no Origin
//	header, so the dialer must supply an allowed origin.
func TestWebSocket_NewCharThenLook(t *testing.T) {
	m := makeManagerWithStartRoom(t)
	srv := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	headers := http.Header{}
	headers.Set("Origin", "https://darkpawns.labz0rz.com")
	c, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("WebSocket dial: %v", err)
	}
	defer c.Close()

	// ── 1. Login ─────────────────────────────────────────────────────────────
	wsWrite(t, c, MsgLogin, map[string]interface{}{
		"player_name": "Testchar",
		"password":    "hunter2",
		"new_char":    true,
	})

	// ── 2. Walk every char-creation stage ────────────────────────────────────
	// Each choice must be preceded by receiving the matching char_create prompt
	// so readPump is in conn.ReadMessage() and ready for the char_input.
	charStages := []string{
		"Y",       // confirm_name   → Yes
		"hunter2", // create_password → hunter2
		"hunter2", // confirm_password → hunter2
		"N",       // color          → no ANSI
		"M",       // sex            → male
		"H",       // race           → human
		"W",       // class          → warrior
		"K",       // hometown       → Kir Drax'in (room 8004 = game.MortalStartRoom)
	}
	for _, choice := range charStages {
		wsReadUntilType(t, c, MsgCharCreate)
		wsWrite(t, c, MsgCharInput, map[string]interface{}{"choice": choice})
	}
	// stats_roll prompt → accept
	wsReadUntilType(t, c, MsgCharCreate)
	wsWrite(t, c, MsgCharInput, map[string]interface{}{"choice": "Y"})

	// motd prompt → press return
	wsReadUntilType(t, c, MsgCharCreate)
	wsWrite(t, c, MsgCharInput, map[string]interface{}{"choice": ""})

	// ── 3. Receive welcome state ──────────────────────────────────────────────
	// completeCharCreation calls sendWelcome which does s.send <- state_msg
	// (blocking, char_creation.go:276 / session_send.go:58).  writePump drains
	// s.send and calls conn.WriteMessage — the message travels over the real
	// TCP socket to our test client.
	welcomeRaw := wsReadUntilType(t, c, MsgState)

	stateBytes, _ := json.Marshal(welcomeRaw["data"])
	var welcome StateData
	if err := json.Unmarshal(stateBytes, &welcome); err != nil {
		t.Fatalf("unmarshal welcome StateData: %v", err)
	}
	if welcome.Player.Name != "Testchar" {
		t.Errorf("welcome: player.name = %q, want Testchar", welcome.Player.Name)
	}
	if welcome.Room.VNum != game.MortalStartRoom {
		t.Errorf("welcome: room.vnum = %d, want %d", welcome.Room.VNum, game.MortalStartRoom)
	}

	// ── 4. Send "look", verify room description arrives ───────────────────────
	// handleCommand (session_login.go:249) → ExecuteCommand → cmdLook
	// (cmd_look.go:101) → s.send <- state_msg (BLOCKING direct send).
	// writePump writes it; ReadMessage must return it.
	wsWrite(t, c, MsgCommand, map[string]interface{}{
		"command": "look",
		"args":    []string{},
	})

	lookRaw := wsReadUntilType(t, c, MsgState)

	lookBytes, _ := json.Marshal(lookRaw["data"])
	var look StateData
	if err := json.Unmarshal(lookBytes, &look); err != nil {
		t.Fatalf("unmarshal look StateData: %v", err)
	}
	if look.Room.VNum != game.MortalStartRoom {
		t.Errorf("look: room.vnum = %d, want %d", look.Room.VNum, game.MortalStartRoom)
	}
	if look.Room.Name != "The Adventurers Guild" {
		t.Errorf("look: room.name = %q, want The Adventurers Guild", look.Room.Name)
	}
	if look.Player.Name != "Testchar" {
		t.Errorf("look: player.name = %q, want Testchar", look.Player.Name)
	}
	if look.Room.Description == "" {
		t.Error("look: room.description must not be empty")
	}
}
