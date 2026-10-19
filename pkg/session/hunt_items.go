package session

import (
	"sort"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// playingDescriptorBodies returns d->character in C's descriptor_list order.
// C prepends each accepted descriptor, so the newest connection comes first.
// The snapshot intentionally carries a switched character body when one is
// active, since hunt_items reads i->character rather than the account owner.
func (m *Manager) playingDescriptorBodies() []game.Actor {
	type descriptorBody struct {
		body        game.Actor
		connectedAt int64
		connection  int
		name        string
	}

	m.mu.RLock()
	descriptors := make([]descriptorBody, 0, len(m.sessions))
	for _, s := range m.sessions {
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
			descriptors = append(descriptors, descriptorBody{
				body: body, connectedAt: s.connectedAt.UnixNano(),
				connection: s.connectionNumber, name: body.GetName(),
			})
		}
	}
	m.mu.RUnlock()

	sort.SliceStable(descriptors, func(i, j int) bool {
		if descriptors[i].connectedAt != descriptors[j].connectedAt {
			return descriptors[i].connectedAt > descriptors[j].connectedAt
		}
		if descriptors[i].connection != descriptors[j].connection {
			return descriptors[i].connection > descriptors[j].connection
		}
		return descriptors[i].name < descriptors[j].name
	})
	bodies := make([]game.Actor, len(descriptors))
	for i := range descriptors {
		bodies[i] = descriptors[i].body
	}
	return bodies
}
