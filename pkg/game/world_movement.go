package game

import (
	"fmt"
)

func (w *World) CharTransfer(charName string, isMob bool, toRoomVNum int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, ok := w.rooms[toRoomVNum]; !ok {
		return fmt.Errorf("char_transfer: target room %d does not exist", toRoomVNum)
	}

	// Find current room
	var fromRoomVNum int
	if isMob {
		for _, m := range w.activeMobs {
			if m.GetName() == charName {
				fromRoomVNum = m.GetRoom()
				break
			}
		}
	} else {
		if p, ok := w.players[charName]; ok {
				fromRoomVNum = p.RoomVNum
		}
	}

	if fromRoomVNum == toRoomVNum {
		return nil // already there
	}

	// Stop fighting for everyone who was fighting the transferee in the old room
	// Source: char_from_room iterates world[room].people to stop mutual fights
	if fromRoomVNum >= 0 {
		// Stop the transferee from fighting
		if isMob {
			for _, m := range w.activeMobs {
				if m.GetName() == charName {
					m.StopFighting()
					break
				}
			}
		} else {
			if p, ok := w.players[charName]; ok {
				p.StopFighting()
			}
		}

		// Stop anyone in the old room from fighting the transferee
		for _, p := range w.players {
			if p.RoomVNum == fromRoomVNum && p.IsFighting() && p.GetFighting() == charName {
				p.StopFighting()
		}
		}
		for _, m := range w.activeMobs {
			if m.GetRoom() == fromRoomVNum && m.GetFighting() == charName {
				m.StopFighting()
			}
		}
	}

	// Move the character
	if isMob {
		for _, m := range w.activeMobs {
			if m.GetName() == charName {
				m.SetRoom(toRoomVNum)
				break
			}
		}
	} else {
		if p, ok := w.players[charName]; ok {
			p.SetRoom(toRoomVNum)

			// Move mount with rider (recall/teleport take mounts)
			// Source: act.wizard.c do_recall moves get_mount(ch) with the player
			if p.MountName != "" {
				for _, m := range w.activeMobs {
					if m.GetName() == p.MountName && m.GetRoom() == fromRoomVNum {
						m.SetRoom(toRoomVNum)
						break
					}
				}
			}
		}
	}

	return nil
}

// GetAllCharsInRoom returns all characters (players and mobs) in a room.
// This is the Go equivalent of iterating world[room].people.
func (w *World) GetAllCharsInRoom(roomVNum int) []interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var chars []interface{}
	for _, p := range w.players {
		if p.RoomVNum == roomVNum {
			chars = append(chars, p)
		}
	}
	for _, m := range w.activeMobs {
		if m.GetRoom() == roomVNum {
			chars = append(chars, m)
		}
	}
	return chars
}

// PlayerTransfer moves a player to a new room, stopping fights and moving mounts.
func (w *World) PlayerTransfer(p *Player, toRoomVNum int) error {
	return w.CharTransfer(p.GetName(), false, toRoomVNum)
}

// MobTransfer moves a mob to a new room, stopping fights.
func (w *World) MobTransfer(m *MobInstance, toRoomVNum int) error {
	return w.CharTransfer(m.GetName(), true, toRoomVNum)
}

// GetItemsInRoom returns all items in a given room.

// AddFollowerQuietInterface adds a follower via interface{} params (for spell layer access).
func (w *World) AddFollowerQuiet(ch, leader interface{}) {
	switch c := ch.(type) {
	case *Player:
		switch l := leader.(type) {
		case *Player:
			AddFollowerQuiet(c, l)
		}
	case *MobInstance:
		switch l := leader.(type) {
		case *Player:
			AddFollowerQuietMob(c, l)
		}
	}
}

// StopFollowerByName removes a named character from following via string lookup.
func (w *World) StopFollowerByName(name string) {
	if p, ok := w.players[name]; ok {
		StopFollower(w, p)
		return
	}
	for _, m := range w.activeMobs {
		if m.GetName() == name {
			StopFollowerMob(w, m)
			return
		}
	}
}

// CircleFollowByName checks if following would create a loop via string names.
func (w *World) CircleFollowByName(followerName, leaderName string) bool {
	ch, chOk := w.players[followerName]
	victim, vOk := w.players[leaderName]
	if chOk && vOk {
		return CircleFollow(w, ch, victim)
	}
	// Simple chain walk for mob followers
	cur := leaderName
	for {
		if cur == followerName {
			return true
		}
		if p, ok := w.players[cur]; ok {
			cur = p.Following
			if cur == "" {
				return false
			}
			continue
		}
		return false
	}
}

// NumFollowers returns the count of characters following leaderName.
func (w *World) NumFollowers(leaderName string) int {
	count := 0
	for _, p := range w.players {
		if p.Following == leaderName {
			count++
		}
	}
	for _, m := range w.activeMobs {
		if m.GetFollowing() == leaderName {
			count++
		}
	}
	return count
}
