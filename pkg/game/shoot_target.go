package game

import (
	"sort"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ResolveShootTarget implements get_char_room followed by do_shoot's fallback
// (src/handler.c:865-881; src/act.offensive.c:862-864). This lookup intentionally
// has no viewer: C does not apply CAN_SEE, self/me or player-only 0. semantics.
func (w *World) ResolveShootTarget(roomVNum int, name string) combat.Combatant {
	type occupant struct {
		body     combat.Combatant
		sequence uint64
		keywords string
	}
	// Snapshot the registry without holding the world lock while reading bodies.
	w.mu.RLock()
	players := make([]*Player, 0, len(w.players))
	for _, p := range w.players {
		players = append(players, p)
	}
	mobs := make([]*MobInstance, 0, len(w.activeMobs))
	for _, m := range w.activeMobs {
		mobs = append(mobs, m)
	}
	w.mu.RUnlock()
	var people []occupant
	for _, p := range players {
		if p.GetRoom() == roomVNum {
			people = append(people, occupant{p, p.GetRoomEntrySequence(), p.GetName()})
		}
	}
	for _, m := range mobs {
		if m.GetRoom() == roomVNum {
			people = append(people, occupant{m, m.GetRoomEntrySequence(), charKeywords(m)})
		}
	}
	sort.Slice(people, func(i, j int) bool { return people[i].sequence > people[j].sequence })
	number := GetNumber(&name)
	if number != 0 {
		matched := 0
		for _, person := range people {
			if isnameWithAbbrevs(name, person.keywords) {
				matched++
				if matched == number {
					return person.body
				}
			}
		}
	}
	if len(people) > 0 {
		return people[0].body
	}
	return nil
}
