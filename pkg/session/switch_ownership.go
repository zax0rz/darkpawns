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
		body.SetSwitchedOriginal(nil)
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

// switchedIdentityOwner is the holder of character-owned session metadata.
// Only active switching enters this selection; false retains main's reader.
func (m *Manager) switchedIdentityOwner(p *game.Player) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	involved := false
	for _, s := range m.sessions {
		if !s.activePCSwitch() {
			continue
		}
		if s.switchedOriginal == p {
			return s, true
		}
		involved = involved || s.switchedPlayer == p
	}
	if involved {
		for _, s := range m.sessions {
			if s.player == p && !s.activePCSwitch() {
				return s, true
			}
		}
	}
	return nil, involved
}

func (s *Session) actingOLCZone() int {
	if s.manager == nil {
		return s.olcZone
	}
	m := s.manager
	m.mu.RLock()
	p, zone, switched := s.player, s.olcZone, s.activePCSwitch()
	m.mu.RUnlock()
	if switched {
		if owner, involved := m.switchedIdentityOwner(p); involved && owner != nil {
			m.mu.RLock()
			zone = owner.olcZone
			m.mu.RUnlock()
		}
	}
	return zone
}

// switchedAwayOriginal reports whether playerName addresses the original
// body of a session currently switched into another body. C's do_switch
// detaches that body's descriptor for NPC and PC riders alike
// (act.wizard.c:1199-1200, ch->desc = NULL), so a send_to_char addressed to
// the switched-away original drops at the descriptor gate.
func (m *Manager) switchedAwayOriginal(s *Session, playerName string) bool {
	return s != nil && s.isSwitched && s.switchedOriginal != nil &&
		strings.EqualFold(playerName, s.switchedOriginal.Name)
}

// sessionRidingMob finds the session whose descriptor rides this NPC body.
func (m *Manager) sessionRidingMob(mob *game.MobInstance) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.sessions {
		if s.isSwitched && s.switchedMob == mob {
			return s
		}
	}
	return nil
}

// returnSwitchedMob is extract_char's do_return arm for a body hosting a
// descriptor (src/handler.c:1158-1159): the rider returns to the original
// body before the extracted character is freed. die_follower has already run
// (src/handler.c:1116-1117), matching C's line order. The world extraction
// pass runs under playerLifecycleMu (Manager.ExtractPendingChars and
// drainPendingExtractions both hold it before draining), so this must not
// re-lock it.
func (m *Manager) returnSwitchedMob(mob *game.MobInstance) {
	s := m.sessionRidingMob(mob)
	if s == nil {
		return
	}
	_ = m.returnSwitch(s)
}
