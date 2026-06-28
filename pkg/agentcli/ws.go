package agentcli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSConn wraps gorilla/websocket.Conn for the agent CLI.
type WSConn struct {
	conn    *websocket.Conn
	seqMu   sync.Mutex
	lastSeq uint64
	writeMu sync.Mutex
}

// Dial connects to a WebSocket endpoint.
func Dial(ctx context.Context, addr string) (*WSConn, error) {
	return DialWithHeaders(ctx, addr, nil)
}

// DialWithHeaders connects to a WebSocket endpoint with custom headers.
func DialWithHeaders(ctx context.Context, addr string, headers http.Header) (*WSConn, error) {
	c, _, err := websocket.DefaultDialer.DialContext(ctx, addr, headers)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}
	return &WSConn{conn: c}, nil
}

// WriteJSON sends a JSON message.
func (w *WSConn) WriteJSON(v any) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	return w.conn.WriteJSON(v)
}

// ReadJSON reads a JSON message.
func (w *WSConn) ReadJSON(v any) error {
	_ = w.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	return w.conn.ReadJSON(v)
}

// ReadMessage reads a raw message.
func (w *WSConn) ReadMessage() (int, []byte, error) {
	_ = w.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	return w.conn.ReadMessage()
}

// Close closes the connection.
func (w *WSConn) Close() error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	return w.conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

// UnmarshalJSON is a helper to parse a raw JSON message into a typed struct.
func UnmarshalJSON(raw []byte, target any) error {
	return json.Unmarshal(raw, target)
}
