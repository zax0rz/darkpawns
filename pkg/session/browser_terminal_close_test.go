package session

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// C src/interpreter.c:1721,1751-1752 closes an empty name without goodbye.
// Browser terminal framing must preserve that and close in an orderly way.
func TestBrowserTerminalEmptyNameClosesWithoutText(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	srv := httptest.NewServer(http.HandlerFunc(m.HandleWebSocket))
	t.Cleanup(srv.Close)

	headers := http.Header{}
	headers.Set("Origin", "https://darkpawns.org")
	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), headers)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	wsWrite(t, client, MsgTerminal, map[string]interface{}{})
	wsReadUntilType(t, client, MsgOut) // greeting and name prompt
	wsWrite(t, client, MsgLine, map[string]interface{}{"line": ""})

	_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
	var got strings.Builder
	for {
		_, data, err := client.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
				t.Fatalf("connection ended with %v, want an orderly empty-name close (got %q)", err, got.String())
			}
			break
		}
		var frame struct {
			Type string `json:"type"`
			Data struct {
				Text string `json:"text"`
			} `json:"data"`
		}
		if json.Unmarshal(data, &frame) == nil && frame.Type == MsgOut {
			got.WriteString(frame.Data.Text)
		}
	}
	if got.Len() != 0 {
		t.Fatalf("empty-name close invented text: %q", got.String())
	}
}
