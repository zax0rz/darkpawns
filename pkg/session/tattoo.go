// Package session provides command handlers and WebSocket-based player sessions.
//
//nolint:unused // Game logic port — not yet wired to command registry.
//lint:file-ignore U1000 Game logic port — not yet wired to command registry.
package session

import (
	"github.com/zax0rz/darkpawns/pkg/game"
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

// tattooAf shares C's effective-only tattoo modifiers with the game path.
func tattooAf(ch *Session, add bool) {
	if ch != nil {
		game.TattooAf(ch.player, add)
	}
}

/*
IMPROVEMENTS

1. Tattoo affects should integrate with the full affect system (duration-based, visible in score)
   - tattooAf currently uses direct stat modification rather than engine.Affect.
   - Use engine.NewAffect + Player.AddAffect (see pkg/game/player_affects.go),
     the same path spell-cast affects already use, instead of mutating stats
     directly here.

2. Skull tattoo needs mob spawn + follower system wired up
   - useTattoo case TatSkull currently only sends act messages as placeholders.
   - Requires: read_mobile(9), char_to_room, add_follower_quiet, and
     applying AFF_CHARM via affect_to_char on the spawned mob.

3. TatTimer should persist across saves (add to PlayerDB serialization)
   - Currently Tattoo and TatTimer are in-memory only on the Player struct.
   - The save/load system (pkg/game/save.go or similar) needs to serialize
     both fields so the cooldown survives server restarts.
*/
