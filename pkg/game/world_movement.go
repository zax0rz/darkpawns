package game

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// mobHasLight returns true if the mob has a lit light source equipped.
func mobHasLight(m *MobInstance) bool {
	if m == nil {
		return false
	}
	for _, item := range m.EquipmentSnapshot() {
		if isLitLightSource(item) {
			return true
		}
	}
	return false
}

func (w *World) CharTransfer(charName string, isMob bool, toRoomVNum int) error {
	return w.charTransfer(charName, isMob, toRoomVNum, true)
}

// charTransfer permits callers of bare char_from_room/char_to_room to leave
// mounts behind; command-level transfers retain their existing mount behavior.
func (w *World) charTransfer(charName string, isMob bool, toRoomVNum int, moveMount bool) error {
	var body combat.Combatant
	if isMob {
		for _, mob := range w.GetAllMobs() {
			if mob.GetName() == charName {
				body = mob
				break
			}
		}
	} else {
		if p, ok := w.GetPlayer(charName); ok {
			body = p
		}
	}
	if body == nil {
		return nil
	}
	return w.transferBody(body, toRoomVNum, moveMount)
}

// transferBody retains the supplied object; only name-taking entry selects once.
func (w *World) transferBody(body combat.Combatant, toRoomVNum int, moveMount bool) error {
	w.mu.RLock()
	_, exists := w.rooms[toRoomVNum]
	w.mu.RUnlock()
	if !exists {
		return fmt.Errorf("char_transfer: target room %d does not exist", toRoomVNum)
	}
	var mount *MobInstance
	if p, ok := body.(*Player); ok && moveMount {
		mount = w.riddenMount(p)
	}
	w.stopRoomFights(body)
	if mount != nil {
		w.stopRoomFights(mount)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	from := body.GetRoom()
	w.nextRoomEntrySequence++
	switch body := body.(type) {
	case *MobInstance:
		body.moveRoomLocked(w, toRoomVNum)
		body.mu.Lock()
		body.RoomEntrySequence = w.nextRoomEntrySequence
		body.mu.Unlock()
	case *Player:
		lit := body.HasLight()
		if lit && from >= 0 {
			w.adjustRoomLight(from, -1)
		}
		body.mu.Lock()
		body.RoomVNum = toRoomVNum
		body.RoomEntrySequence = w.nextRoomEntrySequence
		body.mu.Unlock()
		if lit {
			w.adjustRoomLight(toRoomVNum, 1)
		}
	}
	if mount != nil {
		mount.moveRoomLocked(w, toRoomVNum)
		w.nextRoomEntrySequence++
		mount.mu.Lock()
		mount.RoomEntrySequence = w.nextRoomEntrySequence
		mount.mu.Unlock()
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
	return w.transferBody(p, toRoomVNum, true)
}

// MobTransfer moves a mob to a new room, stopping fights.
func (w *World) MobTransfer(m *MobInstance, toRoomVNum int) error {
	return w.transferBody(m, toRoomVNum, true)
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
			cur = p.GetFollowing()
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
		if p.GetFollowing() == leaderName {
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

// TransferCombatant is the cycle-free spell bridge for an already selected body.
func (w *World) TransferCombatant(body combat.Combatant, room int) error {
	return w.transferBody(body, room, true)
}
