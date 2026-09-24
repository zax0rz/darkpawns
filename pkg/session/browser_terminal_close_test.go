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

// An empty name ends the connection with C's goodbye. Over the browser
// terminal the goodbye must arrive before the close, and the close must be an
// orderly one: the reader used to close the socket while the writer was still
// flushing, dropping the message and ending with an abnormal close (1006).
func TestBrowserTerminalGoodbyeArrivesBeforeClose(t *testing.T) {
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
				t.Fatalf("connection ended with %v, want an orderly close after the goodbye (got %q)", err, got.String())
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
	if !strings.Contains(got.String(), "Goodbye.") {
		t.Fatalf("goodbye never arrived; got %q", got.String())
	}
}
