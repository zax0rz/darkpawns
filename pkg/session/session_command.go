// Package session manages WebSocket connections and player sessions.
package session

import "log/slog"
import "github.com/zax0rz/darkpawns/pkg/common"

func (m *Manager) RegisterCommand(name string, handler func(common.CommandSession, []string) error) {
	// This is a stub implementation
	// In a real implementation, this would register the command with the session manager
	slog.Debug("RegisterCommand called (stub)", "name", name)
}

// Sessions returns all active sessions
func (m *Manager) Sessions() []common.CommandSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]common.CommandSession, 0, len(m.sessions))
	for _, sess := range m.sessions {
		// Create a wrapper that implements common.CommandSession
		wrapper := &commandSessionWrapper{session: sess}
		sessions = append(sessions, wrapper)
	}
	return sessions
}

// commandSessionWrapper wraps a Session to implement common.CommandSession
type commandSessionWrapper struct {
	session *Session
}

func (w *commandSessionWrapper) Send(msg string) {
	w.session.Send(msg)
}

func (w *commandSessionWrapper) Close() {
	w.session.Close()
}

func (w *commandSessionWrapper) GetPlayer() interface{} {
	return w.session.GetPlayer()
}

func (w *commandSessionWrapper) GetPlayerName() string {
	return w.session.GetPlayerName()
}

func (w *commandSessionWrapper) GetPlayerRoomVNum() int {
	return w.session.GetPlayerRoomVNum()
}

func (w *commandSessionWrapper) IsAuthenticated() bool {
	return w.session.IsAuthenticated()
}

func (w *commandSessionWrapper) HasPlayer() bool {
	return w.session.HasPlayer()
}

// Lock locks the manager mutex
