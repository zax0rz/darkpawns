package session

import "github.com/zax0rz/darkpawns/pkg/game"

// attachedBody is the descriptor attached to this concrete character, not the
// retained identity/lifecycle owner. C ch->desc (src/act.wizard.c:1192).
// This lookup does not acquire the world or lifecycle lock.
func (m *Manager) attachedBody(p *game.Player) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.attachedBodyLocked(p)
}

// attachedBodyLocked requires m.mu. A retained linkdead session is not a C
// descriptor. A switched original has no descriptor even though its identity
// remains a registry key.
func (m *Manager) attachedBodyLocked(p *game.Player) *Session {
	for _, s := range m.sessions {
		if s.player == p && s.hasTransport() && !s.SendClosed() && !s.menuActive && !s.charCreating {
			if s.isSwitched && s.switchedMob != nil {
				continue
			}
			return s
		}
	}
	return nil
}
