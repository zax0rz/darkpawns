package game

import (
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// isnameWithAbbrevs is a faithful port of C isname_with_abbrevs
// (src/handler.c:115-134). It tokenizes namelist on whitespace and returns
// true if isAbbrev(str, token) matches any token. This is the matcher used by
// get_char_room_vis against a character's keyword namelist. (isAbbrev itself
// is defined in house_player.go and is a faithful is_abbrev port.)
func isnameWithAbbrevs(str, namelist string) bool {
	if namelist == "" {
		return false
	}
	str = strings.ToLower(str)
	for _, tok := range strings.Fields(namelist) {
		if isAbbrev(str, tok) {
			return true
		}
	}
	return false
}

// charKeywords returns the keyword namelist a target should be matched
// against, mirroring C's i->player.name field:
//   - mobs:  the prototype's Keywords (space-separated), falling back to the
//     name parsed out of the ShortDesc if Keywords is empty.
//   - players: the player's Name.
func charKeywords(c combat.Combatant) string {
	if c == nil {
		return ""
	}
	if mob, ok := c.(*MobInstance); ok && mob != nil {
		if mob.Prototype != nil && mob.Prototype.Keywords != "" {
			return mob.Prototype.Keywords
		}
		// Fall back to the short description for mobs without a keyword list,
		// matching the historical Go behavior so old world data still resolves.
		return mob.GetShortDesc()
	}
	return c.GetName()
}

// ResolveCharInRoom is the canonical in-room character target resolver,
// faithful to C's get_char_room_vis (src/handler.c:1276-1300). It is the
// single function all in-room character-target commands should route through
// (DP-907): consider, kick, backstab, bash, trip, rescue, steal-victim, kill,
// hit, look-at-char, recite-target.
//
// Semantics (matching C):
//   - "self" / "me" resolves to ch itself.
//   - An optional leading ordinal "N." selects the Nth visible match
//     (GetNumber strips it; "2.guard" → 2nd visible guard). "0.name" is
//     player-only (like C's get_player_vis special case).
//   - Matching is per-keyword abbreviation against the keyword namelist
//     (isnameWithAbbrevs), NOT a substring match against the ShortDesc —
//     this is what made consider vs. kick disagree.
//   - canSee (CAN_SEE) gates every candidate; the ordinal counts only visible
//     matches.
//   - Mobs and players are scanned together in room order. To preserve the
//     existing player-then-mob precedence observed by callers, players are
//     checked first unless the ordinal forces a single pool.
//
// The returned CharTarget carries both the combat.Combatant and the concrete
// *Player / *MobInstance so callers needing typed access (consider, look,
// hit) don't have to re-resolve or type-switch blindly.
func (w *World) ResolveCharInRoom(ch *Player, name string) (CharTarget, bool) {
	if ch == nil {
		return CharTarget{}, false
	}
	original := name
	n := GetNumber(&name) // strips "N." prefix; returns 1 if none, 0 if non-numeric
	if n == 0 {
		return CharTarget{}, false
	}

	// "self" / "me" → ch (C: handler.c:1284-1285).
	if lc := strings.ToLower(name); lc == "self" || lc == "me" {
		return CharTarget{Combatant: ch, Player: ch}, true
	}

	playerOnly := strings.HasPrefix(original, "0.") // C: 0.<name> → get_player_vis

	// Build the visible candidate list in a STABLE order. Players first
	// (existing Go convention), then mobs — unless the caller only wants a
	// player via the "0." prefix. Mobs come from a map (w.activeMobs), whose
	// iteration order is randomized, so we sort by a stable key (mob VNum,
	// then instance ID) to make ordinals like "2.guard" reproducible across
	// calls — matching C's stable people-list ordering. (DP-907)
	chRoom := ch.GetRoom()
	var players []*Player
	for _, p := range w.GetPlayersInRoom(chRoom) {
		if p == ch {
			continue
		}
		players = append(players, p)
	}
	sort.Slice(players, func(i, j int) bool { return players[i].Name < players[j].Name })

	var mobs []*MobInstance
	if !playerOnly {
		mobs = w.GetMobsInRoom(chRoom)
	}
	sort.Slice(mobs, func(i, j int) bool {
		if mobs[i].GetVNum() != mobs[j].GetVNum() {
			return mobs[i].GetVNum() < mobs[j].GetVNum()
		}
		return mobs[i].ID < mobs[j].ID
	})

	var candidates []combat.Combatant
	for _, p := range players {
		candidates = append(candidates, p)
	}
	for _, m := range mobs {
		candidates = append(candidates, m)
	}

	// C iterates people in the room and counts only visible matches; the Nth
	// visible match wins. We replicate that over our ordered candidate list.
	matched := 0
	for _, c := range candidates {
		if !canSee(ch, asActor(c)) {
			continue
		}
		if !isnameWithAbbrevs(name, charKeywords(c)) {
			continue
		}
		matched++
		if matched == n {
			return newCharTarget(c), true
		}
	}
	return CharTarget{}, false
}

// CharTarget is the result of ResolveCharInRoom. At most one of Player / Mob is
// non-nil (unless the target is the resolver's own `ch`, in which case Player
// is set). Combatant is always set when Found is true.
type CharTarget struct {
	Combatant combat.Combatant
	Player    *Player
	Mob       *MobInstance
}

// newCharTarget classifies a resolved Combatant into a CharTarget.
func newCharTarget(c combat.Combatant) CharTarget {
	t := CharTarget{Combatant: c}
	if p, ok := c.(*Player); ok {
		t.Player = p
	}
	if m, ok := c.(*MobInstance); ok {
		t.Mob = m
	}
	return t
}

// asActor adapts a combat.Combatant to the Actor interface used by canSee.
// Both *Player and *MobInstance satisfy Actor; this keeps the assertion local.
func asActor(c combat.Combatant) Actor {
	if a, ok := c.(Actor); ok {
		return a
	}
	return nil
}
