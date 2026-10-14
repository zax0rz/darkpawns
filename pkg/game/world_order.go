package game

import "sort"

// orderedCharacter is one body in C's global character_list. Player and
// mobile bodies live in separate Go registries, so sequence merges their
// insertion order at the points where C traverses character_list.
type orderedCharacter struct {
	player   *Player
	mob      *MobInstance
	sequence uint64
}

// cOrderedCharacters snapshots the live bodies in C's newest-first
// character_list order. create_char() and read_mobile() both prepend their
// bodies to that list (src/db.c:1716-1717,1745-1746), so newer sequence values
// come first. The name/VNum fallbacks make unregistered unit fixtures stable;
// world-owned bodies receive unique sequences in AddPlayer and spawnMob.
func (w *World) cOrderedCharacters() []orderedCharacter {
	w.mu.RLock()
	characters := make([]orderedCharacter, 0, len(w.players)+len(w.activeMobs))
	for _, player := range w.players {
		if player == nil {
			continue
		}
		characters = append(characters, orderedCharacter{player: player, sequence: player.characterSequence})
	}
	for _, mob := range w.activeMobs {
		if mob == nil {
			continue
		}
		characters = append(characters, orderedCharacter{mob: mob, sequence: mob.characterSequence})
	}
	w.mu.RUnlock()

	sort.SliceStable(characters, func(i, j int) bool {
		left, right := characters[i], characters[j]
		if left.sequence != right.sequence {
			return left.sequence > right.sequence
		}
		if left.player != nil && right.player != nil {
			return left.player.Name < right.player.Name
		}
		if left.mob != nil && right.mob != nil {
			if left.mob.GetID() != right.mob.GetID() {
				return left.mob.GetID() > right.mob.GetID()
			}
			return left.mob.GetVNum() < right.mob.GetVNum()
		}
		return left.player != nil
	})
	return characters
}
