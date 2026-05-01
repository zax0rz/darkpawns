// Package session manages WebSocket connections and player sessions.
package session

import (
	"log/slog"
	"time"
	"encoding/json"
)
import "github.com/gorilla/websocket"

func (s *Session) readPump() {
	defer func() {
		s.manager.Unregister(s.playerName)
// #nosec G104
		s.conn.Close()
	}()

	s.conn.SetReadLimit(16384) // 16KB max message size (C4)
// #nosec G104
	s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.conn.SetPongHandler(func(string) error {
// #nosec G104
		s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
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
			s.sendError(err.Error())
		}
	}
}

// writePump writes messages to the WebSocket.
func (s *Session) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
// #nosec G104
		s.conn.Close()
	}()

	for {
		select {
		case message, ok := <-s.send:
// #nosec G104
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
// #nosec G104
				s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
// #nosec G104
			s.conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
// #nosec G104
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
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

// handleLogin authenticates a player.
