package game

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// CharacterUtilitySkill routes the mobile body of the utility/stealth skills
// registered via wrapSkill in pkg/session/commands.go. C runs these same
// do_* handlers against ch (the switched mobile) after the interpreter gates.
func CharacterUtilitySkill(ctx CharacterCommandContext, commandName string, args []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	rawArg := strings.TrimSpace(ctx.RawArgs)
	if rawArg == "" && len(args) > 0 {
		rawArg = strings.Join(args, " ")
	}
	switch strings.ToLower(commandName) {
	case "sneak":
		return characterSneak(ctx, mob, SkillSneak, skillNumSneak)
	case "stealth":
		return characterSneak(ctx, mob, SkillStealth, skillNumStealth)
	case "hide":
		return characterHide(ctx, mob, false)
	case "kabuki":
		return characterHide(ctx, mob, true)
	case "mindlink":
		return characterMindlink(ctx, mob, rawArg)
	case "turn":
		return characterTurn(ctx, mob)
	case "palm":
		return characterPalm(ctx, mob, rawArg)
	case "point":
		return characterPoint(ctx, mob, rawArg)
	case "compare":
		return characterCompare(ctx, mob, rawArg)
	case "sharpen":
		return characterSharpen(ctx, mob, rawArg)
	case "carve":
		return characterCarve(ctx, mob, rawArg)
	default:
		return CharacterCommandUnsupported
	}
}

func characterSneak(ctx CharacterCommandContext, mob *MobInstance, skill string, spell int) CharacterCommandResult {
	if mob.IsAffected(affMounted) {
		sendBodyMessage(ctx.World, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	sendBodyMessage(ctx.World, mob, "Okay, you'll try to move silently for a while.\r\n")
	if mob.IsAffected(affSneak) {
		mob.RemoveAffectBySpell(skillNumSneak)
		mob.RemoveAffectBySpell(skillNumStealth)
		mob.RemoveAffected(affSneak)
	}
	// #nosec G404 — game RNG, not cryptographic
	percent := dprng.Number(1, 101)
	prob := dexAppSkill(mob.GetDex()).Sneak
	if percent > prob {
		return CharacterCommandHandled
	}
	mob.AddAffect(engine.NewAffectDirect(
		spell,
		engine.ApplyNone,
		mob.GetLevel(),
		0,
		engine.AFFSneak,
		skill,
	))
	return CharacterCommandHandled
}

func characterHide(ctx CharacterCommandContext, mob *MobInstance, kabuki bool) CharacterCommandResult {
	if mob.IsAffected(affMounted) {
		sendBodyMessage(ctx.World, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if msg := hideDaytimeSectorMessage(ctx.World, mob.GetRoom()); msg != "" {
		sendBodyMessage(ctx.World, mob, msg+"\r\n")
		return CharacterCommandHandled
	}
	if kabuki {
		sendBodyMessage(ctx.World, mob, "You attempt to practice the art of kabuki.\r\n")
	} else {
		sendBodyMessage(ctx.World, mob, "You attempt to hide yourself.\r\n")
	}
	if mob.IsAffected(affHide) {
		mob.RemoveAffected(affHide)
	}
	// #nosec G404 — game RNG, not cryptographic
	percent := dprng.Number(1, 101)
	prob := dexAppSkill(mob.GetDex()).Hide
	if percent <= prob {
		mob.SetAffected(affHide)
	}
	return CharacterCommandHandled
}

func characterMindlink(ctx CharacterCommandContext, mob *MobInstance, rawArg string) CharacterCommandResult {
	targetName, _ := OneArgument(rawArg)
	if targetName == "" {
		sendBodyMessage(ctx.World, mob, "Link your mind to whose?\r\n")
		return CharacterCommandHandled
	}
	target, found := ctx.World.ResolveCharInRoom(mob, targetName)
	if !found {
		sendBodyMessage(ctx.World, mob, "They don't seem to be here.\r\n")
		return CharacterCommandHandled
	}
	if target.Combatant == mob {
		sendBodyMessage(ctx.World, mob, "You wish you could.\r\n")
		return CharacterCommandHandled
	}
	// Mobiles have GET_SKILL(ch, SKILL_MINDLINK) == 0 (src/new_cmds2.c:267).
	sendBodyMessage(ctx.World, mob, "Yeah, right.\r\n")
	return CharacterCommandHandled
}

func characterTurn(ctx CharacterCommandContext, mob *MobInstance) CharacterCommandResult {
	// Mobiles have GET_SKILL(ch, SKILL_TURN) == 0 (src/new_cmds2.c:965).
	sendBodyMessage(ctx.World, mob, "Huh?!?\r\n")
	return CharacterCommandHandled
}

func characterPalm(ctx CharacterCommandContext, mob *MobInstance, rawArg string) CharacterCommandResult {
	// C do_palm draws number(1, 101) at declaration before checking carry/arg
	// (src/new_cmds.c:2112).
	// #nosec G404 — game RNG, not cryptographic
	_ = dprng.Number(1, 101)
	arg1, _ := OneArgument(rawArg)
	if mobCarriedItemCount(mob) >= mobMaxCarryCount(mob) {
		sendBodyMessage(ctx.World, mob, "Your arms are already full!\r\n")
		return CharacterCommandHandled
	}
	if arg1 == "" {
		sendBodyMessage(ctx.World, mob, "Palm what?\r\n")
		return CharacterCommandHandled
	}
	// With prob == 0, prob > percent is always false, so C calls get_from_room(ch, arg1, FALSE).
	palmCtx := ctx
	palmCtx.RawArgs = arg1
	getForMobile(palmCtx, mob)
	return CharacterCommandHandled
}

func characterPoint(ctx CharacterCommandContext, mob *MobInstance, rawArg string) CharacterCommandResult {
	res := doPointActor(ctx.World, mob, rawArg, ctx.PrivateOutput)
	if res.MessageToCh != "" {
		if ctx.PrivateOutput != nil {
			ctx.PrivateOutput(res.MessageToCh)
		} else {
			sendBodyMessage(ctx.World, mob, res.MessageToCh)
		}
	}
	return CharacterCommandHandled
}

func characterCompare(ctx CharacterCommandContext, mob *MobInstance, rawArg string) CharacterCommandResult {
	arg1, arg2 := halfChop(rawArg)
	res := doCompareActor(mob, arg1, arg2)
	if res.MessageToCh != "" {
		if ctx.PrivateOutput != nil {
			ctx.PrivateOutput(res.MessageToCh)
		} else {
			sendBodyMessage(ctx.World, mob, res.MessageToCh)
		}
	}
	return CharacterCommandHandled
}

func characterSharpen(ctx CharacterCommandContext, mob *MobInstance, rawArg string) CharacterCommandResult {
	arg1, _ := halfChop(rawArg)
	res := doSharpenActor(ctx.World, mob, arg1)
	if res.MessageToCh != "" {
		if ctx.PrivateOutput != nil {
			ctx.PrivateOutput(res.MessageToCh)
		} else {
			sendBodyMessage(ctx.World, mob, res.MessageToCh)
		}
	}
	return CharacterCommandHandled
}

func characterCarve(ctx CharacterCommandContext, mob *MobInstance, rawArg string) CharacterCommandResult {
	res := doCarveActor(ctx.World, mob, rawArg, ctx.PrivateOutput)
	if res.MessageToCh != "" {
		msg := res.MessageToCh
		if !res.NoMessageToChTerminator {
			msg += "\r\n"
		}
		if ctx.PrivateOutput != nil {
			ctx.PrivateOutput(msg)
		} else {
			sendBodyMessage(ctx.World, mob, msg)
		}
	}
	return CharacterCommandHandled
}
