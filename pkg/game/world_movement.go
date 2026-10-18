package game

import (
	"fmt"
	"sort"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/spells"
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
	players := w.GetPlayersInRoom(roomVNum)
	mobs := w.GetMobsInRoom(roomVNum)
	chars := make([]interface{}, 0, len(players)+len(mobs))
	for _, p := range players {
		chars = append(chars, p)
	}
	for _, m := range mobs {
		chars = append(chars, m)
	}
	sort.SliceStable(chars, func(i, j int) bool {
		return combatRoomSequence(chars[i].(combat.Combatant)) > combatRoomSequence(chars[j].(combat.Combatant))
	})
	return chars
}

// PlayerTransfer moves a player to a new room, stopping fights and moving mounts.
func (w *World) PlayerTransfer(p *Player, toRoomVNum int) error {
	return w.transferBody(p, toRoomVNum, true)
}

// PlayerTransferForSpell mirrors spell-side char_from_room/char_to_room by
// moving only the player body. Those C helpers do not transfer a ridden mount.
func (w *World) PlayerTransferForSpell(actor interface{}, toRoomVNum int) error {
	p, ok := actor.(*Player)
	if !ok || p == nil {
		return fmt.Errorf("spell player transfer: invalid actor")
	}
	if err := w.transferBody(p, toRoomVNum, false); err != nil {
		return err
	}
	// char_to_room calls circle_check after placing the character (src/handler.c:553).
	w.consumeRoomSummoningCircle(toRoomVNum)
	return nil
}

// CharacterTransferForSpell mirrors spell_summon's char_from_room /
// char_to_room path, including the destination circle check and its explicit
// mount break after transfer. It accepts the spell package's interface actor
// without weakening the concrete command transfer APIs.
func (w *World) CharacterTransferForSpell(actor interface{}, toRoomVNum int) error {
	// char_from_room sets in_room to NOWHERE before calling circle_check, so
	// that source-side call cannot consume an origin-room circle (handler.c:527,
	// utils.c:877-891). char_to_room's destination check does consume one.
	switch body := actor.(type) {
	case *Player:
		if err := w.PlayerTransferForSpell(body, toRoomVNum); err != nil {
			return err
		}
		if actorHasMountAffect(body) {
			w.unmountActor(body)
		}
		return nil
	case *MobInstance:
		if err := w.transferBody(body, toRoomVNum, false); err != nil {
			return err
		}
		w.consumeRoomSummoningCircle(toRoomVNum)
		if actorHasMountAffect(body) {
			w.unmountActor(body)
		}
		if rider := w.GetRider(body); rider != nil {
			w.unmountActor(rider)
		}
		return nil
	default:
		return fmt.Errorf("spell character transfer: unsupported actor %T", actor)
	}
}

func (w *World) consumeRoomSummoningCircle(roomVNum int) {
	for _, item := range w.GetItemsInRoom(roomVNum) {
		if item.GetVNum() == spells.CocVnum {
			item.SetTimer(item.GetTimer() - 1)
			return
		}
	}
}

// MindsightNotifyWatchers sends spell_mindsight's direct observer message to
// other player characters in the caster's current room (src/spells.c:945-947).
func (w *World) MindsightNotifyWatchers(actor interface{}) {
	caster, ok := actor.(*Player)
	if !ok || caster == nil {
		return
	}
	if class := caster.GetClass(); class != ClassPsionic && class != ClassMystic {
		return
	}
	for _, observer := range w.GetPlayersInRoom(caster.GetRoom()) {
		if observer != caster {
			observer.SendMessage("You feel like you're being watched.\r\n")
		}
	}
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

// AddFollowerForSpell applies add_follower's link and audience messages for a
// manual spell target. C spell_charm calls the noisy add_follower, so using the
// quiet helpers here drops its target, leader, and room notices.
func (w *World) AddFollowerForSpell(follower, leader interface{}) bool {
	followerBody := combatantFromInterface(follower)
	leaderBody := combatantFromInterface(leader)
	if followerBody == nil || leaderBody == nil {
		return false
	}

	switch ch := followerBody.(type) {
	case *Player:
		leaderActor := asActor(leaderBody)
		ch.SetFollowingBody(leaderBody)
		Act(w, false, ch, leaderActor, nil, nil, "You now follow $N.", "", ToChar)
		if leaderActor != nil && canSee(leaderActor, ch) && leaderActor.GetPosition() > combat.PosSleeping {
			Act(w, true, ch, leaderActor, nil, nil, "$n starts following you.", "", ToVict)
		}
		Act(w, true, ch, leaderActor, nil, nil, "$n starts to follow $N.", "", ToNotVict)
		return true
	case *MobInstance:
		addFollowerMobToBody(w, ch, leaderBody)
		return true
	default:
		return false
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
