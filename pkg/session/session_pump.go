// Package session manages WebSocket connections and player sessions.
package session

import (
	"encoding/json"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func (s *Session) readPump() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error(
				"CRITICAL PANIC RECOVERED in readPump",
				"player", s.playerName,
				"recover", r,
				"stack", string(debug.Stack()),
			)
			s.sendError("An internal server error occurred. Your connection has been reset.")
			s.manager.Unregister(s.playerName)
		}
		// Always decrement IP connection count (C5 leak fix)
		if !s.connCountDecremented && (s.request != nil || s.remoteIP != "") {
			s.connCountDecremented = true
			ip := s.RemoteIP()
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
		s.Close()
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

		// DP-GOAT P0-3: Clear takeover probe — any incoming message proves
		// this session is alive and should not be replaced.
		if s.takeOverPending.Load() {
			s.takeOverPending.Store(false)
			slog.Info("takeover probe cleared: session is alive", "player", s.playerName)
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
		if r := recover(); r != nil {
			slog.Error(
				"CRITICAL PANIC RECOVERED in writePump",
				"player", s.playerName,
				"recover", r,
				"stack", string(debug.Stack()),
			)
		}
		ticker.Stop()
		s.Close()
	}()

	for {
		select {
		case message, ok := <-s.send:
			_ = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// DP-GOAT P0-1 + P0-2: Stamp sequence number + strip ANSI for agents.
			// Unmarshal into generic map, transform, re-marshal. This avoids the
			// fragility of raw JSON string injection (old P0-1) and the broken
			// raw-byte ANSI strip (old P0-2 which couldn't match \u001b escapes).
			var raw map[string]interface{}
			if err := json.Unmarshal(message, &raw); err == nil {
				// P0-1: stamp sequence number on every outbound message
				s.msgSeq++
				raw["seq"] = s.msgSeq

				// P0-2: strip ANSI from all string values for agent sessions
				if s.isAgent {
					stripANSIRecursive(raw)
				}

				if marshaled, err := json.Marshal(raw); err == nil {
					message = marshaled
				}
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

// stripANSIRecursive walks a decoded JSON structure and strips ANSI escape
// sequences from every string value. Operates on decoded Go strings (not raw
// JSON bytes) so it correctly handles ESC bytes that json.Marshal encodes as
// \u001b — the previous raw-byte stripANSI could never match those.
//
// There is a nearly identical function in pkg/game/act_comm.go
// (deleteAnsiControls). That one operates on game-layer strings; this one
// handles arbitrary JSON-decoded structures for the agent protocol layer.
//
// DP-GOAT P0-2: agent sessions receive clean text.
func stripANSIRecursive(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		for k, child := range val {
			if s, ok := child.(string); ok {
				val[k] = stripANSIString(s)
			} else {
				stripANSIRecursive(child)
			}
		}
	case []interface{}:
		for i, child := range val {
			if s, ok := child.(string); ok {
				val[i] = stripANSIString(s)
			} else {
				stripANSIRecursive(child)
			}
		}
	}
}

// stripANSIString removes ANSI escape sequences from a Go string.
// Matches ESC[...{letter} sequences.
func stripANSIString(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip past the escape sequence terminator (letter A-Z or a-z)
			j := i + 2
			for j < len(s) {
				c := s[j]
				if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
					i = j + 1
					break
				}
				j++
			}
			if j >= len(s) {
				// Unterminated escape — skip the ESC
				b.WriteByte(s[i])
				i++
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// handleLogin authenticates a player.
