package game

// combat_helpers.go — helper functions for combat
//
// All rights reserved. See license.doc for complete information.
//
// Copyright (C) 1993, 94 by the Trustees of the Johns Hopkins University
// CircleMUD is based on DikuMUD, Copyright (C) 1990, 1991.
//
// All parts of this code not covered by the copyright by the Trustees of
// the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
// Dark Pawns Coding Team.
//
// This includes all original code done for Dark Pawns MUD by other authors.
// All code is the intellectual property of the author, and is used here
// by permission.
//
// No original code may be duplicated, reused, or executed without the
// written permission of the author. All rights reserved.

import (
	"fmt"
	"math/rand/v2"
)

// Internal helpers (ported from C macros)
// ---------------------------------------------------------------------------

// IS_MOUNTED — from act.offensive.c: checks if a player is mounted.
func isMounted(ch *Player) bool {
	return ch.IsAffected(affMount)
}

// improveSkill implements CircleMUD-style skill improvement.
// Random chance based on current skill level and player stats.
func improveSkill(ch *Player, skill string) {
	cur := ch.GetSkill(skill)
	if cur <= 0 || cur >= 100 {
		return
	}
	// Higher skill = harder to improve (like CircleMUD)
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	if rand.IntN(100)+1 > cur {
		// Stat-based check: INT/WIS average gives improvement chance
		chance := (ch.GetInt() + ch.GetWis()) / 4
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		if rand.IntN(100) < chance {
			ch.SetSkill(skill, cur+1)
			ch.SendMessage(fmt.Sprintf("You feel a bit more competent in %s.\r\n", skill))
		}
	}
}

// ---------------------------------------------------------------------------
// rawKill — handles immediate death (raw_kill() from fight.c)
// ---------------------------------------------------------------------------

// rawKill immediately kills the target with the given attack type.
func (w *World) rawKill(victim *Player, attackType int) {
	// Handle death via existing infrastructure
	// Corpse creation is handled by HandleDeath -> handlePlayerDeath

	// Trigger death processing
	w.HandleDeath(victim, nil, attackType)
}
