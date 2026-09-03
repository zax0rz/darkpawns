package game

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func DoDisembowel(ch *Player, target combat.Combatant) SkillResult {
	if target == nil {
		return SkillResult{Success: false, MessageToCh: "Disembowel who?"}
	}
	if ch.GetSkill(SkillDisembowel) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}
	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "Nah. Hari Kari is for wimps."}
	}
	wielded, _ := ch.Equipment.GetItemInSlot(SlotWield)
	if wielded == nil || wielded.Prototype == nil {
		return SkillResult{Success: false, MessageToCh: "You need to wield a weapon to make it a success."}
	}
	if wielded.Prototype.Values[3] != 11 { // TYPE_PIERCE
		return SkillResult{Success: false, MessageToCh: "Only piercing weapons can be used for disemboweling."}
	}
	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}
	// C draws both the percentage and the normal-player probability even when
	// the target is asleep. The command's subcmd is zero, so prob is
	// number(50,100), not GET_SKILL(ch, SKILL_DISEMBOWEL).
	// #nosec G404 — game RNG
	percent := dprng.Number(1, 101)
	// #nosec G404 — game RNG
	prob := dprng.Number(50, 100)
	if target.GetPosition() > combat.PosSleeping && percent > prob {
		return SkillResult{
			Success:             false,
			SkillMsgType:        SkillDisembowelNum,
			SkillMsgAfterDamage: true,
			SkillMsgInDamage:    true,
			DamageSkill:         SkillDisembowel,
			StartCombat:         true,
			WaitCh:              2,
		}
	}

	// The passed skill roll calls hit(ch, vict, SKILL_DISEMBOWEL). That path
	// consumes the ordinary d20, then consumes the wielded weapon's dice even
	// though disembowel replaces the rolled damage with level*2+damroll.
	if !combat.CalculateHitChance(ch, target, combat.HitModifiers{}) {
		return SkillResult{
			Success:             false,
			SkillMsgType:        SkillDisembowelNum,
			SkillMsgAfterDamage: true,
			SkillMsgInDamage:    true,
			DamageSkill:         SkillDisembowel,
			StartCombat:         true,
			WaitCh:              2,
			DeferredImprove:     []string{SkillDisembowel},
		}
	}
	weaponNum, weaponSides := ch.Equipment.GetWeaponDamage()
	_ = combat.RollDice(weaponNum, weaponSides)
	dam := ch.GetLevel()*2 + ch.GetDamroll()
	return SkillResult{
		Success:             true,
		Damage:              dam,
		SkillMsgType:        SkillDisembowelNum,
		SkillMsgAfterDamage: true,
		SkillMsgInDamage:    true,
		DamageSkill:         SkillDisembowel,
		StartCombat:         true,
		WaitCh:              2,
		DeferredImprove:     []string{SkillDisembowel},
	}
}

// DoDragonKick implements do_dragon_kick() from act.offensive.c lines 636-690.
// Requires 10 move. Damage: level * 1.5.
func DoDragonKick(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillDragonKick) == 0 {
		return SkillResult{Success: false, MessageToCh: "What's that, idiot-san?"}
	}

	// Self-target — act.offensive.c:659-663
	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "Aren't we funny today...\r\n"}
	}

	// Mounted — act.offensive.c:664-668
	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}

	if !ch.SpendMove(10) {
		return SkillResult{Success: false, MessageToCh: "You're too exhausted!"}
	}
	// #nosec G404
	percent := ((5 - (target.GetAC() / 10)) * 2) + dprng.Number(1, 101)
	prob := ch.GetSkill(SkillDragonKick)
	// C: WAIT_STATE(ch, PULSE_VIOLENCE+2) sits outside the if/else — both
	// branches get WaitCh=3 — act.offensive.c:689.
	if percent > prob {
		return SkillResult{
			Success:      false,
			SkillMsgType: SkillDragonKickNum,
			StartCombat:  true,
			WaitCh:       3,
		}
	}
	dam := int(float64(ch.GetLevel()) * 1.5)
	return SkillResult{
		Success:         true,
		Damage:          dam,
		SkillMsgType:    SkillDragonKickNum,
		DamageSkill:     SkillDragonKick,
		StartCombat:     true,
		WaitCh:          3,
		DeferredImprove: []string{SkillDragonKick},
	}
}

// DoTigerPunch implements do_tiger_punch() from act.offensive.c lines 693-744.
// Requires bare hands. Damage: level * 2.5.
func DoTigerPunch(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillTigerPunch) == 0 {
		return SkillResult{Success: false, MessageToCh: "What's that, idiot-san?"}
	}
	if func() bool { _, ok := ch.Equipment.GetItemInSlot(SlotWield); return ok }() {
		return SkillResult{Success: false, MessageToCh: "That's pretty tough to do while wielding a weapon."}
	}
	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())
	// #nosec G404
	percent := ((7 - (target.GetAC() / 10)) * 2) + dprng.Number(1, 101)
	prob := ch.GetSkill(SkillTigerPunch)
	if percent > prob {
		return SkillResult{
			Success: false, WaitCh: 2,
			MessageToCh:   ActMessage("You snap a tiger punch at $N but miss!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n snaps a tiger punch at you but misses!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to tiger punch $N but misses!", chPronouns, &victPronouns, ""),
		}
	}
	dam := int(float64(ch.GetLevel()) * 2.5)
	improveSkill(ch, SkillTigerPunch)
	return SkillResult{
		Success: true, Damage: dam, WaitCh: 2,
		MessageToCh:   ActMessage("You snap a lightning-fast tiger punch into $N!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n snaps a lightning-fast tiger punch into you!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n tiger punches $N!", chPronouns, &victPronouns, ""),
	}
}

// DoShoot implements do_shoot() from act.offensive.c lines 746-980.
// Cannot shoot while fighting. Simplified for same-room targets.
func DoShoot(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillShoot) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}
	if ch.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "But you are already engaged in close-range combat!"}
	}
	// #nosec G404
	percent := dprng.Number(1, 101)
	prob := ch.GetSkill(SkillShoot)
	if percent >= prob {
		return SkillResult{
			Success: false, WaitCh: 1,
			MessageToCh:   "Twang... you miss!",
			MessageToVict: "Something streaks toward you but narrowly misses!",
			MessageToRoom: "A projectile narrowly misses its target!",
		}
	}
	dam := ch.GetDamroll() + dprng.Number(1, 6) + dprng.Number(1, 4)
	improveSkill(ch, SkillShoot)
	return SkillResult{
		Success: true, Damage: dam, WaitCh: 1,
		MessageToCh:   "You hear a roar of pain! Your shot hits!",
		MessageToVict: "A projectile pierces you!",
		MessageToRoom: fmt.Sprintf("%s fires a projectile that strikes %s!", ch.Name, target.GetName()),
	}
}

// DoSubdue implements do_subdue() from act.offensive.c lines 1084-1160.
// Non-lethal stun. Cannot be fighting.
func DoSubdue(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillSubdue) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how!"}
	}
	if ch.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "You're too busy right now!"}
	}
	if target.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "You can't get close enough!"}
	}
	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())
	// #nosec G404
	percent := dprng.Number(1, 101+target.GetLevel())
	prob := ch.GetSkill(SkillSubdue)
	if levelDiff := target.GetLevel() - ch.GetLevel(); levelDiff > 0 {
		percent += levelDiff
	}
	if !target.IsNPC() && (target.GetLevel() > ch.GetLevel()+3 || target.GetLevel() < ch.GetLevel()-3) {
		percent = prob + 1
	}
	if percent > prob {
		return SkillResult{
			Success: false, WaitCh: 3,
			MessageToCh:   ActMessage("$N avoids your misplaced blow to the back of $S head.", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n misses a blow to the back of your head.", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$N avoids $n's misplaced blow to the back of $S head.", chPronouns, &victPronouns, ""),
		}
	}
	improveSkill(ch, SkillSubdue)
	return SkillResult{
		Success: true, Damage: 0, StunTarget: true, WaitCh: 1, WaitTarget: 3,
		MessageToCh:   ActMessage("You knock $M out cold.", chPronouns, &victPronouns, ""),
		MessageToVict: "Someone sneaks up behind you and knocks you out!",
		MessageToRoom: ActMessage("$n knocks out $N with a well-placed blow to the back of the head.", chPronouns, &victPronouns, ""),
	}
}

// DoSleeper implements do_sleeper() from act.offensive.c lines 1184-1280.
// Requires bare hands. Non-lethal sleep.
func DoSleeper(ch *Player, target combat.Combatant, world *World) SkillResult {
	if ch.GetSkill(SkillSleeper) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}
	if ch.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "You can't do this while fighting!"}
	}
	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}
	if world != nil && world.RoomHasFlag(ch.GetRoom(), "peaceful") {
		return SkillResult{Success: false, MessageToCh: "This room just has such a peaceful, easy feeling..."}
	}
	if func() bool { _, ok := ch.Equipment.GetItemInSlot(SlotWield); return ok }() {
		return SkillResult{Success: false, MessageToCh: "You can't get a good grip on them while you are holding that weapon!"}
	}
	if target == nil {
		return SkillResult{Success: false, MessageToCh: "Sleeper who?"}
	}
	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "Can't get to sleep fast enough, huh?"}
	}
	if !target.IsNPC() && ch.GetFlags()&(1<<uint(PlrOutlaw)) == 0 {
		return SkillResult{
			Success:       false,
			MessageToCh:   "You can not sleeper them because you are not an Outlaw!",
			MessageToVict: fmt.Sprintf("%s failed to sleeper you because %s is not an Outlaw.", ch.GetName(), ch.GetName()),
		}
	}
	if target.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "You can't get a good grip on them while they're fighting!"}
	}
	if isShopKeeperInWorld(world, target) {
		return SkillResult{Success: false, MessageToCh: "Ha Ha. Don't think so."}
	}
	if target.GetPosition() <= combat.PosSleeping {
		return SkillResult{Success: false, MessageToCh: "What's the point of doing that now?"}
	}
	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())
	// #nosec G404
	percent := dprng.Number(1, 101+target.GetLevel())
	prob := ch.GetSkill(SkillSleeper)
	if mob, ok := target.(*MobInstance); ok && (mob.HasMobFlag(MobFlagAware) || mob.HasMobFlag(MobFlagNosleep)) {
		prob = 0
	}
	if levelDiff := target.GetLevel() - ch.GetLevel(); levelDiff > 0 {
		percent += levelDiff
	}
	if !target.IsNPC() && (target.GetLevel() > ch.GetLevel()+3 || target.GetLevel() < ch.GetLevel()-3) {
		prob = 0
	}
	if percent > prob {
		return SkillResult{
			Success: false, WaitCh: 2, RetaliateHit: true, RetaliateHitAfterMessages: true,
			MessageToCh:   ActMessage("You try to grab $N in a sleeper hold but fail!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to put a sleeper hold on you, but you break free!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to put $N in a sleeper hold...", chPronouns, &victPronouns, ""),
		}
	}
	return SkillResult{
		Success: true, Damage: 0, SleepTarget: true, WaitCh: 2,
		MessageToCh:         ActMessage("You put $N in a sleeper hold.", chPronouns, &victPronouns, ""),
		MessageToVict:       "You feel very sleepy... Zzzzz..",
		MessageToRoom:       ActMessage("$n puts $N in a sleeper hold.", chPronouns, &victPronouns, ""),
		MessageToRoomSecond: ActMessage("$N goes to sleep.", chPronouns, &victPronouns, ""),
		RoomIncludesTarget:  true,
		DeferredImprove:     []string{SkillSleeper}, DeferredImproveAfterRoom: true,
	}
}

// DoNeckbreak implements do_neckbreak() from act.offensive.c lines 1295-1376.
// Requires bare hands + 51 move. Damage: 18d(level).
func DoNeckbreak(ch *Player, target combat.Combatant, world *World) SkillResult {
	if ch.GetSkill(SkillNeckbreak) == 0 {
		return SkillResult{Success: false, MessageToCh: "What's that, idiot-san?"}
	}
	if func() bool { _, ok := ch.Equipment.GetItemInSlot(SlotWield); return ok }() {
		return SkillResult{Success: false, MessageToCh: "You can't do this and wield a weapon at the same time!"}
	}

	// C resolves shopkeeper protection before self, peaceful, mounted, move,
	// and RNG branches (act.offensive.c:1320-1325).
	if isShopKeeperInWorld(world, target) {
		return SkillResult{Success: false, MessageToCh: "Haha.. Don't think so."}
	}
	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "Aren't we funny today..."}
	}
	if world != nil && world.roomHasFlag(ch.GetRoom(), "peaceful") {
		return SkillResult{Success: false, MessageToCh: "You can't contemplate violence in such a place!"}
	}
	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}
	if !ch.SpendMove(51) {
		return SkillResult{Success: false, MessageToCh: "You haven't the energy to do this!"}
	}

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())
	// #nosec G404
	percent := ((7 - (target.GetAC() / 10)) * 2) + dprng.Number(1, 101)
	prob := ch.GetSkill(SkillNeckbreak)
	if percent > prob {
		return SkillResult{
			Success: false, WaitCh: 3,
			MessageToCh:   ActMessage("You try to break $S neck, but $E is too strong!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to break your neck, but can't!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to break $N's neck, but $N slips free!", chPronouns, &victPronouns, ""),
			RetaliateHit:  true,
			// C emits all three failure act() lines before hit(vict,ch).
			RetaliateHitAfterMessages: true,
		}
	}
	dam := combat.RollDice(18, ch.GetLevel())
	return SkillResult{
		Success:          true,
		Damage:           dam,
		SkillMsgType:     SkillNeckbreakNum,
		SkillMsgInDamage: true,
		DamageSkill:      SkillNeckbreak,
		StartCombat:      true,
		WaitCh:           3,
		// C improves only after damage() returns, after set-190's dice draw.
		DeferredImprove: []string{SkillNeckbreak},
	}
}

// CheckNPCDodge checks if an NPC mob dodges an attack.
// Source: fight.c:1970-1975 — number(0,100) < GET_LEVEL(ch)
func CheckNPCDodge(mob interface {
	GetLevel() int
	IsAffected(int) bool
	GetFighting() string
},
) bool {
	if mob.GetFighting() == "" || !mob.IsAffected(affDodge) {
		return false
	}
	// #nosec G404
	return dprng.Number(0, 99) < mob.GetLevel()
}
