// Package session manages WebSocket connections and player sessions.
package session

import (
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/command"
)

func (m *Manager) RegisterCommand(name string, handler func(common.CommandSession, []string) error) {
	// Bridge common.CommandSession handler into cmdRegistry.
	cmdRegistry.Register(name, command.Handler(handler), name+" (registered via RegisterCommand)", 0, 0)
	slog.Debug("RegisterCommand: registered", "name", name)
}

// Sessions returns all active sessions
func (m *Manager) Sessions() []common.CommandSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]common.CommandSession, 0, len(m.sessions))
	for _, sess := range m.sessions {
		sessions = append(sessions, &commandSessionWrapper{session: sess})
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

func (w *commandSessionWrapper) Lock()   {}
func (w *commandSessionWrapper) Unlock() {}

func (w *commandSessionWrapper) GetManager() interface{} {
	return w.session.manager
}
