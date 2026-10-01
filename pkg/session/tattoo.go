// Package session provides command handlers and WebSocket-based player sessions.
//
//nolint:unused // Game logic port — not yet wired to command registry.
//lint:file-ignore U1000 Game logic port — not yet wired to command registry.
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
		// Spawn skull mob (vnum 9), charm it, add as follower
		// Source: src/tattoo.c
		if mob, err := ch.manager.world.SpawnMob(9, ch.player.RoomVNum); err == nil {
			mob.SetFollowing(ch.player.Name)
			mob.SetLevel(1)
			mob.SetAffected(3) // AFF_CHARM bit
			broadcastToRoom(ch, "$n's tattoo glows brightly for a second, and a skull appears!")
			broadcastToRoom(ch, "$n's tattoo glows brightly for a second, and a skull appears!")
			ch.Send("Your tattoo glows brightly for a second, and a skull appears!\r\n")
		} else {
			ch.Send("Your tattoo flickers but nothing happens.\r\n")
		}

	case TatEye:
		// call_magic(ch, ch, NULL, SPELL_GREATPERCEPT, DEFAULT_WAND_LVL, CAST_WAND)
		spells.Cast(ch.player, ch.player, spells.SpellGreatPercept, DefaultWandLvl, ch.manager.world)

	case TatShip:
		// call_magic(ch, ch, NULL, SPELL_CHANGE_DENSITY, DEFAULT_WAND_LVL, CAST_WAND)
		spells.Cast(ch.player, ch.player, spells.SpellChangeDensity, DefaultWandLvl, ch.manager.world)

	case TatAngel:
		// call_magic(ch, ch, NULL, SPELL_BLESS, DEFAULT_WAND_LVL, CAST_WAND)
		spells.Cast(ch.player, ch.player, spells.SpellBless, DefaultWandLvl, ch.manager.world)

	default:
		ch.Send("Your tattoo can't be 'use'd.\r\n")
		return false
	}

	ch.player.TatTimer = TatCooldownHours
	return false
}

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
