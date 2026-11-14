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

// mobRNumForVNum mirrors real_mobile (src/db.c): the prototype's index in the
// mob table. Keeper rnum 0 is Puff (the first mobile in the database), which
// is the only index a player's zeroed nr can collide with.
func (w *World) mobRNumForVNum(vnum int) int {
	parsed := w.GetParsedWorld()
	if parsed == nil {
		return -1
	}
	for i := range parsed.Mobs {
		if parsed.Mobs[i].VNum == vnum {
			return i
		}
	}
	return -1
}

// ShowShopsKeeper mirrors C's show_shops keeper resolution
// (src/shop.c:1345-1350): get_char_num(SHOP_KEEPER(shop_nr)) scans
// character_list newest-first for the first body whose nr equals the keeper's
// prototype rnum. Players never set nr -- clear_char zeroes it (src/db.c),
// and create_char prepends players to the same list -- so any live player
// collides with keeper rnum 0, and the logged-in actor's gold is what C
// prints as the keeper's Coins. Exactly one return is non-nil on a match.
func (w *World) ShowShopsKeeper(keeperVNum int) (player *Player, mob *MobInstance) {
	keeperRNum := w.mobRNumForVNum(keeperVNum)
	if keeperRNum < 0 {
		return nil, nil
	}
	for _, oc := range w.cOrderedCharacters() {
		if oc.player != nil {
			if keeperRNum == 0 {
				return oc.player, nil
			}
			continue
		}
		if oc.mob != nil && w.mobRNumForVNum(oc.mob.GetVNum()) == keeperRNum {
			return nil, oc.mob
		}
	}
	return nil, nil
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

// playingPlayersInDescriptorOrder returns the playing players in C's
// descriptor_list order (newest connection first): full_moon and lunar_hunter
// traverse descriptor_list (src/new_cmds.c:1314-1316,2668-2670), which
// comm.c:1591-1603 prepends, so the newest connection iterates first. The
// players' relative characterSequence order matches their descriptor order
// because each character follows its descriptor at login, so filtering the
// newest-first character_list snapshot recovers it. The playing-descriptor
// predicate is C's !i->connected && i->character.
func (w *World) playingPlayersInDescriptorOrder() []*Player {
	ordered := w.cOrderedCharacters()
	players := make([]*Player, 0, len(ordered))
	for _, entry := range ordered {
		if entry.player != nil && w.playerHasPlayingDescriptor(entry.player) {
			players = append(players, entry.player)
		}
	}
	return players
}

// GetAllCharsInCharacterListOrder returns every live player and mobile in
// C's global character_list order. Area effects that traverse character_list
// use this instead of room people order, then apply their own room predicate.
func (w *World) GetAllCharsInCharacterListOrder() []any {
	ordered := w.cOrderedCharacters()
	characters := make([]any, 0, len(ordered))
	for _, body := range ordered {
		if body.player != nil {
			characters = append(characters, body.player)
		} else if body.mob != nil {
			characters = append(characters, body.mob)
		}
	}
	return characters
}
