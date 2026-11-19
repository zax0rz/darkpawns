package session

import "github.com/zax0rz/darkpawns/pkg/game"

// playingDescriptorBodies returns d->character in C's descriptor_list order.
// C prepends each accepted descriptor, so the newest connection comes first.
// The snapshot intentionally carries a switched character body when one is
// active, since hunt_items reads i->character rather than the account owner.
func (m *Manager) playingDescriptorBodies() []game.Actor {
	m.mu.RLock()
	sessions := m.sessionsInDescriptorOrderLocked()
	bodies := make([]game.Actor, 0, len(sessions))
	for _, s := range sessions {
		if !s.isConPlayingLocked() {
			continue
		}
		var body game.Actor = s.player
		if s.isSwitched {
			switch {
			case s.switchedMob != nil:
				body = s.switchedMob
			case s.switchedPlayer != nil:
				body = s.switchedPlayer
			}
		}
		if body != nil {
			bodies = append(bodies, body)
		}
	}
	m.mu.RUnlock()
	return bodies
}
