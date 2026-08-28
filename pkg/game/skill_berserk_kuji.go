// Ported from src/new_cmds.c: do_berserk() (lines 2134-2187) and
// check_kk_success() / do_kuji_kiri() (lines 1500-1739).
//
// All parts of this code not covered by the copyright by the Trustees of
// the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
// Dark Pawns Coding Team.
//
// See LICENSE for license information.

package game

import (
	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// Skill name constants — berserk and the nine kuji-kiri seals.
const (
	SkillBerserk = "berserk"

	SkillKkRin   = "kk_rin"
	SkillKkKyo   = "kk_kyo"
	SkillKkToh   = "kk_toh"
	SkillKkKai   = "kk_kai"
	SkillKkJin   = "kk_jin"
	SkillKkRetsu = "kk_retsu"
	SkillKkZai   = "kk_zai"
	SkillKkZhen  = "kk_zhen"
	SkillKkSha   = "kk_sha"
)

// Skill numbers from spells.h — used as engine.Affect.SpellID, matching
// C's af.type = subcmd / SKILL_BERSERK / SKILL_KK_*.
const (
	skillNumKkRin   = 158
	skillNumKkKyo   = 159
	skillNumKkToh   = 160
	skillNumKkKai   = 161
	skillNumKkJin   = 162
	skillNumKkRetsu = 163
	skillNumKkZai   = 164
	skillNumKkZhen  = 165
	skillNumKkSha   = 166
	skillNumBerserk = 171
)

// kkSkillNum maps a kuji-kiri seal name to its SKILL_KK_* number.
func kkSkillNum(seal string) int {
	switch seal {
	case SkillKkRin:
		return skillNumKkRin
	case SkillKkKyo:
		return skillNumKkKyo
	case SkillKkToh:
		return skillNumKkToh
	case SkillKkKai:
		return skillNumKkKai
	case SkillKkJin:
		return skillNumKkJin
	case SkillKkRetsu:
		return skillNumKkRetsu
	case SkillKkZai:
		return skillNumKkZai
	case SkillKkZhen:
		return skillNumKkZhen
	case SkillKkSha:
		return skillNumKkSha
	}
	return 0
}

// DoBerserk implements do_berserk() from new_cmds.c:2134-2187.
// Self-buff: +hitroll/+damroll/+AC(worse) for 1 tick, gated on a skill-vs-
// percent roll. Immortals (level > LVL_IMMORT) always succeed.
func DoBerserk(ch *Player) SkillResult {
	if ch.GetSkill(SkillBerserk) == 0 {
		return SkillResult{Success: false, MessageToCh: "You're about as berserk as you're gonna get."}
	}

	// C draws percent before checking whether the caster is already berserk.
	// Preserve that draw even when the affect gate rejects the command.
	// #nosec G404 — game RNG, not cryptographic
	percent := dprng.Number(1, 101) // 1-101, 101 = guaranteed failure

	if ch.IsAffected(affBerserk) {
		return SkillResult{Success: false, MessageToCh: "You're unable to summon your battle rage right now."}
	}

	if ch.GetLevel() > LVL_IMMORT {
		percent = 0 // immortals always succeed
	}
	prob := ch.GetSkill(SkillBerserk)
	failed := percent > prob

	msg := "Your vision turns sanguine as you summon up your battle rage!"
	hitDamMod, acMod := 2, 25
	if failed {
		msg = "You fail to summon up your battle rage."
		hitDamMod, acMod = 0, 0
	}

	// C's do_berserk has a dangling-else bug: improve_skill() is written at
	// the same indentation as the else branch but isn't inside it (no
	// braces), so it actually runs unconditionally on every use. Ported
	// faithfully rather than "fixed".
	improveSkill(ch, SkillBerserk)

	// C reuses a single `struct affected_type af` across all three
	// affect_to_char() calls without resetting .bitvector, so AFF_BERSERK
	// ends up set on all three (hitroll/damroll/AC), not just the first.
	ch.AddAffect(engine.NewAffectDirect(skillNumBerserk, ApplyHitroll, 1, hitDamMod, engine.AFFBerserk, "berserk"))
	ch.AddAffect(engine.NewAffectDirect(skillNumBerserk, ApplyDamroll, 1, hitDamMod, engine.AFFBerserk, "berserk"))
	ch.AddAffect(engine.NewAffectDirect(skillNumBerserk, ApplyAC, 1, acMod, engine.AFFBerserk, "berserk"))

	return SkillResult{
		Success:     !failed,
		MessageToCh: msg,
		WaitCh:      2, // PULSE_VIOLENCE * 2
	}
}

// checkKkSuccess implements check_kk_success() from new_cmds.c:1500-1541.
//
// NOTE: the original C switch has no case for SKILL_KK_TOH, so prob stays 0
// for that seal and the 1-101 roll always exceeds it — the Toh seal's
// success check always fails. Ported faithfully; do not "fix".
func checkKkSuccess(ch *Player, seal string) bool {
	// #nosec G404 — game RNG, not cryptographic
	percent := dprng.Number(1, 101) // 1-101, 101 = guaranteed failure
	var prob int
	switch seal {
	case SkillKkRin:
		prob = ch.GetSkill(SkillKkRin)
	case SkillKkKyo:
		prob = ch.GetSkill(SkillKkKyo)
	case SkillKkSha:
		prob = ch.GetSkill(SkillKkSha)
	case SkillKkKai:
		prob = ch.GetSkill(SkillKkKai)
	case SkillKkJin:
		prob = ch.GetSkill(SkillKkJin)
	case SkillKkRetsu:
		prob = ch.GetSkill(SkillKkRetsu)
	case SkillKkZai:
		prob = ch.GetSkill(SkillKkZai)
	case SkillKkZhen:
		prob = ch.GetSkill(SkillKkZhen)
	}
	return percent <= prob
}

// DoKujiKiri implements do_kuji_kiri() from new_cmds.c:1552-1739.
// seal is one of the SkillKk* constants identifying which seal was invoked.
//
// Every seal applies an AFF_KUJI_KIRI lockout (5 ticks) regardless of
// success/failure — that's what blocks back-to-back kuji-kiri use. On
// success the seal's own numeric/status effect is also applied; on failure
// it is not (matching C: af[0].modifier and af[1] are zeroed on failure).
// The secondary affect (af1 in C) is only ever observable on success — on
// failure C always zeroes both its bitvector and modifier, leaving nothing
// but inert metadata, so it's skipped here rather than created as a no-op.
func DoKujiKiri(ch *Player, seal string, world *World) SkillResult {
	if ch.GetClass() != ClassNinja && ch.GetLevel() < LVL_IMMORT {
		return SkillResult{Success: false, MessageToCh: "You know nothing of kuji-kiri!"}
	}
	if ch.IsFighting() {
		return SkillResult{Success: false, MessageToCh: "You are too busy fighting to practice kuji-kiri!"}
	}
	if ch.IsAffected(affKujiKiri) {
		return SkillResult{Success: false, MessageToCh: "You can not practice kuji-kiri again right now!"}
	}
	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}
	if ch.GetSkill(seal) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have not mastered this art yet!"}
	}

	success := checkKkSuccess(ch, seal)
	skillNum := kkSkillNum(seal)

	// af0 always carries the AFF_KUJI_KIRI lockout; some seals also route
	// their own numeric effect through it (C: af[0] defaults to
	// {bitvector: AFF_KUJI_KIRI, location: APPLY_SPELL}, then the seal's
	// case statement may override location/modifier).
	af0Location := ApplySpell
	af0Modifier := 0

	// af1 carries an optional secondary status effect, only meaningful on
	// success (see doc comment above).
	haveAf1 := false
	af1Bitvector := uint64(0)
	af1Location := ApplySpell
	af1Modifier := 0

	toVict := ""
	toRoom := "$n interlaces $s fingers and meditates deeply."
	heal := 0

	switch seal {
	case SkillKkRin:
		af0Location = ApplyAC
		af0Modifier = -(15 + ch.GetLevel()/2)
		haveAf1 = true
		af1Bitvector = engine.AFFMetalskin
		toVict = "Interlacing your fingers, you harden your mind and body."
		toRoom = "$n interlaces $s fingers, and $s skin becomes metal!"
	case SkillKkKyo:
		af0Location = ApplyHitroll
		af0Modifier = 1
		toVict = "Interlacing your fingers, you focus your battle rage."
	case SkillKkToh:
		af0Location = ApplyDamroll
		af0Modifier = 1
		haveAf1 = true
		af1Location = ApplyAC
		af1Modifier = 10
		toVict = "Interlacing your fingers, you focus your inner strength."
	case SkillKkSha:
		if ch.GetLevel() < 15 {
			heal = 15
		} else {
			heal = ch.GetLevel()
		}
		toVict = "Interlacing your fingers, you heal your wounds."
	case SkillKkKai:
		af0Location = ApplyDamroll
		af0Modifier = -1
		haveAf1 = true
		af1Location = ApplyAC
		af1Modifier = -10
		toVict = "Interlacing your fingers, your body becomes your fortress."
	case SkillKkJin:
		toVict = "Interlacing your fingers, you focus on recooperation."
	case SkillKkRetsu:
		toRoom = ""
		if success {
			spells.Cast(ch, ch, spells.SpellTeleport, ch.GetLevel(), world)
		} else {
			toVict = "Your concentration is broken!"
		}
	case SkillKkZai:
		af0Location = ApplyHitroll
		af0Modifier = 0
		haveAf1 = true
		af1Bitvector = engine.AFFInvisible
		toVict = "Interlacing your fingers, you slowly fade from view."
		toRoom = "$n interlaces $s fingers and slowly fades from view."
	case SkillKkZhen:
		toVict = "Interlacing your fingers, you focus on your endurance."
	}

	if !success {
		af0Modifier = 0
		haveAf1 = false
		heal = 0
		toVict = "You try the art of kuji-kiri, but can't concentrate!"
		// C only calls improve_skill() on FAILURE here — faithfully ported.
		improveSkill(ch, seal)
	}

	ch.AddAffect(engine.NewAffectDirect(skillNum, af0Location, 5, af0Modifier, engine.AFFKujiKiri, "kuji-kiri"))
	if haveAf1 {
		ch.AddAffect(engine.NewAffectDirect(skillNum, af1Location, 5, af1Modifier, af1Bitvector, "kuji-kiri"))
	}

	if heal > 0 {
		ch.SetHP(min(ch.GetMaxHP(), ch.GetHP()+heal))
	}

	return SkillResult{
		Success:       success,
		MessageToCh:   toVict,
		MessageToRoom: toRoom,
	}
}
