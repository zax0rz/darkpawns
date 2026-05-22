// Package session manages WebSocket connections and player sessions.
package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/auth"
)

func (s *Session) readPump() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("PANIC in readPump", "recover", r, "player", s.playerName, "stack", debug.Stack())
		}
		// Always decrement IP connection count (C5 leak fix)
		if !s.connCountDecremented && s.request != nil {
			s.connCountDecremented = true
			ip := auth.GetIPFromRequest(s.request)
			if ip != "" {
				s.manager.ipConnMu.Lock()
				s.manager.ipConnCount[ip]--
				if s.manager.ipConnCount[ip] <= 0 {
					delete(s.manager.ipConnCount, ip)
				}
				s.manager.ipConnMu.Unlock()
			}
		}
		s.manager.Unregister(s.playerName)
		_ = s.conn.Close()
	}()

	s.conn.SetReadLimit(16384) // 16KB max message size (C4)
	_ = s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.conn.SetPongHandler(func(string) error {
		_ = s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("WebSocket error", "error", err)
			}
			break
		}

		if err := s.handleMessage(message); err != nil {
			slog.Error("handle message error", "error", err)
			s.sendErrorWithState(err)
		}
	}
}

// writePump writes messages to the WebSocket.
func (s *Session) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		_ = s.conn.Close()
	}()

	for {
		select {
		case message, ok := <-s.send:
			_ = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// DP-GOAT P0-1: Stamp sequence number on every outbound message
			s.msgSeq++
			seqJSON := fmt.Sprintf(`,"seq":%d`, s.msgSeq)
			// Inject after the type value: find `"type":"`, skip to closing `"`, insert
			idx := bytes.Index(message, []byte(`"type":"`))
			if idx >= 0 {
				closeQuote := idx + 8
				end := bytes.IndexByte(message[closeQuote:], '"')
				if end >= 0 {
					insertAt := closeQuote + end + 1
					newMsg := make([]byte, 0, len(message)+len(seqJSON))
					newMsg = append(newMsg, message[:insertAt]...)
					newMsg = append(newMsg, seqJSON...)
					newMsg = append(newMsg, message[insertAt:]...)
					message = newMsg
				}
			}

			// DP-GOAT P0-2: Strip ANSI escape codes for agent sessions
			// Agents receive plain text; no need to parse escape sequences
			// They're applied in fmt strings across pkg/game and pkg/session —
			// this single strip catches everything.
			if s.isAgent {
				message = stripANSI(message)
			}

			_ = s.conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			_ = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages.
func (s *Session) handleMessage(data []byte) error {
	var msg ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	switch msg.Type {
	case MsgLogin:
		return s.handleLogin(msg.Data)
	case MsgCommand:
		if !s.authenticated {
			return ErrNotAuthenticated
		}
		return s.handleCommand(msg.Data)
	case MsgSubscribe:
		if !s.authenticated {
			return ErrNotAuthenticated
		}
		return s.handleSubscribe(msg.Data)
	case MsgCharInput:
		if s.charCreating {
			return s.handleCharInput(msg.Data)
		}
		return ErrNotInCharCreation
	default:
		return ErrUnknownMessageType
	}
}

// stripANSI removes ANSI escape sequences from raw JSON message bytes.
// Operates on the JSON itself, not decoded values — catches all ANSI
// regardless of where it was added (pkg/game, pkg/session, etc.).
// DP-GOAT P0-2: agent sessions receive clean text.
func stripANSI(msg []byte) []byte {
	result := make([]byte, 0, len(msg))
	for i := 0; i < len(msg); i++ {
		if msg[i] == '\x1b' && i+1 < len(msg) && msg[i+1] == '[' {
			// Skip past the escape sequence terminator (letter)
			for j := i + 2; j < len(msg); j++ {
				c := msg[j]
				if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
					i = j
					break
				}
			}
			continue
		}
		result = append(result, msg[i])
	}
	return result
}

// handleLogin authenticates a player.
