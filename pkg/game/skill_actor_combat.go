package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

const (
	skillNumBackstab    = 131
	skillNumAmbush      = 191
	skillNumBash        = 132
	skillNumCutthroat   = 143
	skillNumCharge      = 147
	skillNumCircle      = 173
	skillNumGroinrip    = 174
	skillNumDisarm      = 177
	skillNumDisembowel  = 184
	skillNumDragonKick  = 188
	skillNumHeadbutt    = 141
	skillNumKick        = 134
	skillNumNeckbreak   = 190
	skillNumParry       = 172
	skillNumRescue      = 137
	skillNumSerpentKick = 156
	skillNumShoot       = 148
	skillNumSlug        = 146
	skillNumSmackheads  = 145
	skillNumSleeper     = 187
	skillNumStrike      = 155
	skillNumSubdue      = 152
	skillNumTigerPunch  = 189
	skillNumTrip        = 144
)

// CharacterCombatSkill routes the mobile body of the selected combat skills.
// The command registration, level lookup, and position gate remain shared with
// player dispatch; this callback ports the do_* body after those interpreter
// gates (src/interpreter.c:910-947).
func CharacterCombatSkill(ctx CharacterCommandContext, commandName string, args []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	switch strings.ToLower(commandName) {
	case "ambush":
		return characterAmbush(ctx, args)
	case "backstab":
		return characterBackstab(ctx, mob, args)
	case "bash":
		return characterBash(ctx, mob, args)
	case "charge":
		return characterCharge(ctx, mob, args)
	case "circle":
		return characterCircle(ctx, mob, args)
	case "cutthroat":
		return characterCutthroat(ctx, mob, args)
	case "disarm":
		return characterDisarm(ctx, mob, args)
	case "disembowel":
		return characterDisembowel(ctx, mob, args)
	case "dragon":
		return characterDragonKick(ctx, mob, args)
	case "groinrip":
		return characterGroinrip(ctx, mob, args)
	case "headbutt":
		return characterHeadbutt(ctx, mob, args)
	case "kick":
		return characterKick(ctx, mob, args)
	case "neckbreak":
		return characterNeckbreak(ctx, mob, args)
	case "parry":
		return characterParry(ctx, mob)
	case "rescue":
		return characterRescue(ctx, mob, args)
	case "serpent":
		return characterSerpentKick(ctx, mob, args)
	case "shoot":
		return characterShoot(ctx, args)
	case "spike", "stake":
		return characterSpikeStake(ctx, mob, commandName, args)
	case "slug":
		return characterSlug(ctx, mob, args)
	case "smackheads":
		return characterSmackheads(ctx, mob, args)
	case "sleeper":
		return characterSleeper(ctx, mob, args)
	case "strike":
		return characterStrike(ctx, mob, args)
	case "subdue":
		return characterSubdue(ctx, mob, args)
	case "tiger":
		return characterTigerPunch(ctx, mob, args)
	case "tag":
		return characterTag(ctx, mob, args)
	case "trip":
		return characterTrip(ctx, mob, args)
	default:
		return CharacterCommandUnsupported
	}
}

func characterSkillArgument(ctx CharacterCommandContext, args []string) string {
	raw := ctx.RawArgs
	if raw == "" && len(args) > 0 {
		raw = strings.Join(args, " ")
	}
	name, _ := OneArgument(raw)
	return name
}

func mobileCommandTarget(ctx CharacterCommandContext, mob *MobInstance, args []string, fightingFallback bool) combat.Combatant {
	return mobileCommandTargetWithRoomFallback(ctx, mob, args, fightingFallback, false)
}

func mobileCommandTargetWithRoomFallback(ctx CharacterCommandContext, mob *MobInstance, args []string, fightingFallback, sameRoom bool) combat.Combatant {
	name := characterSkillArgument(ctx, args)
	if name != "" {
		if target, found := ctx.World.ResolveCharInRoom(mob, name); found {
			return target.Combatant
		}
	}
	if fightingFallback {
		if target, found := ctx.World.ResolveFightingTarget(mob); found {
			if !sameRoom || target.Combatant.GetRoom() == mob.GetRoom() {
				return target.Combatant
			}
		}
	}
	return nil
}

func mobileWielded(mob *MobInstance) *ObjectInstance {
	if mob == nil {
		return nil
	}
	return mob.Equipped(mobWearWield)
}

func combatantWielded(body combat.Combatant) *ObjectInstance {
	switch target := body.(type) {
	case *MobInstance:
		return mobileWielded(target)
	case *Player:
		if target.Equipment == nil {
			return nil
		}
		weapon, _ := target.Equipment.GetItemInSlot(SlotWield)
		return weapon
	default:
		return nil
	}
}

func mobileWeaponType(obj *ObjectInstance) (int, bool) {
	if obj == nil || obj.Prototype == nil {
		return 0, false
	}
	return obj.GetValue(3), true
}

// cByte mirrors the platform C `byte` typedef, which is plain `char` in
// src/structs.h:703 and signed on the reference build. Probability handlers
// that assign a wider result to `byte` therefore retain signed 8-bit wrap.
func cByte(value int) int {
	return mobileSignedPoint(value, 8)
}

func characterBackstab(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumBackstab)
	if prob == 0 {
		mobileSend(ctx, mob, "You have no idea how.\r\n")
		if ctx.Subcmd == 0 {
			return CharacterCommandHandled
		}
	}
	victim := mobileCommandTarget(ctx, mob, args, false)
	if victim == nil {
		mobileSend(ctx, mob, "Backstab who?\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "How can you sneak up on yourself?\r\n")
		return CharacterCommandHandled
	}
	weapon := mobileWielded(mob)
	if weapon == nil {
		mobileSend(ctx, mob, "You need to wield a weapon to make it a success.\r\n")
		return CharacterCommandHandled
	}
	weaponType, ok := mobileWeaponType(weapon)
	if !ok || weaponType != 11 {
		mobileSend(ctx, mob, "Only piercing weapons can be used for backstabbing.\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if victim.GetFightingBody() != nil {
		mobileSend(ctx, mob, "You can't backstab a fighting person -- they're too alert!\r\n")
		return CharacterCommandHandled
	}
	if targetMob, isMob := victim.(*MobInstance); isMob && targetMob.HasMobFlag(MobFlagAware) && victim.GetPosition() > combat.PosSleeping {
		characterMobileAwareness(ctx, mob, victim)
		ctx.World.MobFirstHit(victim, mob)
		return CharacterCommandHandled
	}
	percent := dprng.Number(1, 101)
	if ctx.Subcmd != 0 {
		prob = dprng.Number(50, 100)
	}
	if victim.GetPosition() > combat.PosSleeping && percent > prob {
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillBackstabNum)
	} else {
		mobileSkillHit(ctx.World, mob, victim, SkillBackstabNum)
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE)
	return CharacterCommandHandled
}

func characterBash(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumBash)
	if prob == 0 {
		mobileSend(ctx, mob, "You'd better leave all the martial arts to fighters.\r\n")
		if ctx.Subcmd == 0 {
			return CharacterCommandHandled
		}
	}
	if ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		mobileSend(ctx, mob, "This room just has such a peaceful, easy feeling...\r\n")
		return CharacterCommandHandled
	}
	victim := mobileCommandTargetWithRoomFallback(ctx, mob, args, true, true)
	if victim == nil {
		mobileSend(ctx, mob, "Bash who?\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "Aren't we funny today...\r\n")
		return CharacterCommandHandled
	}
	if victim.GetPosition() < combat.PosFighting {
		mobileSend(ctx, mob, "You can't bash someone who's sitting already!\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if !mob.SpendMove(10) {
		mobileSend(ctx, mob, "You haven't the energy!\r\n")
		return CharacterCommandHandled
	}
	percent := ((5 - (victim.GetAC() / 10)) * 2) + dprng.Number(1, 101)
	if ctx.Subcmd != 0 {
		prob = 131
	}
	if targetMob, isMob := victim.(*MobInstance); isMob && targetMob.HasMobFlag(MobFlagNobash) && mob.GetLevel() < LVL_IMMORT {
		percent = 101
	}
	if victim.GetPosition() <= combat.PosSleeping || mob.GetLevel() >= LVL_IMMORT {
		percent = 0
	}
	if percent > prob {
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillBashNum)
		mob.SetPosition(combat.PosSitting)
		mob.SetWaitStatePulses(engine.PULSE_VIOLENCE)
	} else if ctx.World.mobileCommandSkillDamage(mob, victim, mob.GetLevel()/2+1, SkillBashNum) {
		victim.SetPosition(combat.PosSitting)
		setCombatWait(victim, 2*engine.PULSE_VIOLENCE)
	}
	return CharacterCommandHandled
}

func characterCharge(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumCharge)
	if prob == 0 {
		mobileSend(ctx, mob, "You couldn't charge if you wanted to!\r\n")
		if ctx.Subcmd == 0 {
			return CharacterCommandHandled
		}
	}
	victim := mobileCommandTarget(ctx, mob, args, true)
	if victim == nil {
		mobileSend(ctx, mob, "Great! Fine! Charge who?!?!\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "You charge headlong into the ground, impressing everyone..\n\r")
		return CharacterCommandHandled
	}
	weapon := mobileWielded(mob)
	if weapon == nil {
		mobileSend(ctx, mob, "You're barehanded, try it with a sword or lance next time.\n\r")
		return CharacterCommandHandled
	}
	weaponType, ok := mobileWeaponType(weapon)
	if !ok || (weaponType != 12 && weaponType != 3) {
		mobileSend(ctx, mob, "You need sword or a lance to run 'em through!\r\n")
		return CharacterCommandHandled
	}
	percent := cByte(((5 - (victim.GetAC() / 10)) * 2) + dprng.Number(1, 101))
	if mob.IsMountedMob() {
		percent = cByte(percent + 5)
	}
	if targetMob, isMob := victim.(*MobInstance); isMob && targetMob.HasMobFlag(MobFlagNobash) {
		percent = cByte(percent + 25)
	}
	if ctx.Subcmd != 0 {
		prob = 131
	}
	if percent > prob {
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillChargeNum)
		if !mob.IsMountedMob() {
			mob.SetPosition(combat.GetPositionFromHP(mob.GetHP(), combat.PosSitting))
		}
	} else {
		damage := 2 * dprng.Dice(weapon.GetValue(1), weapon.GetValue(2))
		if mob.IsMountedMob() {
			damage += 50
		}
		ctx.World.mobileCommandSkillDamage(mob, victim, damage, SkillChargeNum)
	}
	mob.SetWaitState(2)
	return CharacterCommandHandled
}

func characterCircle(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	victim := mobileCommandTarget(ctx, mob, args, true)
	if victim == nil {
		mobileSend(ctx, mob, "Circle who?\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "How can you stab yourself in the back?\r\n")
		return CharacterCommandHandled
	}
	if fighting := mob.GetFightingBody(); fighting != nil && fighting.GetFightingBody() == mob {
		mobileSend(ctx, mob, "You're a little too busy right now!\r\n")
		return CharacterCommandHandled
	}
	weapon := mobileWielded(mob)
	if weapon == nil {
		mobileSend(ctx, mob, "You need to wield a weapon to make it a success.\r\n")
		return CharacterCommandHandled
	}
	weaponType, ok := mobileWeaponType(weapon)
	if !ok || weaponType != 11 {
		mobileSend(ctx, mob, "Only piercing weapons can be used for backstabbing.\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if targetMob, isMob := victim.(*MobInstance); isMob && targetMob.HasMobFlag(MobFlagAware) && victim.GetPosition() > combat.PosSleeping {
		characterMobileAwareness(ctx, mob, victim)
		if victim.GetFightingBody() == nil {
			ctx.World.MobFirstHit(victim, mob)
		}
		return CharacterCommandHandled
	}
	prob := ctx.World.DummyMobSkill(skillNumCircle)
	if prob <= 0 {
		mobileSend(ctx, mob, "You make a circle in the air.\r\n")
		return CharacterCommandHandled
	}
	percent := dprng.Number(1, 101)
	if victim.GetPosition() > combat.PosSleeping && percent > prob {
		if victim.GetFightingBody() != nil {
			if stopper, ok := ctx.World.combatEngine.(interface{ StopCombat(combat.Combatant) }); ok {
				stopper.StopCombat(victim)
			} else {
				victim.StopFighting()
			}
			ctx.World.MobFirstHit(victim, mob)
		}
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillCircleNum)
	} else {
		mobileSkillHit(ctx.World, mob, victim, SkillCircleNum)
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE + 2)
	return CharacterCommandHandled
}

func characterCutthroat(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumCutthroat)
	if prob == 0 {
		mobileSend(ctx, mob, "You're not trained in slitting throats!\n\r")
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, false)
	if victim == nil {
		mobileSend(ctx, mob, "Cut what throat where?\n\r")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "That would be bad.\n\r")
		return CharacterCommandHandled
	}
	weapon := mobileWielded(mob)
	if weapon == nil {
		mobileSend(ctx, mob, "You need to wield a weapon to make it a success.\n\r")
		return CharacterCommandHandled
	}
	weaponType, ok := mobileWeaponType(weapon)
	if !ok || weaponType != 11 {
		mobileSend(ctx, mob, "Only daggers and such can be used for cutting a throat.\n\r")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	if ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		mobileSend(ctx, mob, "You feel too peaceful to slit a throat!\n\r")
		return CharacterCommandHandled
	}
	if !victim.IsNPC() && victim.GetLevel() <= 10 {
		ActWithPrivateSink(ctx.World, false, mob, asActor(victim), nil, nil,
			"Ancient forces protect $N from your wrath!", "", ToChar, mob, ctx.PrivateOutput)
		return CharacterCommandHandled
	}
	if combatantIsAffected(victim, affCutthroat) {
		mobileSend(ctx, mob, "Their throat is already slit!\r\n")
		return CharacterCommandHandled
	}
	if mob.GetFightingBody() != nil || victim.GetFightingBody() != nil {
		mobileSend(ctx, mob, "You can't get close enough!\r\n")
		return CharacterCommandHandled
	}
	percent := dprng.Number(1, 101)
	if mob.GetLevel() > LVL_IMMORT {
		prob = 102
	}
	if victim.GetLevel() > LVL_IMMORT {
		prob = 255 // byte prob = -1 in C wraps to 255.
	}
	if percent < prob {
		applyMobileCutthroatAffect(victim, mob.GetLevel()*2)
		victim.SendMessage(fmt.Sprintf("Suddenly %s slits your throat!\r\n", mob.GetName()))
		mobileSend(ctx, mob, "You slit their throat from ear to ear!\r\n")
		ctx.World.mobileCommandSkillDamage(mob, victim, mob.GetLevel()/2, SkillCutthroatNum)
	} else {
		mobileSend(ctx, mob, "Your slash at their throat barely misses!\n\r")
		victim.SendMessage(fmt.Sprintf("%s makes a vicious lunge at your throat!\r\n", mob.GetName()))
		mobileSkillHit(ctx.World, mob, victim, TypeHit)
	}
	mob.SetWaitState(2)
	return CharacterCommandHandled
}

func applyMobileCutthroatAffect(victim combat.Combatant, duration int) {
	affect := engine.NewAffectDirect(skillNumCutthroat, ApplyHitroll, duration, -2, engine.AFFCutthroat, SkillCutthroat)
	switch target := victim.(type) {
	case *Player:
		target.SetAffect(affCutthroat, true)
		target.AddAffect(affect)
	case *MobInstance:
		target.JoinAffect(affect, false, false)
	}
}

func characterDisarm(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumDisarm)
	if prob == 0 {
		mobileSend(ctx, mob, "You'd better leave all the martial arts to fighters.\r\n")
		if ctx.Subcmd == 0 {
			return CharacterCommandHandled
		}
	}
	var victim combat.Combatant
	if mob.GetFightingBody() != nil {
		victim = mob.GetFightingBody()
	} else {
		victim = mobileCommandTarget(ctx, mob, args, false)
		if victim == mob {
			mobileSend(ctx, mob, "Just try removing your weapon instead.\r\n")
			return CharacterCommandHandled
		}
	}
	if victim == nil {
		mobileSend(ctx, mob, "Disarm who?\r\n")
		return CharacterCommandHandled
	}
	weapon := combatantWielded(victim)
	if weapon == nil {
		ActWithPrivateSink(ctx.World, false, mob, asActor(victim), nil, nil,
			"$E doesn't have anything to disarm.", "", ToChar, mob, ctx.PrivateOutput)
		return CharacterCommandHandled
	}
	if mob.GetFightingBody() == nil || victim != mob.GetFightingBody() {
		mobileSend(ctx, mob, "You can't disarm them if you aren't fighting them!\r\n")
		return CharacterCommandHandled
	}
	percent := dprng.Number(1, 101+victim.GetLevel())
	if ctx.Subcmd != 0 {
		prob = 200
	}
	if percent < prob {
		ActWithPrivateSink(ctx.World, false, mob, victim, weapon, nil,
			"You disarm $N and $p goes flying!", "", ToChar, mob, ctx.PrivateOutput)
		Act(ctx.World, true, mob, victim, weapon, nil, "$n knocks $p from $N's hand!", "", ToNotVict)
		Act(ctx.World, true, mob, victim, weapon, nil, "$n deftly disarms you, knocking $p from your hand!", "", ToVict)
		if !mobileUnequipWield(victim) {
			return CharacterCommandHandled
		}
		if victim.GetFightingBody() == nil {
			ctx.World.MobFirstHit(victim, mob)
		}
	} else {
		ActWithPrivateSink(ctx.World, false, mob, victim, nil, nil,
			"You try to disarm $N but fail, tumbling to the ground in the process!", "", ToChar, mob, ctx.PrivateOutput)
		Act(ctx.World, true, mob, victim, nil, nil, "$n tries to disarm you but fails and falls flat on $s face instead!", "", ToVict)
		Act(ctx.World, true, mob, victim, nil, nil, "$n tries to disarm $N, but fails and falls flat on $s face!", "", ToNotVict)
		mob.SetPosition(combat.PosSitting)
		if victim.GetFightingBody() == nil {
			ctx.World.MobFirstHit(victim, mob)
		}
	}
	mob.SetWaitState(2)
	return CharacterCommandHandled
}

func mobileUnequipWield(victim combat.Combatant) bool {
	switch target := victim.(type) {
	case *MobInstance:
		return target.UnequipItem(mobWearWield) != nil
	case *Player:
		if target.Equipment == nil {
			return false
		}
		weapon, ok := target.Equipment.GetItemInSlot(SlotWield)
		if !ok || weapon == nil || target.Equipment.Unequip(SlotWield, target.Inventory) != nil {
			return false
		}
		target.applySpellUnequipWipe(weapon)
		target.MarkCrashNeeded()
		return true
	default:
		return false
	}
}

func characterDisembowel(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumDisembowel)
	if prob == 0 {
		mobileSend(ctx, mob, "You have no idea how.\r\n")
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, true)
	if victim == nil {
		mobileSend(ctx, mob, "Disembowel who?\r\n")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "Nah. Hari Kari is for wimps.\r\n")
		return CharacterCommandHandled
	}
	weapon := mobileWielded(mob)
	if weapon == nil {
		mobileSend(ctx, mob, "You need to wield a weapon to make it a success.\r\n")
		return CharacterCommandHandled
	}
	weaponType, ok := mobileWeaponType(weapon)
	if !ok || weaponType != 11 {
		mobileSend(ctx, mob, "Only piercing weapons can be used for disemboweling.\r\n")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	percent := dprng.Number(1, 101)
	prob = dprng.Number(50, 100) // do_disembowel's subcmd-zero branch (src/act.offensive.c:281).
	if victim.GetPosition() > combat.PosSleeping && percent > prob {
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillDisembowelNum)
	} else {
		mobileSkillHit(ctx.World, mob, victim, SkillDisembowelNum)
	}
	mob.SetWaitState(2)
	return CharacterCommandHandled
}

func characterDragonKick(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	prob := ctx.World.DummyMobSkill(skillNumDragonKick)
	if prob == 0 {
		mobileSend(ctx, mob, "What's that, idiot-san?\r\n")
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
	if !mob.SpendMove(10) {
		mobileSend(ctx, mob, "You're too exhausted!\r\n")
		return CharacterCommandHandled
	}
	// do_dragon_kick declares `percent` as int (src/act.offensive.c:638).
	percent := ((5 - (victim.GetAC() / 10)) * 2) + dprng.Number(1, 101)
	if percent > prob {
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillDragonKickNum)
	} else {
		ctx.World.mobileCommandSkillDamage(mob, victim, int(float64(mob.GetLevel())*1.5), SkillDragonKickNum)
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE + 2)
	return CharacterCommandHandled
}

func characterGroinrip(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	if ctx.PrivateOutput == nil {
		return CharacterCommandHandled
	}
	if ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		mobileSend(ctx, mob, "You cannot commit acts of violence here!\r\n")
		return CharacterCommandHandled
	}
	prob := ctx.World.DummyMobSkill(skillNumGroinrip)
	if prob == 0 {
		mobileSend(ctx, mob, "You're not trained in martial arts!\n\r")
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, true)
	if victim == nil {
		mobileSend(ctx, mob, "Groinrip who?\n\r")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "No masochism allowed!\r\n")
		return CharacterCommandHandled
	}
	if isShopKeeperInWorld(ctx.World, victim) {
		mobileSend(ctx, mob, "Ha Ha. Don't think so.\r\n")
		return CharacterCommandHandled
	}
	if !victim.IsNPC() && victim.GetLevel() >= LVL_IMMORT {
		mobileSend(ctx, mob, "How dare you try to touch a god!\r\nYou are thrown across the room...\r\n")
		mob.SetPosition(combat.GetPositionFromHP(mob.GetHP(), combat.PosSitting))
		return CharacterCommandHandled
	}
	if victim.GetSex() != SexMale {
		mobileSend(ctx, mob, "Umm, they have nothing there to tug on!\r\n")
		return CharacterCommandHandled
	}
	percent := dprng.Number(1, 121)
	if victim.GetPosition() <= combat.PosSleeping || mob.GetLevel() > LVL_IMMORT {
		percent = 0
	}
	if percent < prob {
		ctx.World.mobileCommandSkillDamage(mob, victim, mob.GetLevel(), SkillGroinripNum)
		victimActor := asActor(victim)
		Act(ctx.World, true, victimActor, nil, nil, nil,
			"$n falls to $s knees, clutching $s groin and throwing up\r\neverywhere!", "", ToRoom)
		ctx.World.MaybeSpawnPuke(mob.GetRoom())
	} else {
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillGroinripNum)
	}
	mob.SetWaitState(2)
	return CharacterCommandHandled
}

func characterHeadbutt(ctx CharacterCommandContext, mob *MobInstance, args []string) CharacterCommandResult {
	if ctx.PrivateOutput == nil && ctx.Subcmd == 0 {
		return CharacterCommandHandled
	}
	if ctx.World.RoomHasFlag(mob.GetRoom(), "peaceful") {
		mobileSend(ctx, mob, "The Gods prevent thy violent act.\r\n")
		return CharacterCommandHandled
	}
	prob := ctx.World.DummyMobSkill(skillNumHeadbutt)
	if prob == 0 {
		mobileSend(ctx, mob, "You aren't qualified to headbutt anyone!\r\n")
		if ctx.Subcmd == 0 {
			return CharacterCommandHandled
		}
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	victim := mobileCommandTarget(ctx, mob, args, true)
	if victim == nil {
		mobileSend(ctx, mob, "Headbutt who?\n\r")
		return CharacterCommandHandled
	}
	if victim == mob {
		mobileSend(ctx, mob, "You bang your head into the nearest wall...\n\r")
		return CharacterCommandHandled
	}
	if victim.GetLevel() >= LVL_IMMORT && !victim.IsNPC() {
		mobileSend(ctx, mob, "How dare you try to headbutt a god!\n\r")
		mobileSend(ctx, mob, "You are thrown across the room...\n\r")
		mob.SetPosition(combat.GetPositionFromHP(mob.GetHP(), combat.PosSitting))
		return CharacterCommandHandled
	}
	percent := dprng.Number(1, 121)
	if victim.GetPosition() <= combat.PosSleeping || mob.GetLevel() > LVL_IMMORT {
		percent = 0
	}
	if targetMob, isMob := victim.(*MobInstance); isMob && targetMob.HasMobFlag(MobFlagNobash) {
		percent = 0
	}
	if mob.GetLevel()/2 > mob.GetHP() {
		mobileSend(ctx, mob, "But that could kill you!\n\r")
		return CharacterCommandHandled
	}
	if ctx.Subcmd != 0 {
		prob = dprng.Number(50, 100)
	}
	if victim.GetPosition() <= combat.PosDead {
		mob.SetWaitStatePulses(3)
		return CharacterCommandHandled
	}
	if percent > prob {
		ctx.World.mobileCommandSkillDamage(mob, victim, 0, SkillHeadbuttNum)
	} else {
		recoil := mob.GetLevel() / 4
		if mob.Equipped(mobWearHead) != nil {
			recoil = mob.GetLevel() / 3
		}
		mob.TakeDamage(recoil)
		ctx.World.mobileCommandSkillDamage(mob, victim, mob.GetLevel(), SkillHeadbuttNum)
		if victim.GetPosition() > combat.PosStunned {
			victim.SetPosition(combat.GetPositionFromHP(victim.GetHP(), combat.PosSitting))
		}
	}
	mob.SetWaitState(3)
	return CharacterCommandHandled
}

func setCombatWait(victim combat.Combatant, pulses int) {
	if waiter, ok := victim.(interface{ SetWaitStatePulses(int) }); ok {
		waiter.SetWaitStatePulses(pulses)
	}
}

func characterMobileAwareness(ctx CharacterCommandContext, mob *MobInstance, victim combat.Combatant) {
	victimActor := asActor(victim)
	Act(ctx.World, false, victimActor, mob, nil, nil, "You notice $N lunging at you!", "", ToChar)
	ActWithPrivateSink(ctx.World, false, victimActor, mob, nil, nil,
		"$e notices you lunging at $m!", "", ToVict, mob, ctx.PrivateOutput)
	Act(ctx.World, false, victimActor, mob, nil, nil, "$n notices $N lunging at $m!", "", ToNotVict)
}

// mobileSkillHit is hit(ch, victim, type) for a mobile command body. The C
// hit() skill arms bypass get_minusdam for backstab/circle/disembowel and use
// the mobile's own damage dice; ordinary skill-tagged hits use the wielded or
// natural attack's normal type and reduction (src/fight.c:1763-1888).
func mobileSkillHit(world *World, mob *MobInstance, victim combat.Combatant, attackType int) {
	if world == nil || mob == nil || victim == nil {
		return
	}
	if mob.GetRoom() != victim.GetRoom() {
		if mob.GetFightingBody() == victim {
			mob.StopFighting()
		}
		return
	}
	defer func() {
		if mob.GetPosition() > combat.PosStunned {
			world.FireMobFightScript(mob, victim, mob.GetRoom())
		}
	}()
	if !combat.CalculateHitChance(mob, victim, mob.HitModifiers()) {
		world.mobileCommandSkillDamage(mob, victim, 0, mobileHitDamageType(mob, attackType))
		return
	}
	combatType := combat.AttackNormal
	switch attackType {
	case SkillBackstabNum:
		combatType = combat.AttackBackstab
	case SkillCircleNum:
		combatType = combat.AttackCircle
	case SkillDisembowelNum:
		combatType = combat.AttackDisembowel
	}
	damage := combat.CalculateDamage(mob, victim, combat.DiceRoll{}, combatType)
	damageType := mobileHitDamageType(mob, attackType)
	switch attackType {
	case SkillBackstabNum:
		damage *= int(combat.BackstabMult(mob.GetLevel()))
		damageType = SkillBackstabNum
	case SkillCircleNum:
		damage *= int(combat.BackstabMult(mob.GetLevel())) / 3
		damageType = SkillCircleNum
	case SkillDisembowelNum:
		damage = mob.GetLevel()*2 + mob.GetDamroll()
		damageType = SkillDisembowelNum
	}
	world.mobileCommandSkillDamage(mob, victim, damage, damageType)
}

func mobileHitDamageType(mob *MobInstance, attackType int) int {
	switch attackType {
	case SkillBackstabNum, SkillCircleNum, SkillDisembowelNum:
		return attackType
	}
	if weapon := mobileWielded(mob); weapon != nil {
		if weaponType, ok := mobileWeaponType(weapon); ok && weapon.GetTypeFlag() == ITEM_WEAPON {
			return combat.TYPE_HIT + weaponType
		}
	}
	if mob.Proto() != nil && mob.Proto().BareHandAttack != 0 {
		return combat.TYPE_HIT + mob.Proto().BareHandAttack
	}
	if mob.IsAffected(affFleshAlter) {
		return combat.TYPE_HIT + fleshAlteredType(mob.GetLevel())
	}
	return combat.TYPE_HIT
}
