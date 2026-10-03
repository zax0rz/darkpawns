package session

import "github.com/zax0rz/darkpawns/pkg/game"

// occupiedZoneRooms ports descriptor_list/CON_PLAYING in src/db.c:2290-2302.
// Snapshot attached bodies under Manager.mu, then read body rooms after release.
// Entry flag/body writes also take Manager.mu; no new lock is held across input.
// No lifecycle lock: detach can at worst leave one stale occupancy observation.
func (m *Manager) occupiedZoneRooms() []int {
	var players []*game.Player
	var mobs []*game.MobInstance
	m.mu.RLock()
	for _, s := range m.sessions {
		if s.player == nil || !s.authenticated || s.menuActive || s.charCreating || s.superseded.Load() || !s.hasTransport() || s.SendClosed() {
			continue
		}
		if s.isSwitched && s.switchedMob != nil {
			mobs = append(mobs, s.switchedMob)
		} else {
			players = append(players, s.player)
		}
	}
	m.mu.RUnlock()
	rooms := make([]int, 0, len(players)+len(mobs))
	for _, p := range players {
		rooms = append(rooms, p.GetRoomVNum())
	}
	for _, mob := range mobs {
		rooms = append(rooms, mob.GetRoom())
	}
	return rooms
}
