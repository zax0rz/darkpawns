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
	// Follower acts belong to final extraction, before room/registry removal
	// (src/handler.c:1116-1117), not early combat retirement.
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

// stopRawKillBody is raw_kill's own fighting stop (src/fight.c:541-542): the
// dying body leaves combat, and nobody else's fighting is touched. extract_char
// only flags the body (src/handler.c:1194-1209), so survivors keep pointing at
// it — and keep swinging at the flagged corpse — until extract_char_final at
// the next heartbeat (src/handler.c:1147-1151), which the extraction pass's
// retireCombatBody mirrors.
func (w *World) stopRawKillBody(body combat.Combatant) {
	if body == nil {
		return
	}
	if ce, ok := w.combatEngine.(interface{ StopCombat(combat.Combatant) }); ok {
		ce.StopCombat(body)
	} else {
		body.StopFighting()
	}
	if marker, ok := body.(interface{ SetCombatRetired(bool) }); ok {
		marker.SetCombatRetired(true)
	}
}

// stopKillerAfterDie is die_with_killer's stop_fighting(killer) (src/fight.c:
// 615-616): the killer leaves combat when its victim dies, retargeting through
// C's dead-opponent scan when another fighter names it. raw_kill-only paths
// (do_kill's instakill) never stop the killer — its FIGHTING stays on the
// flagged corpse until the extraction pass.
func (w *World) stopKillerAfterDie(killer combat.Combatant) {
	if killer == nil {
		return
	}
	if ce, ok := w.combatEngine.(interface{ StopCombat(combat.Combatant) }); ok {
		ce.StopCombat(killer)
	} else {
		killer.StopFighting()
	}
}
