package game

import (
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// CharacterKujiKiri ports do_kuji_kiri for a mobile command body. Mobile
// GET_SKILL reads the shared dummy_mob array, whose default is zero but which
// restore and Lua set_skill can mutate. (src/db.c:104,1096,1281;
// src/utils.h:289,343; src/new_cmds.c:1501-1540,1552-1739;
// src/act.wizard.c:1599-1601; src/scripts.c:1365-1380.)
func CharacterKujiKiri(ctx CharacterCommandContext, seal string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok {
		return CharacterCommandUnsupported
	}

	if mob.GetLevel() < LVL_IMMORT {
		mobileSend(ctx, mob, "You know nothing of kuji-kiri!\r\n")
		return CharacterCommandHandled
	}
	if mob.IsFighting() {
		mobileSend(ctx, mob, "You are too busy fighting to practice kuji-kiri!\r\n")
		return CharacterCommandHandled
	}
	if mob.IsAffected(affKujiKiri) {
		mobileSend(ctx, mob, "You can not practice kuji-kiri again right now!\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}

	// C check_kk_success() draws before do_kuji_kiri checks GET_SKILL.
	// #nosec G404 -- C's game RNG draw; required for stream parity.
	percent := dprng.Number(1, 101)
	skillNum := kkSkillNum(seal)
	skill := ctx.World.DummyMobSkill(skillNum)
	if skill == 0 {
		mobileSend(ctx, mob, "You have not mastered this art yet!\r\n")
		return CharacterCommandHandled
	}

	success := percent <= skill
	af0Location, af0Modifier := ApplySpell, 0
	af1Bitvector, af1Location, af1Modifier := engine.AFFKujiKiri, ApplySpell, 0
	toVict := ""
	toRoom := "$n interlaces $s fingers and meditates deeply."
	heal := 0

	switch seal {
	case SkillKkRin:
		af0Location, af0Modifier = ApplyAC, -(15 + mob.GetLevel()/2)
		af1Bitvector = engine.AFFMetalskin
		toVict = "Interlacing your fingers, you harden your mind and body."
		toRoom = "$n interlaces $s fingers, and $s skin becomes metal!"
	case SkillKkKyo:
		af0Location, af0Modifier = ApplyHitroll, 1
		toVict = "Interlacing your fingers, you focus your battle rage."
	case SkillKkToh:
		af0Location, af0Modifier = ApplyDamroll, 1
		af1Location, af1Modifier = ApplyAC, 10
		toVict = "Interlacing your fingers, you focus your inner strength."
	case SkillKkSha:
		heal = max(15, mob.GetLevel())
		toVict = "Interlacing your fingers, you heal your wounds."
	case SkillKkKai:
		af0Location, af0Modifier = ApplyDamroll, -1
		af1Location, af1Modifier = ApplyAC, -10
		toVict = "Interlacing your fingers, your body becomes your fortress."
	case SkillKkJin:
		toVict = "Interlacing your fingers, you focus on recooperation."
	case SkillKkRetsu:
		toRoom = ""
		if success {
			spells.Cast(mob, mob, spells.SpellTeleport, mob.GetLevel(), ctx.World)
		} else {
			toVict = "Your concentration is broken!"
		}
	case SkillKkZai:
		af0Location = ApplyHitroll
		af1Bitvector = engine.AFFInvisible
		toVict = "Interlacing your fingers, you slowly fade from view."
		toRoom = "$n interlaces $s fingers and slowly fades from view."
	case SkillKkZhen:
		toVict = "Interlacing your fingers, you focus on your endurance."
	default:
		return CharacterCommandUnsupported
	}

	if !success {
		af0Modifier = 0
		af1Bitvector = engine.AFFNothing
		af1Modifier = 0
		heal = 0
		toVict = "You try the art of kuji-kiri, but can't concentrate!"
		// C improve_skill() returns before drawing for an NPC.
	}

	mob.JoinAffect(engine.NewAffectDirect(skillNum, af0Location, 5, af0Modifier, engine.AFFKujiKiri, "kuji-kiri"), false, false)
	mob.JoinAffect(engine.NewAffectDirect(skillNum, af1Location, 5, af1Modifier, af1Bitvector, "kuji-kiri"), false, false)
	if heal > 0 {
		mob.SetHealth(min(mob.GetMaxHP(), mob.GetHP()+heal))
	}
	if toVict != "" {
		mobileSend(ctx, mob, toVict+"\r\n")
	}
	if toRoom != "" && ctx.World != nil {
		Act(ctx.World, true, mob, nil, nil, nil, toRoom, "", ToRoom)
	}
	return CharacterCommandHandled
}
