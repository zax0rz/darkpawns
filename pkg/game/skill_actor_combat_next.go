package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

func characterKick(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	if ctx.World.DummyMobSkill(skillNumKick) == 0 {
		mobileSend(ctx, mob, "You'd better leave all the martial arts to fighters.\r\n")
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, true)
	if victim == nil {
		mobileSend(ctx, mob, "Kick who?\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "Aren't we funny today...\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}

	percent := ((7 - (victim.GetAC() / 10)) << 1) + dprng.Number(1, 101)
	if percent > ctx.World.DummyMobSkill(skillNumKick) {
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillKickNum)
	} else {
		ctx.World.mobileCommandSkillDamage(mob, victim, mob.GetLevel()>>1, SkillKickNum)
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE + 2)
	return CharacterCommandHandled
}

func characterNeckbreak(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumNeckbreak)
	if prob == 0 {
		mobileSend(ctx, mob, "What's that, idiot-san?\r\n")
		return CharacterCommandHandled
	}
	if mobileWielded(mob) != nil {
		mobileSend(ctx, mob, "You can't do this and wield a weapon at the same time!\r\n")
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, false)
	if victim == nil {
		mobileSend(ctx, mob, "I don't see them here.\r\n")
		return CharacterCommandHandled
	}
	if isShopKeeperInWorld(ctx.World, victim) {
		mobileSend(ctx, mob, "Haha.. Don't think so.\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "Aren't we funny today...\r\n")
		return CharacterCommandHandled
	}
	if ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		mobileSend(ctx, mob, "You can't contemplate violence in such a place!\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if mob.GetMove() < 51 {
		mobileSend(ctx, mob, "You haven't the energy to do this!\r\n")
		return CharacterCommandHandled
	}
	mob.SetMove(mob.GetMove() - 51)

	percent := ((7 - (victim.GetAC() / 10)) << 1) + dprng.Number(1, 101)
	if percent > prob {
		ActWithPrivateSink(ctx.World, false, mob, victim, nil, nil,
			"You try to break $S neck, but $E is too strong!", "", ToChar, mob, ctx.PrivateOutput)
		Act(ctx.World, true, mob, victim, nil, nil,
			"$n tries to break your neck, but can't!", "", ToVict)
		Act(ctx.World, true, mob, victim, nil, nil,
			"$n tries to break $N's neck, but $N slips free!", "", ToNotVict)
		ctx.World.MobFirstHit(victim, mob)
	} else {
		damage := combat.RollDice(18, mob.GetLevel())
		ctx.World.mobileCommandSkillDamage(mob, victim, damage, SkillNeckbreakNum)
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE * 3)
	return CharacterCommandHandled
}

func characterTigerPunch(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumTigerPunch)
	if prob == 0 {
		mobileSend(ctx, mob, "What's that, idiot-san?\r\n")
		return CharacterCommandHandled
	}
	if mobileWielded(mob) != nil {
		mobileSend(ctx, mob, "That's pretty tough to do while wielding a weapon.\r\n")
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, true)
	if victim == nil {
		mobileSend(ctx, mob, "Hit who?\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "Aren't we funny today...\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	percent := ((7 - (victim.GetAC() / 10)) << 1) + dprng.Number(1, 101)
	if percent > prob {
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillTigerPunchNum)
	} else {
		damage := int(float64(mob.GetLevel()) * 2.5)
		ctx.World.mobileCommandSkillDamage(mob, victim, damage, SkillTigerPunchNum)
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE * 2)
	return CharacterCommandHandled
}

func characterTrip(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumTrip)
	if prob == 0 {
		mobileSend(ctx, mob, "You'd better leave the sneaky stuff to the thieves.\r\n")
		if ctx.Subcmd == 0 {
			return CharacterCommandHandled
		}
	}
	if ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		mobileSend(ctx, mob, "This room just has such a peaceful, easy feeling...\r\n")
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, true)
	if victim == nil {
		mobileSend(ctx, mob, "Trip who?\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "You trip over your shoe laces...\n\r")
		return CharacterCommandHandled
	}
	if affected, ok := victim.(interface{ IsAffected(int) bool }); ok && affected.IsAffected(affFly) {
		mobileSend(ctx, mob, "You can't trip something that's FLYING!\r\n")
		return CharacterCommandHandled
	}
	if victim.GetPosition() <= combat.PosSleeping {
		mobileSend(ctx, mob, "Whats the point of doing that now?\r\n")
		return CharacterCommandHandled
	}
	percent := dprng.Number(1, 121)
	if victim.GetLevel() >= LVL_IMMORT {
		percent = 101
	}
	if targetMob, ok := victim.(*MobInstance); ok && targetMob.HasMobFlag(MobFlagNobash) {
		percent = 101
	}
	percent = cByte(percent + max(victim.GetLevel()-mob.GetLevel(), 0))
	if percent > prob {
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillTripNum)
		mob.SetPosition(combat.PosSitting)
		mob.SetWaitStatePulses(engine.PULSE_VIOLENCE)
	} else if ctx.World.mobileCommandSkillDamage(mob, victim, mob.GetLevel()/2+1, SkillTripNum) {
		victim.SetPosition(combat.PosSitting)
		if waiter, ok := victim.(interface{ SetWaitStatePulses(int) }); ok {
			waiter.SetWaitStatePulses(engine.PULSE_VIOLENCE)
		}
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE * 2)
	return CharacterCommandHandled
}

func characterParry(ctx CharacterCommandContext, mob *MobInstance) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumParry)
	if prob == 0 {
		mobileSend(ctx, mob, "You're not good enough at swordplay to parry!\r\n")
		if ctx.Subcmd == 0 {
			return CharacterCommandHandled
		}
	}
	victim := mob.GetFightingBody()
	if victim == nil {
		mobileSend(ctx, mob, "But you aren't fighting anyone!\r\n")
		return CharacterCommandHandled
	}
	if victim.GetFightingBody() != mob {
		mobileSend(ctx, mob, "But noone's attacking you!\r\n")
		return CharacterCommandHandled
	}
	if mobileWielded(mob) == nil {
		mobileSend(ctx, mob, "Parry with what? You're unarmed!\r\n")
		return CharacterCommandHandled
	}

	percent := dprng.Number(1, 101)
	if ctx.Subcmd != 0 {
		prob = dprng.Number(50, 100)
	}
	if percent > prob {
		mobileSend(ctx, mob, "With a dazzling show of swordplay, you attempt to parry...but are outmaneuvered!\r\n")
		mob.SetWaitState(3)
		return CharacterCommandHandled
	}

	mobileSend(ctx, mob, "With a dazzling show of swordplay, you move into defensive position...\r\n")
	Act(ctx.World, true, mob, asActor(victim), nil, nil,
		"$n displays a dazzling show of swordplay, fending off $N's every blow!", "", ToRoom)
	Act(ctx.World, true, mob, asActor(victim), nil, nil,
		"$n displays a dazzling show of swordplay, fending off your every blow!", "", ToVict)
	if marker, ok := ctx.World.combatEngine.(interface {
		MarkParried(body combat.Combatant, action string)
	}); ok {
		marker.MarkParried(victim, "parry")
	}
	mob.SetWaitState(2)
	return CharacterCommandHandled
}

func characterRescue(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumRescue)
	if prob == 0 {
		mobileSend(ctx, mob, "But only true warriors can do this!\r\n")
		if ctx.Subcmd == 0 {
			return CharacterCommandHandled
		}
	}
	victim := mobileCommandTarget(ctx, mob, args, false)
	if victim == nil {
		mobileSend(ctx, mob, "Whom do you want to rescue?\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "What about fleeing instead?\r\n")
		return CharacterCommandHandled
	}
	if mob.GetFightingBody() == victim {
		mobileSend(ctx, mob, "How can you rescue someone you are trying to kill?\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	// IS_OUTLAW(ch) is always false for an NPC, including a switched body.
	if ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		mobileSend(ctx, mob, "This room just has such a peaceful, easy feeling...\r\n")
		return CharacterCommandHandled
	}

	var attacker combat.Combatant
	for _, raw := range ctx.World.GetAllCharsInRoom(mob.GetRoomVNum()) {
		candidate, ok := raw.(combat.Combatant)
		if ok && candidate.GetFightingBody() == victim {
			attacker = candidate
			break
		}
	}
	if attacker == nil {
		ActWithPrivateSink(ctx.World, false, mob, asActor(victim), nil, nil,
			"But nobody is fighting $M!", "", ToChar, mob, ctx.PrivateOutput)
		return CharacterCommandHandled
	}

	percent := dprng.Number(1, 101)
	if ctx.Subcmd != 0 {
		prob = 100
	}
	if percent > prob {
		mobileSend(ctx, mob, "You fail the rescue!\r\n")
		return CharacterCommandHandled
	}

	mobileSend(ctx, mob, "Banzai!  To the rescue...\r\n")
	Act(ctx.World, false, asActor(victim), mob, nil, nil,
		"You are rescued by $N, you are confused!", "", ToChar)
	Act(ctx.World, false, mob, asActor(victim), nil, nil,
		"$n heroically rescues $N!", "", ToNotVict)

	if stopper, ok := ctx.World.combatEngine.(interface{ StopCombat(combat.Combatant) }); ok {
		stopper.StopCombat(victim)
		stopper.StopCombat(attacker)
		stopper.StopCombat(mob)
	}
	victim.StopFighting()
	attacker.StopFighting()
	mob.StopFighting()
	mob.SetFightingBody(attacker)
	attacker.SetFightingBody(mob)
	if err := ctx.World.StartCombat(mob, attacker); err != nil {
		slog.Warn("switched-body rescue combat entry failed", "rescuer", mob.GetName(), "target", attacker.GetName(), "error", err)
	}
	if ctx.Subcmd != 0 {
		ctx.World.MobFirstHit(victim, attacker)
	}
	if waiter, ok := victim.(interface{ SetWaitState(int) }); ok {
		waiter.SetWaitState(2)
	}
	return CharacterCommandHandled
}

// characterSlug ports the switched-body do_slug path. GET_SKILL is the shared
// dummy_mob value for an NPC; target fallback and the command gates follow
// src/new_cmds.c:820-879.
func characterSlug(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	if ctx.World.DummyMobSkill(skillNumSlug) == 0 {
		mobileSend(ctx, mob, "You couldn't slug your way out of a wet paper bag.\n\r")
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, true)
	if victim == nil {
		mobileSend(ctx, mob, "Slug who?\n\r")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "You curl up your fist and slug yourself in the nose! Ouch!\n\r")
		return CharacterCommandHandled
	}
	if mobileWielded(mob) != nil {
		mobileSend(ctx, mob, "You can't make a fist while wielding a weapon!\n\r")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}

	percent := dprng.Number(1, 101)
	if percent > ctx.World.DummyMobSkill(skillNumSlug) {
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillSlugNum)
	} else {
		// C multiplies by number(1,4)*.5 before converting the result to int.
		// Keep the second draw after the hit check (new_cmds.c:859-870).
		damage := int(float64(mob.GetLevel()) * (float64(dprng.Number(1, 4)) * 0.5))
		ctx.World.mobileCommandSkillDamage(mob, victim, damage, SkillSlugNum)
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE * 2)
	return CharacterCommandHandled
}

// characterSerpentKick ports do_serpent_kick for a mobile command body.
// C uses the same shared NPC skill table and performs the training spawn after
// damage; see src/new_cmds2.c:693-728.
func characterSerpentKick(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumSerpentKick)
	if prob == 0 {
		mobileSend(ctx, mob, "You'd better leave all the martial arts to others.\r\n")
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, true)
	if victim == nil {
		mobileSend(ctx, mob, "Kick who?\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "Aren't we funny today...\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}

	percent := ((7 - (victim.GetAC() / 10)) << 1) + dprng.Number(1, 101)
	if victim.GetPosition() <= combat.PosSleeping {
		prob = 110
	}
	if percent > prob {
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillSerpentKickNum)
	} else {
		damage := int(float64(mob.GetLevel()) * 1.5)
		ctx.World.mobileCommandSkillDamage(mob, victim, damage, SkillSerpentKickNum)
		// C evaluates number(0,80) before checking the level threshold.
		// CreateMobileI owns create_mobile's level and hunting state.
		// #nosec G404 -- matching the C game RNG draw.
		if dprng.Number(0, 80) == 0 && mob.GetLevel() > 18 {
			if _, err := ctx.World.CreateMobileI(mob, 18221, huntingStartRoom, mob.GetLevel()+3, true); err != nil {
				slog.Error("serpent-kick training mobile spawn failed", "mob_vnum", 18221, "error", err)
			}
		}
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE * 2)
	return CharacterCommandHandled
}

// characterSmackheads ports the two-victim do_smackheads body. C's second
// half_chop argument remains a full name for get_char_room_vis, while the
// first is one_argument's first token (src/new_cmds.c:966-1097).
func characterSmackheads(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	if ctx.World.DummyMobSkill(skillNumSmackheads) == 0 {
		mobileSend(ctx, mob, "The only heads you're gonna smack are yours and Rosie's.\n\r")
		return CharacterCommandHandled
	}
	raw := ctx.RawArgs
	if raw == "" && len(args) > 0 {
		raw = strings.Join(args, " ")
	}
	name1, rest := OneArgument(raw)
	name2 := strings.TrimSpace(rest)
	victim1, found1 := ctx.World.ResolveCharInRoom(mob, name1)
	victim2, found2 := ctx.World.ResolveCharInRoom(mob, name2)
	if !found1 || !found2 || victim1.Combatant == victim2.Combatant {
		mobileSend(ctx, mob, "Looks like the gangs not all here...\n\r")
		return CharacterCommandHandled
	}
	first, second := victim1.Combatant, victim2.Combatant
	if first == mob || second == mob {
		mobileSend(ctx, mob, "We call that 'headbutt' around here, son...\n\r")
		return CharacterCommandHandled
	}
	if mobileWielded(mob) != nil {
		mobileSend(ctx, mob, "You need your hands free to smack some heads!\n\r")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if mob.GetFightingBody() != nil {
		mobileSend(ctx, mob, "You're a little busy right now!\r\n")
		return CharacterCommandHandled
	}
	if first.GetFightingBody() != nil || second.GetFightingBody() != nil {
		mobileSend(ctx, mob, "They are too busy fighting at the moment!\r\n")
		return CharacterCommandHandled
	}
	if ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		mobileSend(ctx, mob, "You can't commit acts of violence here!\r\n")
		return CharacterCommandHandled
	}

	percent := cByte(((5 - ((first.GetAC()+second.GetAC())/2)/10) << 1) + dprng.Number(1, 101))
	if percent > ctx.World.DummyMobSkill(skillNumSmackheads) {
		actorMessage := fmt.Sprintf("%s and %s slip out of your hands!", first.GetName(), second.GetName())
		roomMessage := fmt.Sprintf("%s and %s duck as %s lunges at them!", first.GetName(), second.GetName(), mob.GetName())
		mobileActToChar(ctx, mob, nil, nil, actorMessage)
		Act(ctx.World, false, mob, nil, nil, nil, roomMessage, "", ToRoom)
		ctx.World.mobileCommandSkillDamage(mob, first, 0, SkillSmackheadsNum)
		ctx.World.mobileCommandSkillDamage(mob, second, 0, SkillSmackheadsNum)
	} else {
		actorMessage := fmt.Sprintf("You grab the heads of %s and %s and bang them together with a sickening *SMACK*.", first.GetName(), second.GetName())
		roomMessage := fmt.Sprintf("%s grabs the heads of %s and %s and bangs them together with a sickening *SMACK*.", mob.GetName(), first.GetName(), second.GetName())
		mobileActToChar(ctx, mob, nil, nil, actorMessage)
		Act(ctx.World, false, mob, nil, nil, nil, roomMessage, "", ToRoom)
		first.SetPosition(combat.PosStunned)
		second.SetPosition(combat.PosStunned)
		first.SetPosition(combat.GetPositionFromHP(first.GetHP(), first.GetPosition()))
		second.SetPosition(combat.GetPositionFromHP(second.GetHP(), second.GetPosition()))
		ctx.World.mobileCommandSkillDamage(mob, first, 3*mob.GetLevel(), SkillSmackheadsNum)
		ctx.World.mobileCommandSkillDamage(mob, second, 3*mob.GetLevel(), SkillSmackheadsNum)
		setCombatWait(first, engine.PULSE_VIOLENCE*3)
		setCombatWait(second, engine.PULSE_VIOLENCE*3)
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE * 3)
	return CharacterCommandHandled
}

// characterSpikeStake ports do_spike's actor body. Its callers include both
// the registered commands and breed_killer's synchronous subcommand call
// (src/new_cmds.c:1098-1198; src/spec_procs2.c:1678-1717).
func characterSpikeStake(ctx CharacterCommandContext, mob *MobInstance, commandName string, args []string) CharacterCommandResult {
	weaponName := "spike"
	if strings.EqualFold(commandName, "stake") || ctx.Subcmd == 1 {
		weaponName = "stake"
	}
	if mob.GetFightingBody() != nil {
		mobileSend(ctx, mob, fmt.Sprintf("You can't %s someone while fighting!\r\n", weaponName))
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, false)
	if victim == nil {
		if characterSkillArgument(ctx, args) == "" {
			mobileSend(ctx, mob, fmt.Sprintf("Whom do you wish to %s?\r\n", weaponName))
		} else {
			mobileSend(ctx, mob, "No-one by that name here.\r\n")
		}
		return CharacterCommandHandled
	}
	weapon := mobileWielded(mob)
	if weapon == nil || !strings.Contains(objName(weapon, mob), weaponName) {
		mobileSend(ctx, mob, fmt.Sprintf("You need to wield a %s to succeed!\r\n", weaponName))
		return CharacterCommandHandled
	}
	if ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		mobileSend(ctx, mob, "You can't commit murder in this holy place!\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "The monster in you won't let you suicide!\r\n")
		return CharacterCommandHandled
	}
	var victimAffected func(int) bool
	if player, ok := victim.(*Player); ok {
		victimAffected = player.IsAffected
	} else if targetMob, ok := victim.(*MobInstance); ok {
		victimAffected = targetMob.IsAffected
	} else {
		mobileSend(ctx, mob, fmt.Sprintf("You can't %s that!\r\n", weaponName))
		return CharacterCommandHandled
	}
	if weaponName == "spike" && !victimAffected(affWerewolf) {
		mobileSend(ctx, mob, "Spiking is only for werewolves..\r\n")
		return CharacterCommandHandled
	}
	if weaponName == "stake" && !victimAffected(affVampire) {
		mobileSend(ctx, mob, "Staking is only for vampires..\r\n")
		return CharacterCommandHandled
	}
	// PLR_FLAGGED is defined to false for an NPC, including the switched body
	// (src/utils.h:234); this path therefore has no same-kind actor refusal.
	if victim.GetLevel() >= LVL_IMMORT && mob.GetLevel() < LVL_IMMORT {
		mobileSend(ctx, mob, "Yeah, right.\r\n")
		return CharacterCommandHandled
	}

	success := mob.GetLevel() > victim.GetLevel()
	if !success {
		// Preserve C's short-circuit: the level gap roll happens only when the
		// actor is not higher level, then sleep is tested (src/new_cmds.c:1159-1163).
		// #nosec G404 -- C's gameplay RNG draw.
		success = victim.GetLevel()-mob.GetLevel() < dprng.Number(0, LVL_IMMORT) || victim.GetPosition() <= combat.PosSleeping
	}
	if success {
		ActWithPrivateSink(ctx.World, false, mob, asActor(victim), weapon, nil,
			"You drive $p into $S chest!", "", ToChar, mob, ctx.PrivateOutput)
		Act(ctx.World, false, mob, asActor(victim), weapon, nil, "$n drives $p into the chest of $N!", "", ToNotVict)
		Act(ctx.World, false, mob, asActor(victim), weapon, nil, "$n drives $p into your chest with a solid blow!", "", ToVict)
		if player, ok := victim.(*Player); ok {
			if player.GetFlags()&(1<<uint(PlrVampire)) != 0 {
				player.SetPlrFlag(PlrVampire, false)
			}
			if player.GetFlags()&(1<<uint(PlrWerewolf)) != 0 {
				player.SetPlrFlag(PlrWerewolf, false)
			}
		}
		room := ctx.World.GetRoomInWorld(victim.GetRoom())
		if room != nil {
			MudLog(fmt.Sprintf("%s %sd %s at %s.", mob.GetName(), weaponName, victim.GetName(), room.Name), MudlogBrief, LVL_IMMORT, true)
		}
		ctx.World.RawKillCombatant(victim, combat.TYPE_UNDEFINED)
	} else {
		ActWithPrivateSink(ctx.World, false, mob, asActor(victim), weapon, nil,
			"$N twists at the last moment, and you miss!", "", ToChar, mob, ctx.PrivateOutput)
		Act(ctx.World, false, mob, asActor(victim), weapon, nil, "$N growls in anger as $n tries to drive a $p into $M!", "", ToNotVict)
		Act(ctx.World, false, mob, asActor(victim), weapon, nil, "$n comes at you with a $p, but you dodge the attempt!", "", ToVict)
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE * 2)
	return CharacterCommandHandled
}

func characterSleeper(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumSleeper)
	if prob == 0 {
		mobileSend(ctx, mob, "You have no idea how.\r\n")
		return CharacterCommandHandled
	}
	if mob.GetFightingBody() != nil {
		mobileSend(ctx, mob, "You can't do this while fighting!\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		mobileSend(ctx, mob, "This room just has such a peaceful, easy feeling...\r\n")
		return CharacterCommandHandled
	}
	if mobileWielded(mob) != nil {
		mobileSend(ctx, mob, "You can't get a good grip on them while you are holding that weapon!\r\n")
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, false)
	if victim == nil {
		mobileSend(ctx, mob, "Sleeper who?\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "Can't get to sleep fast enough, huh?\r\n")
		return CharacterCommandHandled
	}
	if !victim.IsNPC() && ctx.Subcmd == 0 {
		mobileSend(ctx, mob, "You can not sleeper them because you are not an Outlaw!\r\n")
		victim.SendMessage(mob.GetName() + " failed to sleeper you because " + mob.GetName() + " is not an Outlaw.\r\n")
		return CharacterCommandHandled
	}
	if victim.GetFightingBody() != nil {
		mobileSend(ctx, mob, "You can't get a good grip on them while they're fighting!\r\n")
		return CharacterCommandHandled
	}
	if isShopKeeperInWorld(ctx.World, victim) {
		mobileSend(ctx, mob, "Ha Ha. Don't think so.\r\n")
		return CharacterCommandHandled
	}
	if victim.GetPosition() <= combat.PosSleeping {
		mobileSend(ctx, mob, "What's the point of doing that now?\r\n")
		return CharacterCommandHandled
	}

	percent := dprng.Number(1, 101+victim.GetLevel())
	if targetMob, ok := victim.(*MobInstance); ok && (targetMob.HasMobFlag(MobFlagAware) || targetMob.HasMobFlag(MobFlagNosleep)) {
		prob = 0
	}
	percent += max(victim.GetLevel()-mob.GetLevel(), 0)
	if percent > prob {
		ActWithPrivateSink(ctx.World, true, mob, asActor(victim), nil, nil,
			"You try to grab $N in a sleeper hold but fail!", "", ToChar, mob, ctx.PrivateOutput)
		Act(ctx.World, true, mob, asActor(victim), nil, nil,
			"$n tries to put a sleeper hold on you, but you break free!", "", ToVict)
		Act(ctx.World, true, mob, asActor(victim), nil, nil,
			"$n tries to put $N in a sleeper hold...", "", ToNotVict)
		ctx.World.MobFirstHit(victim, mob)
		mob.SetWaitState(2)
		return CharacterCommandHandled
	}

	ActWithPrivateSink(ctx.World, true, mob, asActor(victim), nil, nil,
		"You put $N in a sleeper hold.", "", ToChar, mob, ctx.PrivateOutput)
	Act(ctx.World, true, mob, asActor(victim), nil, nil,
		"You feel very sleepy... Zzzzz..", "", ToVict)
	Act(ctx.World, true, mob, asActor(victim), nil, nil,
		"$n puts $N in a sleeper hold.", "", ToRoom)
	Act(ctx.World, true, mob, asActor(victim), nil, nil,
		"$N goes to sleep.", "", ToRoom)
	victim.SetPosition(combat.PosSleeping)
	affect := engine.NewAffectDirect(skillNumSleeper, engine.ApplyNone, mob.GetLevel()/9, 0, engine.AFFSleep, SkillSleeper)
	switch target := victim.(type) {
	case *Player:
		target.SetAffect(affSleep, true)
		target.AddAffect(affect)
	case *MobInstance:
		target.AddAffect(affect)
	}
	mob.SetWaitState(2)
	return CharacterCommandHandled
}

func characterStrike(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	percent := dprng.Number(1, 101)
	prob := ctx.World.DummyMobSkill(skillNumStrike)
	if prob == 0 {
		mobileSend(ctx, mob, "Yeah, right.\r\n")
		return CharacterCommandHandled
	}
	name := characterSkillArgument(ctx, args)
	var victim combat.Combatant
	if name == "" {
		victim = mob.GetFightingBody()
		if victim == nil {
			mobileSend(ctx, mob, "Strike who?\r\n")
			return CharacterCommandHandled
		}
	} else {
		victim = mobileCommandTarget(ctx, mob, args, false)
		if victim == nil {
			mobileSend(ctx, mob, "They don't seem to be here.\r\n")
			return CharacterCommandHandled
		}
	}
	if victim == mob {
		ActWithPrivateSink(ctx.World, false, mob, nil, nil, nil,
			"You beat yourself about the face and neck.", "", ToChar, mob, ctx.PrivateOutput)
		Act(ctx.World, false, mob, nil, nil, nil,
			"$n slaps $mself around a little.", "", ToRoom)
		return CharacterCommandHandled
	}
	if ctx.World.IsCharmedI(mob) && ctx.World.FollowingBody(mob) == victim {
		ActWithPrivateSink(ctx.World, false, mob, asActor(victim), nil, nil,
			"$N is just such a good friend, you simply can't strike $M.", "", ToChar, mob, ctx.PrivateOutput)
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if victim.GetPosition() <= combat.PosSleeping {
		prob = 100
	}
	damage := 0
	if percent < prob {
		damage = int(float64(mob.GetLevel()) * 0.65)
	}
	ctx.World.mobileCommandSkillDamage(mob, victim, damage, skillNumStrike)
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE + 2)
	return CharacterCommandHandled
}

func characterSubdue(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumSubdue)
	if prob == 0 {
		mobileSend(ctx, mob, "You have no idea how!\r\n")
		if ctx.Subcmd == 0 {
			return CharacterCommandHandled
		}
	}
	if mob.GetFightingBody() != nil {
		mobileSend(ctx, mob, "You're too busy right now!\r\n")
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, false)
	if victim == nil {
		mobileSend(ctx, mob, "Subdue who?\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "Aren't we funny today...\r\n")
		return CharacterCommandHandled
	}
	if ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		mobileSend(ctx, mob, "You can't contemplate violence in such a place!\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if victim.GetFightingBody() != nil {
		mobileSend(ctx, mob, "You can't get close enough!\r\n")
		return CharacterCommandHandled
	}
	if !victim.IsNPC() && ctx.Subcmd == 0 {
		mobileSend(ctx, mob, "You can't subdue them because you are not an Outlaw!\r\n")
		victim.SendMessage(mob.GetName() + " failed to subdue you because " + mob.GetName() + " is not an Outlaw.\r\n")
		return CharacterCommandHandled
	}
	if isShopKeeperInWorld(ctx.World, victim) {
		mobileSend(ctx, mob, "Haha.. Don't think so.\r\n")
		return CharacterCommandHandled
	}
	if victim.GetPosition() <= combat.PosStunned {
		mobileSend(ctx, mob, "What's the point of doing that now?\r\n")
		return CharacterCommandHandled
	}

	percent := dprng.Number(1, 101+victim.GetLevel())
	if mob.GetLevel() >= LVL_IMMORT {
		percent = 0
	}
	if targetMob, ok := victim.(*MobInstance); ok && targetMob.HasMobFlag(MobFlagAware) {
		prob = 0
	}
	percent += max(victim.GetLevel()-mob.GetLevel(), 0)
	if percent > prob {
		Act(ctx.World, true, mob, asActor(victim), nil, nil,
			"$n misses a blow to the back of your head.", "", ToVict)
		ActWithPrivateSink(ctx.World, true, mob, asActor(victim), nil, nil,
			"$N avoids your misplaced blow to the back of $S head.", "", ToChar, mob, ctx.PrivateOutput)
		Act(ctx.World, true, mob, asActor(victim), nil, nil,
			"$N avoids $n's misplaced blow to the back of $S head.", "", ToNotVict)
		ctx.World.MobFirstHit(victim, mob)
	} else {
		victim.SendMessage("Someone sneaks up behind you and knocks you out!\r\n")
		ActWithPrivateSink(ctx.World, true, mob, asActor(victim), nil, nil,
			"You knock $M out cold.", "", ToChar, mob, ctx.PrivateOutput)
		Act(ctx.World, true, mob, asActor(victim), nil, nil,
			"$n knocks out $N with a well-placed blow to the back of the head.", "", ToNotVict)
		victim.SetPosition(combat.PosStunned)
	}
	mob.SetWaitState(3)
	return CharacterCommandHandled
}

func characterTag(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	victim := mobileCommandTarget(ctx, mob, args, false)
	if victim == nil {
		mobileSend(ctx, mob, "They don't seem to be here.\r\n")
		return CharacterCommandHandled
	}
	if player, ok := victim.(*Player); ok && !ctx.World.playerHasDescriptor(player) {
		mobileSend(ctx, mob, "No way! They're linkless!\r\n")
		return CharacterCommandHandled
	}
	var it *Player
	for _, player := range ctx.World.playingPlayers() {
		if player.GetFlags()&(1<<uint(PlrIt)) != 0 {
			it = player
		}
	}
	if it != nil {
		mobileSend(ctx, mob, "But you are not it!\r\n")
		return CharacterCommandHandled
	}
	if player, ok := victim.(*Player); ok && player == it {
		mobileSend(ctx, mob, "You're it already!\r\n")
		return CharacterCommandHandled
	}
	if victim.IsNPC() {
		ActWithPrivateSink(ctx.World, true, mob, asActor(victim), nil, nil,
			"You tag $N, but $E tags you right back!\r\n", "", ToChar, mob, ctx.PrivateOutput)
		return CharacterCommandHandled
	}
	player := victim.(*Player)
	if player.GetFlags()&(1<<uint(PlrIt)) != 0 {
		player.ClearPLRFlag(PlrIt)
	}
	player.SetPLRFlag(PlrIt)
	Act(ctx.World, true, mob, player, nil, nil,
		"$n taps $N and screams, 'TAG! You're it!'", "", ToNotVict)
	Act(ctx.World, true, mob, player, nil, nil,
		"$n taps you and screams, 'TAG! You're it!'", "", ToVict)
	ActWithPrivateSink(ctx.World, true, mob, player, nil, nil,
		"You tap $N and scream, 'TAG! You're it!'", "", ToChar, mob, ctx.PrivateOutput)
	player.SetWaitState(2)
	return CharacterCommandHandled
}
