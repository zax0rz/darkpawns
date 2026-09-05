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

	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// Internal helpers (ported from C macros)
// ---------------------------------------------------------------------------

// IS_MOUNTED — from act.offensive.c: checks if a player is mounted.
func isMounted(ch *Player) bool {
	return ch.IsAffected(affMount)
}

// improveSkill ports src/act.other.c:1704 improve_skill(). Use-based skill gain
// on a successful skill use. Draw-parity is law: number(1,200) is drawn on EVERY
// PC call, BEFORE the percent bounds check — a skill already at >=97 still burns
// exactly one draw. number(1,3) is drawn only when the stat gate passes AND
// percent is in (0,97). The "improves" line fires only on a +3 roll. spells[skill]
// (act.other.c:1721) is the DP catalog name, which is exactly the Skill* constant.
func improveSkill(ch *Player, skill string) {
	if message := improveSkillMessage(ch, skill); message != "" {
		ch.SendMessage(message)
	}
}

// ImproveSkill is the exported form of improveSkill for out-of-package callers
// — pkg/command's sendSkillResult runs SkillResult.DeferredImprove entries
// through it AFTER the skill_message/damage step, matching C's order (R3b,
// DP-1212). Same draw/message contract as improveSkill.
func ImproveSkill(ch *Player, skill string) {
	improveSkill(ch, skill)
}

// improveSkillMessage performs improveSkill's exact draw/mutation contract and
// returns its optional player-facing line. Most callers use improveSkill,
// which writes the line immediately. Result-based command implementations use
// this form when C sends another line before calling improve_skill.
func improveSkillMessage(ch *Player, skill string) string {
	if ch.IsNPC() {
		return ""
	}
	percent := ch.GetSkill(skill)
	// #nosec G404 — game RNG, not cryptographic
	if dprng.Number(1, 200) > ch.GetWis()+ch.GetInt() {
		return ""
	}
	if percent >= 97 || percent <= 0 {
		return ""
	}
	// #nosec G404 — game RNG, not cryptographic
	newpercent := dprng.Number(1, 3)
	percent += newpercent
	ch.SetSkill(skill, percent)
	if newpercent == 3 {
		displayName := skill
		if skill == SkillFleshAlter {
			displayName = "flesh alter"
		}
		return fmt.Sprintf("Your skill in %s improves.\r\n", displayName)
	}
	return ""
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
