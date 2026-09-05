// Package session manages WebSocket connections and player sessions.
package session

import (
	"encoding/json"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func (m *Manager) Lock() {
	m.mu.Lock()
}

// Unlock unlocks the manager mutex
func (m *Manager) Unlock() {
	m.mu.Unlock()
}

// RLock locks the manager mutex for reading
func (m *Manager) RLock() {
	m.mu.RLock()
}

// RUnlock unlocks the manager mutex for reading
func (m *Manager) RUnlock() {
	m.mu.RUnlock()
}

// Mu returns the mutex for synchronization
func (m *Manager) Mu() interface{} {
	return &m.mu
}

// ---------------------------------------------------------------------------
// Session lifecycle and communication — ported from comm.c
// ---------------------------------------------------------------------------

// UnregisterAndClose removes a session for the specified player and cleans up
// all associated resources. This is the Go equivalent of close_socket() from
// comm.c. It handles:
//   - Flushing queues (input/output)
//   - Closing the WebSocket connection
//   - Saving player state if CON_PLAYING
//   - Notifying the room of departure
//   - Removing from the sessions map
//   - Freeing compression/showstr state (not applicable in Go version)
func (m *Manager) UnregisterAndClose(playerName string) {
	m.mu.Lock()
	s, ok := m.sessions[playerName]
	if ok {
		delete(m.sessions, playerName)
	}
	m.mu.Unlock()

	if !ok || s == nil {
		slog.Warn("unregister and close: session not found", "player", playerName)
		return
	}

	// Flush any pending output
	s.FlushQueues()

	// Decrement per-IP connection count (C5)
	if !s.connCountDecremented && (s.request != nil || s.remoteIP != "") {
		s.connCountDecremented = true
		ip := s.RemoteIP()
		m.ipConnMu.Lock()
		m.ipConnCount[ip]--
		if m.ipConnCount[ip] <= 0 {
			delete(m.ipConnCount, ip)
		}
		m.ipConnMu.Unlock()
	}

	m.cleanupSession(s, playerName)

	// Close the WebSocket connection
	if s.conn != nil {
		_ = s.conn.Close()
	}

	slog.Info("session closed", "player", playerName)
}

// FlushQueues drains any pending input/output for a session.
// In the WebSocket Go version this is a no-op for input (handled by readPump), but we
// keep the method for compatibility with the flush_queues() semantics.
// Ported from comm.c:flush_queues().
func (s *Session) FlushQueues() {
	// Drain the send channel (pending output)
	for {
		select {
		case <-s.send:
		default:
			return
		}
	}
}

// SendToAll sends a text message to all connected, playing sessions.
// Ported from comm.c:send_to_all().
func (m *Manager) SendToAll(message string) {
	m.sendToPlaying(message, "broadcast", "SendToAll", nil)
}

// SendToOutdoor sends a message to all playing sessions whose characters are
// awake and in an outdoor room (Sector > 0, i.e. not SECT_INSIDE).
// Ported from comm.c:send_to_outdoor().
func (m *Manager) SendToOutdoor(message string) {
	m.sendToPlaying(message, "outdoor", "SendToOutdoor", func(s *Session) bool {
		// AWAKE check: position >= PosStanding
		if s.player.GetPosition() < combat.PosStanding {
			return false
		}
		// OUTSIDE check: sector type != INSIDE (0)
		roomVNum := s.player.GetRoom()
		if room, ok := m.world.GetRoom(roomVNum); ok && room.Sector == 0 {
			return false // SECT_INSIDE
		}
		return true
	})
}

func (m *Manager) sendToPlaying(message, eventType, label string, eligible func(*Session) bool) {
	if message == "" {
		return
	}

	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: eventType,
			Text: message,
		},
	})
	if err != nil {
		slog.Error(label+" marshal error", "error", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, s := range m.sessions {
		if s.player == nil || !s.authenticated {
			continue
		}
		if eligible != nil && !eligible(s) {
			continue
		}
		select {
		case s.send <- msg:
		default:
			slog.Debug(label+": dropping message to full channel", "player", s.playerName)
		}
	}
}

// CheckIdlePasswords checks for idle pre-login sessions (not yet fully connected)
// and disconnects them if they have been idle for more than one tick cycle.
// Ported from comm.c:check_idle_passwords().
//
// In the Go WebSocket version, a session is considered "pre-login" if authenticated is false
// (i.e. they haven't completed login yet). The idleTics counter is checked:
// - First idle tick: increment counter
// - Second idle tick: send timeout message and mark for close
