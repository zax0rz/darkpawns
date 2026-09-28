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

// makeManagerWithStartRoom returns a Manager whose world contains the newbie
// intro and Kir Drax'in hometown rooms used during first entry.
func makeManagerWithStartRoom(t *testing.T) *Manager {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum:        game.NewbieStartRoom,
				Name:        "A Burning Hut",
				Description: "Flames dance along the walls of the ruined hut.",
				Zone:        80,
			},
			{
				VNum:        game.NewbieHometownRoom(1),
				Name:        "Temple Infirmary",
				Description: "A quiet infirmary tended by the temple healers.",
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
	return newTestManager(t, w, nil)
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

// wsCollectText reads messages for d and returns the concatenated text of every
// MsgText seen, so a spec's multi-message output can be asserted in one place.
func wsCollectText(c *websocket.Conn, d time.Duration) string {
	var b strings.Builder
	deadline := time.Now().Add(d)
	for {
		if err := c.SetReadDeadline(deadline); err != nil {
			return b.String()
		}
		_, raw, err := c.ReadMessage()
		if err != nil {
			return b.String()
		}
		var msg map[string]interface{}
		if jsonErr := json.Unmarshal(raw, &msg); jsonErr != nil {
			continue
		}
		data, _ := json.Marshal(msg["data"])
		switch msg["type"] {
		case MsgText:
			var text TextData
			if jsonErr := json.Unmarshal(data, &text); jsonErr != nil {
				continue
			}
			b.WriteString(text.Text)
		case MsgEvent:
			// The world's MessageSink wraps player messages as text events.
			var event EventData
			if jsonErr := json.Unmarshal(data, &event); jsonErr != nil {
				continue
			}
			if event.Type == "text" {
				b.WriteString(event.Text)
			}
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
	headers.Set("Origin", "https://darkpawns.org")
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
		"K",       // hometown       → Kir Drax'in (room 8162)
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

	// main menu prompt → enter the game
	wsReadUntilType(t, c, MsgCharCreate)
	wsWrite(t, c, MsgCharInput, map[string]interface{}{"choice": "1"})

	// ── 3. Receive the one-time newbie room state ────────────────────────────────
	introRaw := wsReadUntilType(t, c, MsgState)
	introBytes, _ := json.Marshal(introRaw["data"])
	var intro StateData
	if err := json.Unmarshal(introBytes, &intro); err != nil {
		t.Fatalf("unmarshal intro StateData: %v", err)
	}
	if intro.Room.VNum != game.NewbieStartRoom {
		t.Errorf("intro: room.vnum = %d, want %d", intro.Room.VNum, game.NewbieStartRoom)
	}
	if intro.Room.Name != "A Burning Hut" {
		t.Errorf("intro: room.name = %q, want A Burning Hut", intro.Room.Name)
	}

	// ── 4. Send "look" and verify start_room's C bytes ────────────────────────
	wsWrite(t, c, MsgCommand, map[string]interface{}{
		"command": "look",
		"args":    []string{},
	})

	// C's start_room has no CMD_IS gate (src/spec_procs.c:2204-2263), so the
	// first command in the Burning Hut is what runs it: the birth speech, the
	// move to the hometown infirmary/altar, and do_look of that room. The spec
	// consumes the command, so the rendered room arrives as text (the same bytes
	// the pulse produces a few seconds later) and not as a look state.
	output := wsCollectText(c, 2*time.Second)
	if !strings.Contains(output, "now is not your time to die") {
		t.Errorf("look: start_room birth speech missing, got %q", output)
	}
	if !strings.Contains(output, "A quiet infirmary tended by the temple healers.") {
		t.Errorf("look: start_room did not render the hometown room, got %q", output)
	}
}
