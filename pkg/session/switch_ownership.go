package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

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

// activePCSwitch is read/written under m.mu or playerLifecycleMu. NPC command
// dispatch remains the separately owned C1 frontier.
func (s *Session) activePCSwitch() bool {
	return s.isSwitched && s.switchedOriginal != nil && s.switchedPlayer != nil
}

// bodySessionByName changes routing only for identities involved in an active
// PC switch. With no active switch, GetSession's main behavior is unchanged.
func (m *Manager) bodySessionByName(name string) (*Session, bool) {
	if attached, involved := m.switchDescriptorByName(name); involved {
		return attached, attached != nil && attached.hasTransport() && !attached.SendClosed()
	}
	return m.GetSession(name)
}

// detachPCSwitch requires playerLifecycleMu. The registry retains both bodies;
// restoring the descriptor's original does not extract its borrowed body.
func (m *Manager) detachPCSwitch(s *Session, connected bool) {
	m.mu.Lock()
	original, body := s.switchedOriginal, s.switchedPlayer
	s.player = original
	s.isSwitched = false
	s.switchedOriginal = nil
	s.switchedOriginalLevel = 0
	s.switchedPlayer = nil
	s.switchedMob = nil
	m.mu.Unlock()
	if body != nil {
		body.SetLinkless(true)
	}
	if original != nil {
		original.SetLinkless(!connected)
	}
}

// switchDescriptor changes only active-switch readers; false means use the
// original main path verbatim. Caller must not hold m.mu.
func (m *Manager) switchDescriptor(p *game.Player) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.switchDescriptorLocked(p)
}

func (m *Manager) switchDescriptorLocked(p *game.Player) (*Session, bool) {
	original := false
	for _, s := range m.sessions {
		if !s.activePCSwitch() {
			continue
		}
		if s.switchedPlayer == p {
			return s, true
		}
		if s.switchedOriginal == p {
			original = true
		}
	}
	return nil, original
}

func (m *Manager) switchDescriptorByName(name string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	original := false
	for _, s := range m.sessions {
		if !s.activePCSwitch() {
			continue
		}
		if strings.EqualFold(s.switchedPlayer.GetName(), name) {
			return s, true
		}
		if strings.EqualFold(s.switchedOriginal.GetName(), name) {
			original = true
		}
	}
	return nil, original
}
