package session

import (
	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// combatSession snapshots existing descriptor ownership. No name lookup or
// lifecycle lock; delivery occurs after releasing the manager lock.
func (m *Manager) combatSession(body combat.Combatant) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := body.(*game.Player); ok {
		return m.attachedBodyLocked(p)
	}
	mob, ok := body.(*game.MobInstance)
	if !ok || mob == nil {
		return nil
	}
	for _, s := range m.sessions {
		if s.isSwitched && s.switchedMob == mob && s.hasTransport() && !s.SendClosed() {
			return s
		}
	}
	return nil
}

// mobileDescriptorOwner returns the original player whose descriptor is
// attached to this NPC body. The player stays in the world registry for
// identity and persistence, but C detaches its descriptor while switched.
func (m *Manager) mobileDescriptorOwner(body *game.MobInstance) *game.Player {
	if body == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.sessions {
		if s.isSwitched && s.switchedMob == body && s.player != nil {
			return s.player
		}
	}
	return nil
}

// combatBody is the session's combat actor: the switched-into mob while
// switched, otherwise the player body. Nil when the session has no body yet.
func (s *Session) combatBody() combat.Combatant {
	if s.isSwitched && s.switchedMob != nil {
		return s.switchedMob
	}
	if s.player == nil {
		return nil
	}
	return s.player
}

// commandBodyLocked returns the body attached to this session's descriptor.
// Callers that share session lifecycle state must hold manager.mu; detached
// test sessions have no concurrent lifecycle to synchronize.
func (s *Session) commandBodyLocked() combat.Combatant {
	if s.isSwitched {
		if s.switchedMob != nil {
			return s.switchedMob
		}
		if s.switchedPlayer != nil {
			return s.switchedPlayer
		}
	}
	if s.player != nil {
		return s.player
	}
	return nil
}

func (s *Session) descriptorWaitState() int {
	if s == nil {
		return 0
	}
	if s.manager != nil {
		s.manager.mu.RLock()
		defer s.manager.mu.RUnlock()
	}
	waiter, ok := s.commandBodyLocked().(interface{ GetWaitState() int })
	if !ok {
		return 0
	}
	return waiter.GetWaitState()
}

func (m *Manager) combatAudience(room int, exclude []combat.Combatant) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var audience []*Session
	for _, s := range m.sessions {
		body := s.combatBody()
		if body == nil || !s.hasTransport() || s.SendClosed() || s.menuActive || s.charCreating || body.GetRoom() != room || body.GetPosition() <= combat.PosSleeping {
			continue
		}
		omitted := false
		for _, excluded := range exclude {
			if body == excluded {
				omitted = true
				break
			}
		}
		if !omitted {
			audience = append(audience, s)
		}
	}
	return audience
}
