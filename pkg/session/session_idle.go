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

	var toDelete []string
	var timedOut []struct {
		sess   *Session
		staged bool
	}

	for name, s := range m.sessions {
		// C src/comm.c:2170-2186: check_idle_passwords checks CON_PASSWORD,
		// CON_GET_NAME, and CON_MENU (including authenticated sessions
		// returned to the menu after death).
		inMenu := s.menuActive && s.menuStage == "menu"
		if s.authenticated && !inMenu {
			continue
		}

		if !s.idleTicsSet {
			s.idleTicsSet = true
			continue
		}

		// Timed out (C src/comm.c:2183: SEND_TO_Q("\r\nTimed out... goodbye.\r\n", d))
		timeoutMsg, err := json.Marshal(ServerMessage{
			Type: MsgEvent,
			Data: EventData{
				Type: "raw",
				Text: "\r\nTimed out... goodbye.\r\n",
			},
		})
		staged := err == nil && s.stageHeartbeat(timeoutMsg, "", false)
		if err == nil && !staged {
			select {
			case s.send <- timeoutMsg:
			default:
			}
		}
		if staged {
			// Queue the heartbeat close barrier synchronously before
			// EndOutputTurn runs, then wait for the writer to drain and close
			// the underlying transport asynchronously so we don't deadlock
			// EndOutputTurn.
			s.CloseSend()
		}

		toDelete = append(toDelete, name)
		timedOut = append(timedOut, struct {
			sess   *Session
			staged bool
		}{sess: s, staged: staged})
	}

	for _, name := range toDelete {
		delete(m.sessions, name)
	}
	m.mu.Unlock()

	for _, item := range timedOut {
		if item.staged {
			go func(s *Session) {
				if s.drainBeforeCloseFunc != nil {
					s.drainBeforeCloseFunc()
				} else if s.writerDone != nil {
					<-s.writerDone
				}
				s.Close()
			}(item.sess)
			continue
		}
		item.sess.CloseAfterSend()
	}

	if len(toDelete) > 0 {
		slog.Info("timed out idle pre-login sessions", "count", len(toDelete))
	}
}

// CountSessions returns the number of connected and playing sessions.
// Implements engine.UsageCounter for record_usage().
func (m *Manager) CountSessions() (connected int, playing int) {
	m.mu.RLock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.mu.RUnlock()

	for _, s := range sessions {
		if !s.hasDescriptor() {
			continue
		}
		connected++
		if s.isConPlaying() {
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
