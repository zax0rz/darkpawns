// Package session manages WebSocket connections and player sessions.
package session

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/zax0rz/darkpawns/pkg/auth"
)

func (s *Session) maybeRefreshToken() {
	if s.tokenIssuedAt.IsZero() {
		return
	}
	remaining := time.Until(s.tokenIssuedAt.Add(jwtEffectiveLifetime))
	if remaining > jwtRefreshWindow {
		return
	}
	token, err := auth.GenerateJWT(s.player.Name, "")
	if err != nil {
		slog.Error("failed to refresh JWT token", "player", s.player.Name, "error", err)
		return
	}
	s.tokenIssuedAt = time.Now()
	msg, err := json.Marshal(ServerMessage{
		Type: MsgTokenRefresh,
		Data: map[string]string{"token": token},
	})
	if err != nil {
		slog.Error("json.Marshal token_refresh error", "error", err)
		return
	}
	if s.stageHeartbeat(msg, "", false) {
		return
	}
	select {
	case s.send <- msg:
	default:
		slog.Warn("dropping token_refresh: channel full", "player", s.player.Name)
	}
}

// Errors
func (m *Manager) CheckIdlePasswords() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var toDelete []string

	for name, s := range m.sessions {
		// Only check pre-login sessions (not yet authenticated)
		if s.authenticated {
			continue
		}

		if !s.idleTicsSet {
			s.idleTicsSet = true
			continue
		}

		// Timed out
		timeoutMsg, err := json.Marshal(ServerMessage{
			Type: MsgError,
			Data: ErrorData{Message: "\r\nTimed out... goodbye.\r\n"},
		})
		staged := err == nil && s.stageHeartbeat(timeoutMsg, "", false)
		if err == nil && !staged {
			select {
			case s.send <- timeoutMsg:
			default:
			}
		}

		// Close the connection
		if s.conn != nil && !staged {
			_ = s.conn.Close()
		}

		toDelete = append(toDelete, name)
	}

	for _, name := range toDelete {
		// Close channel and remove
		if s, ok := m.sessions[name]; ok {
			s.CloseSend()
			delete(m.sessions, name)
		}
	}

	if len(toDelete) > 0 {
		slog.Info("timed out idle pre-login sessions", "count", len(toDelete))
	}
}

// CountSessions returns the number of connected and playing sessions.
// Implements engine.UsageCounter for record_usage().
func (m *Manager) CountSessions() (connected int, playing int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, s := range m.sessions {
		connected++
		if s.authenticated && s.player != nil {
			playing++
		}
	}
	return
}

// WizlockLevel returns the C game_restrict-equivalent login threshold.
func (m *Manager) WizlockLevel() int {
	m.wizlockMutex.Lock()
	defer m.wizlockMutex.Unlock()
	return m.wizlockLevel
}
