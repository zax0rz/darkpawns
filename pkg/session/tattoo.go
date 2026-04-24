// Package session provides command handlers and WebSocket-based player sessions.
package session

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// Tattoo type constants — from structs.h (tattoo.c)
const (
	TatNone   = 0
	TatDragon = 1
	TatTribal = 2
	TatSkull  = 3
	TatTiger  = 4
	TatWorm   = 5
	TatEye    = 6
	TatSwords = 7
	TatEagle  = 8
	TatHeart  = 9
	TatStar   = 10
	TatShip   = 11
	TatSpider = 12
	TatJyhad  = 13
	TatMom    = 14
	TatAngel  = 15
	TatFox    = 16
	TatOwl    = 17
)

const (
	// DefaultWandLvl is the default caster level for wand-type magic
	// Source: spells.h #define DEFAULT_WAND_LVL 12
	DefaultWandLvl = 12

	// MaxTatAffects is the maximum number of affect entries a tattoo can produce
	// Source: tattoo.c #define MAX_TAT_AFFECTS 3
	MaxTatAffects = 3

	// TatCooldownHours is the cooldown applied after using a tattoo
	// Source: tattoo.c TAT_TIMER(ch)=24
	TatCooldownHours = 24
)

// use_tattoo activates the player's tattoo power.
// Returns true if a tattoo was successfully used.
// Source: src/tattoo.c use_tattoo()
func useTattoo(ch *Session) bool {
	if ch.player == nil {
		return false
	}

	if ch.player.TatTimer > 0 {
		ch.Send(fmt.Sprintf("You can't use your tattoo's magick for %d more hour%s.\r\n",
			ch.player.TatTimer, map[bool]string{true: "s", false: ""}[ch.player.TatTimer > 1]))
		return false
	}

	switch ch.player.Tattoo {
	case TatNone:
		ch.Send("You don't have a tattoo.\r\n")

	case TatSkull:
		// TODO: mob spawn + follow system
		// Original C code:
		//   struct char_data *skull = read_mobile(9, VIRTUAL);
		//   char_to_room(skull, ch->in_room);
		//   add_follower_quiet(skull, ch);
		//   IS_CARRYING_W(skull) = 0;
		//   IS_CARRYING_N(skull) = 0;
		//   af.type = SPELL_CHARM;
		//   af.duration = 20;
		//   af.modifier = 0;
		//   af.location = 0;
		//   af.bitvector = AFF_CHARM;
		//   affect_to_char(skull, &af);
		//   act("...glows brightly...$N appears!", ...)
		//
		// Stubbed: sends the act messages but does not spawn/follow yet.
		broadcastToRoom(ch, "$n's tattoo glows brightly for a second, and a skull appears!")
		ch.Send("Your tattoo glows brightly for a second, and a skull appears!\r\n")

	case TatEye:
		// call_magic(ch, ch, NULL, SPELL_GREATPERCEPT, DEFAULT_WAND_LVL, CAST_WAND)
		spells.Cast(ch.player, ch.player, spells.SpellGreatPercept, DefaultWandLvl, nil)

	case TatShip:
		// call_magic(ch, ch, NULL, SPELL_CHANGE_DENSITY, DEFAULT_WAND_LVL, CAST_WAND)
		spells.Cast(ch.player, ch.player, spells.SpellChangeDensity, DefaultWandLvl, nil)

	case TatAngel:
		// call_magic(ch, ch, NULL, SPELL_BLESS, DEFAULT_WAND_LVL, CAST_WAND)
		spells.Cast(ch.player, ch.player, spells.SpellBless, DefaultWandLvl, nil)

	default:
		ch.Send("Your tattoo can't be 'use'd.\r\n")
		return false
	}

	ch.player.TatTimer = TatCooldownHours
	return false
}

// tattooAf applies or removes the stat modifiers for a player's tattoo.
// add=true applies the affects; add=false removes them.
// Source: src/tattoo.c tattoo_af()
//
// NOTE: This uses direct stat modification rather than the full engine.Affect
// system. The original C code called affect_modify() which directly
// incremented/decremented stats. A future improvement would wire this
// through proper duration-based affects visible in 'score'.
func tattooAf(ch *Session, add bool) {
	if ch.player == nil || ch.player.Tattoo == TatNone {
		return
	}

	// Build the affect table matching the C code's MAX_TAT_AFFECTS array.
	// Each entry: (location, modifier) pair.
	type affEntry struct {
		loc  int
		mod  int
	}

	// Initialize all entries to skip state (APPLY_NONE equivalent)
	var afs [MaxTatAffects]affEntry
	for i := range afs {
		afs[i].loc = -1
	}

	// Populate affect entries based on tattoo type
	// Location values match APPLY_* from structs.h / affect.c:
	//   APPLY_DAMROLL=19, APPLY_HITROLL=18, APPLY_STR=0, APPLY_DEX=2,
	//   APPLY_INT=1, APPLY_WIS=3, APPLY_MOVE=13, APPLY_HIT=11, APPLY_MANA=12
	switch ch.player.Tattoo {
	case TatDragon:
		afs[0] = affEntry{loc: 19, mod: 2} // APPLY_DAMROLL
		afs[1] = affEntry{loc: 0, mod: 2}  // APPLY_STR

	case TatTiger:
		afs[0] = affEntry{loc: 2, mod: 1}  // APPLY_DEX
		afs[1] = affEntry{loc: 13, mod: 10} // APPLY_MOVE

	case TatTribal:
		afs[0] = affEntry{loc: 2, mod: 1} // APPLY_DEX

	case TatWorm:
		afs[0] = affEntry{loc: 19, mod: 2} // APPLY_DAMROLL

	case TatSwords:
		afs[0] = affEntry{loc: 19, mod: 1} // APPLY_DAMROLL
		afs[1] = affEntry{loc: 18, mod: 1} // APPLY_HITROLL

	case TatEagle:
		afs[0] = affEntry{loc: 13, mod: 20} // APPLY_MOVE

	case TatHeart:
		afs[0] = affEntry{loc: 11, mod: 20} // APPLY_HIT

	case TatStar:
		afs[0] = affEntry{loc: 12, mod: 20} // APPLY_MANA

	case TatSpider:
		afs[0] = affEntry{loc: 2, mod: 3} // APPLY_DEX

	case TatJyhad:
		afs[0] = affEntry{loc: 19, mod: 1} // APPLY_DAMROLL

	case TatMom:
		afs[0] = affEntry{loc: 3, mod: 3} // APPLY_WIS

	case TatFox:
		afs[0] = affEntry{loc: 1, mod: 1} // APPLY_INT

	case TatOwl:
		afs[0] = affEntry{loc: 3, mod: 1} // APPLY_WIS
	}

	// Apply or remove each affect
	// Original C: for (i=0; i<MAX_TAT_AFFECTS; i++)
	//   if (af[i].location != APPLY_NONE) affect_modify(ch, ...)
	for _, af := range afs {
		if af.loc == -1 {
			continue
		}
		applyModifier(ch.player, af.loc, af.mod, add)
	}
}

// applyModifier applies a single stat modifier to the player.
// This is the direct equivalent of affect_modify() from the C code, but scoped
// only to the tattoo-only stat locations used by tattoo.c.
func applyModifier(p *game.Player, location int, modifier int, add bool) {
	if !add {
		modifier = -modifier
	}

	switch location {
	case 0: // APPLY_STR
		p.Stats.Str += modifier
	case 1: // APPLY_INT
		p.Stats.Int += modifier
	case 2: // APPLY_DEX
		p.Stats.Dex += modifier
	case 3: // APPLY_WIS
		p.Stats.Wis += modifier
	case 11: // APPLY_HIT
		p.MaxHealth += modifier
		if p.Health > 0 {
			p.Health += modifier
		}
	case 12: // APPLY_MANA
		p.MaxMana += modifier
		if p.Mana > 0 {
			p.Mana += modifier
		}
	case 13: // APPLY_MOVE
		p.MaxMove += modifier
		if p.Move > 0 {
			p.Move += modifier
		}
	case 18: // APPLY_HITROLL
		p.Hitroll += modifier
	case 19: // APPLY_DAMROLL
		p.Damroll += modifier
	}
}

/*
IMPROVEMENTS

1. Tattoo affects should integrate with the full affect system (duration-based, visible in score)
   - tattooAf currently uses direct stat modification rather than engine.Affect.
   - The Player struct needs to implement the full engine.Affectable interface
     (GetStrength/SetStrength, GetDexterity/SetDexterity, etc.) before the
     AffectManager can be used here.

2. Skull tattoo needs mob spawn + follower system wired up
   - useTattoo case TatSkull currently only sends act messages as placeholders.
   - Requires: read_mobile(9), char_to_room, add_follower_quiet, and
     applying AFF_CHARM via affect_to_char on the spawned mob.

3. TatTimer should persist across saves (add to PlayerDB serialization)
   - Currently Tattoo and TatTimer are in-memory only on the Player struct.
   - The save/load system (pkg/game/save.go or similar) needs to serialize
     both fields so the cooldown survives server restarts.
*/
