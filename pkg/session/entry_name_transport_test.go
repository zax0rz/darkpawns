package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

// Real JSON frames and SQLite: raw bytes arrive at the same C name gate.
func TestEntryNameWebSocketBoundary(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Aiko")
	m := entryTransportManager(t, database)
	m.loginLimiter.GetLimiter("127.0.0.1").SetLimit(rate.Inf)
	server := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	defer server.Close()
	conn := entryWebSocketDial(t, server.URL)
	defer func() { _ = conn.Close() }()
	wsWrite(t, conn, MsgLogin, map[string]interface{}{"player_name": "Aiko "})
	var prompt CharCreateData
	entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
	if prompt.Stage != "get_name" || prompt.Prompt != "Invalid name, please try another.\r\nName: " {
		t.Fatalf("initial raw rejection: %+v", prompt)
	}
	for _, name := range []string{"the", "a_b", "Aiko "} {
		wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": name})
		entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
		if prompt.Stage != "get_name" || prompt.Prompt != "Invalid name, please try another.\r\nName: " {
			t.Fatalf("retry %q: %+v", name, prompt)
		}
	}
	wsWrite(t, conn, MsgCharInput, map[string]interface{}{"choice": "  AIKO"})
	entryUnmarshalPrompt(t, wsReadUntilType(t, conn, MsgCharCreate), &prompt)
	if prompt.Stage != "login_password" || prompt.Prompt != "Password: " || !prompt.Secret {
		t.Fatalf("saved folded name: %+v", prompt)
	}
	if n, err := database.CountPlayers(); err != nil || n != 1 {
		t.Fatalf("name rejection changed rows: n=%d err=%v", n, err)
	}
	blank := entryWebSocketDial(t, server.URL)
	defer func() { _ = blank.Close() }()
	wsWrite(t, blank, MsgLogin, map[string]interface{}{"player_name": "   "})
	if err := blank.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := blank.ReadMessage(); !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
		t.Fatalf("all-space initial name did not close normally: %v", err)
	}
}
