package agentcli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// WSConn wraps gorilla/websocket.Conn for the agent CLI.
type WSConn struct {
	conn *websocket.Conn
}

// Dial connects to a WebSocket endpoint.
func Dial(ctx context.Context, addr string) (*WSConn, error) {
	c, _, err := websocket.DefaultDialer.DialContext(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}
	return &WSConn{conn: c}, nil
}

// WriteJSON sends a JSON message.
func (w *WSConn) WriteJSON(v any) error {
	return w.conn.WriteJSON(v)
}

// ReadJSON reads a JSON message.
func (w *WSConn) ReadJSON(v any) error {
	w.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	return w.conn.ReadJSON(v)
}

// ReadMessage reads a raw message.
func (w *WSConn) ReadMessage() (int, []byte, error) {
	w.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	return w.conn.ReadMessage()
}

// Close closes the connection.
func (w *WSConn) Close() error {
	return w.conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

// UnmarshalJSON is a helper to parse a raw JSON message into a typed struct.
func UnmarshalJSON(raw []byte, target any) error {
	return json.Unmarshal(raw, target)
}
