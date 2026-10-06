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

func (m *Manager) combatAudience(room int, exclude []combat.Combatant) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var audience []*Session
	for _, s := range m.sessions {
		var body combat.Combatant = s.player
		if s.isSwitched && s.switchedMob != nil {
			body = s.switchedMob
		}
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
