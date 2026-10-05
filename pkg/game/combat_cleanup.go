package game

import (
	"sort"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// stopRoomFights is char_from_room's pre-move boundary. No World lock is held
// while invoking the engine; ordinary nonfighters avoid a registry snapshot.
func (w *World) stopRoomFights(body combat.Combatant) {
	if body.GetFightingBody() == nil {
		return
	}
	room := body.GetRoom()
	var chars []combat.Combatant
	for _, c := range w.GetAllCharsInRoom(room) {
		if c, ok := c.(combat.Combatant); ok {
			chars = append(chars, c)
		}
	}
	sort.SliceStable(chars, func(i, j int) bool { return combatRoomSequence(chars[i]) > combatRoomSequence(chars[j]) })
	stop := func(c combat.Combatant) {
		if ce, ok := w.combatEngine.(interface{ StopCombat(combat.Combatant) }); ok {
			ce.StopCombat(c)
		} else {
			c.StopFighting()
		}
	}
	for _, c := range chars {
		if c != body && c.GetFightingBody() == body {
			stop(c)
		}
	}
	stop(body)
}

// retireCombatBody runs after releasing World.mu, retaining exact references
// even when the victim has already left the active registry.
func (w *World) retireCombatBody(body combat.Combatant) {
	if body == nil {
		return
	}
	// Only the retained NPC leader edges added in step 2 need new teardown.
	// Preserve existing outgoing follower/PC-leader teardown and its messages.
	if body.IsNPC() {
		for _, p := range w.GetAllPlayers() {
			if w.combatFollowingBody(p) == body {
				StopFollower(w, p)
			}
		}
		for _, m := range w.GetAllMobs() {
			if w.combatFollowingBody(m) == body {
				StopFollowerMob(w, m)
			}
		}
	}
	var others []combat.Combatant
	for _, p := range w.GetAllPlayers() {
		if p != body && p.GetFightingBody() == body {
			others = append(others, p)
		}
	}
	for _, m := range w.GetAllMobs() {
		if m != body && m.GetFightingBody() == body {
			others = append(others, m)
		}
	}
	if ce, ok := w.combatEngine.(interface{ RetireCombatant(combat.Combatant) }); ok {
		ce.RetireCombatant(body)
	} else {
		body.StopFighting()
		if marker, ok := body.(interface{ SetCombatRetired(bool) }); ok {
			marker.SetCombatRetired(true)
		}
	}
	for _, other := range others {
		if other.GetFightingBody() == body {
			if ce, ok := w.combatEngine.(interface{ StopCombat(combat.Combatant) }); ok {
				ce.StopCombat(other)
			} else {
				other.StopFighting()
			}
		}
	}
}
