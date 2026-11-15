package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

const (
	skillNumBearhug    = 142
	skillNumFleshAlter = 168
	skillNumFirstAid   = 179
)

// CharacterFirstAid ports the command body of do_first_aid for a mobile.
// Mobiles share C's dummy_mob skill record; a fresh world reads zero.
func CharacterFirstAid(ctx CharacterCommandContext, args []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	if ctx.World.DummyMobSkill(skillNumFirstAid) == 0 {
		mobileSend(ctx, mob, "You have no idea how!\r\n")
		return CharacterCommandHandled
	}
	name, _ := OneArgument(strings.Join(args, " "))
	if name == "" {
		mobileSend(ctx, mob, "Aid who?\r\n")
		return CharacterCommandHandled
	}
	target, found := ctx.World.ResolveCharInRoom(mob, name)
	if !found {
		mobileSend(ctx, mob, "They don't seem to be here.\r\n")
		return CharacterCommandHandled
	}
	victim := target.Combatant
	if victim == mob {
		mobileSend(ctx, mob, "You wish you could.\r\n")
		return CharacterCommandHandled
	}
	if victim.GetHP() >= 1 {
		mobileSend(ctx, mob, "They don't really need first aid.\r\n")
		return CharacterCommandHandled
	}
	percent := dprng.Number(1, 101+victim.GetLevel())
	if percent < ctx.World.DummyMobSkill(skillNumFirstAid) || mob.GetLevel() > LVL_IMMORT {
		if player, ok := victim.(*Player); ok {
			player.SetHP(1)
			updatePosFromHP(player, 1)
		} else if targetMob, ok := victim.(*MobInstance); ok {
			targetMob.SetHealth(1)
			updateMobPosFromHP(targetMob, 1)
		}
		victActor := asActor(victim)
		ActWithPrivateSink(ctx.World, false, mob, victActor, nil, nil,
			"You apply some makeshift bandages to $N's wounds.", "", ToChar, mob, ctx.PrivateOutput)
		Act(ctx.World, false, mob, victActor, nil, nil,
			"$n applies some bandaging to $N's wounds.", "", ToNotVict)
		Act(ctx.World, true, mob, victActor, nil, nil,
			"$n applies some bandaging to your wounds.", "", ToVict)
		if waiter, ok := victim.(interface{ SetWaitStatePulses(int) }); ok {
			waiter.SetWaitStatePulses(engine.PULSE_VIOLENCE)
		}
	} else {
		mobileSend(ctx, mob, "You fumble and ruin the bandages.\r\n")
		Act(ctx.World, true, mob, nil, nil, nil,
			"$n fumbles with some bandaging and drops it all over the place!", "", ToRoom)
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE + 3)
	return CharacterCommandHandled
}

// CharacterFleshAlter ports do_flesh_alter's mobile body. The live shared
// skill value is read before the command's first random draw.
func CharacterFleshAlter(ctx CharacterCommandContext, _ []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	prob := ctx.World.DummyMobSkill(skillNumFleshAlter)
	if prob == 0 {
		mobileSend(ctx, mob, "You know nothing of altering your flesh!\n\r")
		return CharacterCommandHandled
	}
	rollMax := 101
	if mob.GetFightingBody() != nil {
		rollMax += 10
	}
	if dprng.Number(0, rollMax) > prob {
		mobileSend(ctx, mob, "You lose your concentration!\r\n")
		mob.SetWaitState(2)
		return CharacterCommandHandled
	}
	weapon := fleshAlterWeapon(mob.GetLevel())
	if mob.IsAffected(affFleshAlter) {
		mob.RemoveAffected(affFleshAlter)
		mob.adjustHitroll(-((mob.GetLevel() / 3) + 1))
		mob.adjustDamroll(-((mob.GetLevel() / 2) + 1))
		mobileSend(ctx, mob, "You shift your molecules back to normal.\r\n")
		mobileSend(ctx, mob, fmt.Sprintf("Your hand reverts from a %s.\r\n", weapon))
		Act(ctx.World, true, mob, nil, nil, nil,
			fmt.Sprintf("$n's hand reverts from a %s!", weapon), "", ToRoom)
		return CharacterCommandHandled
	}
	mob.SetAffected(affFleshAlter)
	mob.adjustHitroll((mob.GetLevel() / 3) + 1)
	mob.adjustDamroll((mob.GetLevel() / 2) + 1)
	if wielded := mob.Equipped(mobWearWield); wielded != nil {
		if mob.UnequipItem(mobWearWield) != nil {
			ActWithPrivateSink(ctx.World, false, mob, nil, wielded, nil,
				"You stop using $p.", "", ToChar, mob, ctx.PrivateOutput)
			Act(ctx.World, true, mob, nil, wielded, nil,
				"$n stops using $p.", "", ToRoom)
		}
	}
	mobileSend(ctx, mob, fmt.Sprintf("Your hand turns into a %s!\r\n", weapon))
	Act(ctx.World, true, mob, nil, nil, nil,
		fmt.Sprintf("$n's hand turns into a %s!", weapon), "", ToRoom)
	return CharacterCommandHandled
}

// CharacterBerserk ports do_berserk for the mobile body. improve_skill is a
// no-op for NPCs in C, so it consumes no additional random draws here.
func CharacterBerserk(ctx CharacterCommandContext, _ []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	prob := ctx.World.DummyMobSkill(skillNumBerserk)
	if prob == 0 {
		mobileSend(ctx, mob, "You're about as berserk as you're gonna get.\r\n")
		return CharacterCommandHandled
	}
	percent := dprng.Number(1, 101)
	if mob.IsAffected(affBerserk) {
		mobileSend(ctx, mob, "You're unable to summon your battle rage right now.\r\n")
		return CharacterCommandHandled
	}
	if mob.GetLevel() > LVL_IMMORT {
		percent = 0
	}
	failed := percent > prob
	message, hitDamMod, acMod := "Your vision turns sanguine as you summon up your battle rage!\r\n", 2, 25
	if failed {
		message, hitDamMod, acMod = "You fail to summon up your battle rage.\r\n", 0, 0
	}
	mob.AddAffect(engine.NewAffectDirect(skillNumBerserk, ApplyHitroll, 1, hitDamMod, engine.AFFBerserk, "berserk"))
	mob.AddAffect(engine.NewAffectDirect(skillNumBerserk, ApplyDamroll, 1, hitDamMod, engine.AFFBerserk, "berserk"))
	mob.AddAffect(engine.NewAffectDirect(skillNumBerserk, ApplyAC, 1, acMod, engine.AFFBerserk, "berserk"))
	mob.SetWaitState(2)
	mobileSend(ctx, mob, message)
	return CharacterCommandHandled
}

// CharacterBearhug ports do_bearhug for a switched mobile, including the C
// byte-width percent behavior and the body-owned WAIT_STATE.
func CharacterBearhug(ctx CharacterCommandContext, args []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	prob := ctx.World.DummyMobSkill(skillNumBearhug)
	if prob == 0 {
		mobileSend(ctx, mob, "You'd better leave all the martial arts to fighters.\n\r")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	name, _ := OneArgument(strings.Join(args, " "))
	var victim combat.Combatant
	if target, found := ctx.World.ResolveCharInRoom(mob, name); found {
		victim = target.Combatant
	} else {
		victim = mob.GetFightingBody()
	}
	if victim == nil {
		mobileSend(ctx, mob, "Bear hug who?\r\n")
		return CharacterCommandHandled
	}
	if player, ok := victim.(*Player); ok && player.GetLevel() >= LVL_IMMORT {
		mobileSend(ctx, mob, "The gods reject your impunity.\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "Aren't we funny today...\r\n")
		return CharacterCommandHandled
	}
	if mob.Equipped(mobWearWield) != nil {
		mobileSend(ctx, mob, "You need to be bare handed to get a good grip.\r\n")
		return CharacterCommandHandled
	}
	percent := dprng.Number(1, 150)
	if percent >= 1<<7 {
		percent -= 1 << 8
	}
	if victim.GetPosition() <= combat.PosSleeping || mob.GetLevel() > LVL_IMMORT {
		percent = 101
	}
	if targetMob, ok := victim.(*MobInstance); ok && targetMob.HasMobFlag(MobFlagNobash) {
		percent = 101
	}
	dam := 0
	if percent <= prob {
		dam = mob.GetLevel() + mob.GetLevel()/2
	}
	ctx.World.mobSkillDamage(mob, victim, dam, SkillBearhugNum)
	mob.SetWaitState(2)
	return CharacterCommandHandled
}

// CharacterBite ports the NPC branch of do_bite. That branch is selected
// explicitly by IS_NPC(ch), after descriptor, room, argument and self gates.
func CharacterBite(ctx CharacterCommandContext, args []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	if ctx.PrivateOutput == nil || ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		return CharacterCommandHandled
	}
	name, _ := OneArgument(strings.Join(args, " "))
	if name == "" && mob.GetFightingBody() != nil {
		return CharacterCommandHandled
	}
	target, found := ctx.World.ResolveCharInRoom(mob, name)
	if !found {
		mobileSend(ctx, mob, "Bite who?!\r\n")
		return CharacterCommandHandled
	}
	if target.Combatant == mob {
		mobileSend(ctx, mob, "You bite your tongue and say nothing.\r\n")
		return CharacterCommandHandled
	}
	victim := asActor(target.Combatant)
	ActWithPrivateSink(ctx.World, true, mob, victim, nil, nil,
		"You give $N a love bite.", "", ToChar, mob, ctx.PrivateOutput)
	Act(ctx.World, true, mob, victim, nil, nil,
		"$n tries to give you a little love bite.", "", ToVict)
	Act(ctx.World, true, mob, victim, nil, nil,
		"$n gives $N a love bite.", "", ToNotVict)
	return CharacterCommandHandled
}

// CharacterBehead ports do_behead's room-object path for a switched mobile.
func CharacterBehead(ctx CharacterCommandContext, args []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	name, _ := OneArgument(strings.Join(args, " "))
	target, found := ctx.World.ResolveCharInRoom(mob, name)
	if mob.GetPosition() == combat.PosFighting {
		mobileSend(ctx, mob, "You're a little busy for that!\r\n")
		return CharacterCommandHandled
	}
	if name == "" {
		mobileSend(ctx, mob, "Behead who?\r\n")
		return CharacterCommandHandled
	}
	if found && target.Combatant == mob {
		mobileSend(ctx, mob, "This MUD doesn't support self-mutilation!\r\n")
		return CharacterCommandHandled
	}
	if found {
		mobileSend(ctx, mob, "You kill it first and THEN you behead it!\r\n")
		return CharacterCommandHandled
	}
	room := ctx.World.GetRoomInWorld(mob.GetRoomVNum())
	if room == nil {
		mobileSend(ctx, mob, "You are in a void.\r\n")
		return CharacterCommandHandled
	}
	obj, found := ctx.World.ResolveObjectInRoom(mob, name)
	if !found {
		mobileSend(ctx, mob, fmt.Sprintf("You can't seem to find a %s to behead!\r\n", name))
		return CharacterCommandHandled
	}
	if strings.Contains(obj.GetKeywords(), "headless") {
		mobileSend(ctx, mob, "You can't behead something without a head!\r\n")
		return CharacterCommandHandled
	}
	if obj.GetTypeFlag() != ITEM_CONTAINER || obj.GetValue(3) == 0 {
		mobileSend(ctx, mob, "You can't behead that!\r\n")
		return CharacterCommandHandled
	}
	wielded := mob.Equipped(mobWearWield)
	slashWeapon := wielded != nil && wielded.GetValue(3) == 3
	performBeheadWithPreOutput(ctx.World, mob, obj, wielded != nil, slashWeapon,
		func(headObj *ObjectInstance) bool { return canTakeObjMobile(ctx, mob, headObj) },
		func(headObj *ObjectInstance) error { return ctx.World.MoveObjectToMobInventoryFront(headObj, mob) },
		func() {
			if slashWeapon {
				Act(ctx.World, true, mob, nil, obj, nil, "$n beheads $p!", "", ToRoom)
				ActWithPrivateSink(ctx.World, true, mob, nil, obj, nil,
					"You behead $p!", "", ToChar, mob, ctx.PrivateOutput)
				return
			}
			Act(ctx.World, true, mob, nil, obj, nil, "$n rips the head off $p!", "", ToRoom)
			ActWithPrivateSink(ctx.World, true, mob, nil, obj, nil,
				"You rip the head off $p with your bare hands!", "", ToChar, mob, ctx.PrivateOutput)
		})
	return CharacterCommandHandled
}
