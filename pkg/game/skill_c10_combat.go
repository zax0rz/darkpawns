package game

import (
	"fmt"
	"math/rand"
	"github.com/zax0rz/darkpawns/pkg/combat"
)

func DoDisembowel(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillDisembowel) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}
	wielded, _ := ch.Equipment.GetItemInSlot(SlotWield)
	if wielded == nil || wielded.Prototype == nil {
		return SkillResult{Success: false, MessageToCh: "You need to wield a weapon to make it a success."}
	}
	if wielded.Prototype.Values[3] != 11 { // TYPE_PIERCE
		return SkillResult{Success: false, MessageToCh: "Only piercing weapons can be used for disemboweling."}
	}
	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())
	// #nosec G404 — game RNG
	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillDisembowel)
	if target.GetPosition() > combat.PosSleeping && percent > prob {
		return SkillResult{
			Success: false, WaitCh: 2,
			MessageToCh: ActMessage("You try to disembowel $N, but $E dodges!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to disembowel you, but misses!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to disembowel $N, but fails!", chPronouns, &victPronouns, ""),
		}
	}
	dam := ch.Level*2 + ch.GetDamroll()
	improveSkill(ch, SkillDisembowel)
	return SkillResult{
		Success: true, Damage: dam, WaitCh: 2,
		MessageToCh: ActMessage("You drive your blade deep into $N's gut!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n drives $s blade deep into your gut!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n disembowels $N in a shower of gore!", chPronouns, &victPronouns, ""),
	}
}

// DoDragonKick implements do_dragon_kick() from act.offensive.c lines 636-690.
// Requires 10 move. Damage: level * 1.5.
func DoDragonKick(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillDragonKick) == 0 {
		return SkillResult{Success: false, MessageToCh: "What's that, idiot-san?"}
	}
	if ch.Move < 10 {
		return SkillResult{Success: false, MessageToCh: "You're too exhausted!"}
	}
	ch.Move -= 10
	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())
	// #nosec G404
	percent := ((5 - (target.GetAC()/10))*2) + (rand.Intn(101) + 1)
	prob := ch.GetSkill(SkillDragonKick)
	if percent > prob {
		return SkillResult{
			Success: false, WaitCh: 1,
			MessageToCh: ActMessage("You attempt a dragon kick on $N but miss!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n attempts a dragon kick on you but misses!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n attempts a dragon kick on $N but misses!", chPronouns, &victPronouns, ""),
		}
	}
	dam := int(float64(ch.Level) * 1.5)
	improveSkill(ch, SkillDragonKick)
	return SkillResult{
		Success: true, Damage: dam, WaitCh: 1,
		MessageToCh: ActMessage("You unleash a devastating dragon kick against $N!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n unleashes a devastating dragon kick against you!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n dragon kicks $N!", chPronouns, &victPronouns, ""),
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
	percent := ((7 - (target.GetAC()/10))*2) + (rand.Intn(101) + 1)
	prob := ch.GetSkill(SkillTigerPunch)
	if percent > prob {
		return SkillResult{
			Success: false, WaitCh: 2,
			MessageToCh: ActMessage("You snap a tiger punch at $N but miss!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n snaps a tiger punch at you but misses!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to tiger punch $N but misses!", chPronouns, &victPronouns, ""),
		}
	}
	dam := int(float64(ch.Level) * 2.5)
	improveSkill(ch, SkillTigerPunch)
	return SkillResult{
		Success: true, Damage: dam, WaitCh: 2,
		MessageToCh: ActMessage("You snap a lightning-fast tiger punch into $N!", chPronouns, &victPronouns, ""),
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
	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillShoot)
	if percent >= prob {
		return SkillResult{
			Success: false, WaitCh: 1,
			MessageToCh: "Twang... you miss!",
			MessageToVict: "Something streaks toward you but narrowly misses!",
			MessageToRoom: "A projectile narrowly misses its target!",
		}
	}
	dam := ch.GetDamroll() + rand.Intn(6) + 1 + rand.Intn(4) + 1
	improveSkill(ch, SkillShoot)
	return SkillResult{
		Success: true, Damage: dam, WaitCh: 1,
		MessageToCh: "You hear a roar of pain! Your shot hits!",
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
	percent := rand.Intn(101+target.GetLevel()) + 1
	prob := ch.GetSkill(SkillSubdue)
	if levelDiff := target.GetLevel() - ch.Level; levelDiff > 0 {
		percent += levelDiff
	}
	if !target.IsNPC() && (target.GetLevel() > ch.Level+3 || target.GetLevel() < ch.Level-3) {
		percent = prob + 1
	}
	if percent > prob {
		return SkillResult{
			Success: false, WaitCh: 3,
			MessageToCh: ActMessage("$N avoids your misplaced blow to the back of $S head.", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n misses a blow to the back of your head.", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$N avoids $n's misplaced blow to the back of $S head.", chPronouns, &victPronouns, ""),
		}
	}
	improveSkill(ch, SkillSubdue)
	return SkillResult{
		Success: true, Damage: 0, StunTarget: true, WaitCh: 1, WaitTarget: 3,
		MessageToCh: ActMessage("You knock $M out cold.", chPronouns, &victPronouns, ""),
		MessageToVict: "Someone sneaks up behind you and knocks you out!",
		MessageToRoom: ActMessage("$n knocks out $N with a well-placed blow to the back of the head.", chPronouns, &victPronouns, ""),
	}
}

// DoSleeper implements do_sleeper() from act.offensive.c lines 1184-1280.
// Requires bare hands. Non-lethal sleep.
func DoSleeper(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillSleeper) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}
	if ch.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "You can't do this while fighting!"}
	}
	if func() bool { _, ok := ch.Equipment.GetItemInSlot(SlotWield); return ok }() {
		return SkillResult{Success: false, MessageToCh: "You can't get a good grip on them while holding that weapon!"}
	}
	if target.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "You can't get a good grip on them while they're fighting!"}
	}
	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())
	// #nosec G404
	percent := rand.Intn(101+target.GetLevel()) + 1
	prob := ch.GetSkill(SkillSleeper)
	if levelDiff := target.GetLevel() - ch.Level; levelDiff > 0 {
		percent += levelDiff
	}
	if !target.IsNPC() && (target.GetLevel() > ch.Level+3 || target.GetLevel() < ch.Level-3) {
		percent = prob + 1
	}
	if percent > prob {
		return SkillResult{
			Success: false, WaitCh: 2,
			MessageToCh: ActMessage("You try to grab $N in a sleeper hold but fail!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to put a sleeper hold on you, but you break free!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to put $N in a sleeper hold...", chPronouns, &victPronouns, ""),
		}
	}
	improveSkill(ch, SkillSleeper)
	return SkillResult{
		Success: true, Damage: 0, StunTarget: true, WaitCh: 2,
		MessageToCh: ActMessage("You put $N in a sleeper hold.", chPronouns, &victPronouns, ""),
		MessageToVict: "You feel very sleepy... Zzzzz..",
		MessageToRoom: ActMessage("$n puts $N in a sleeper hold. $N goes to sleep.", chPronouns, &victPronouns, ""),
	}
}

// DoNeckbreak implements do_neckbreak() from act.offensive.c lines 1295-1360.
// Requires bare hands + 51 move. Damage: 18d(level).
func DoNeckbreak(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillNeckbreak) == 0 {
		return SkillResult{Success: false, MessageToCh: "What's that, idiot-san?"}
	}
	if func() bool { _, ok := ch.Equipment.GetItemInSlot(SlotWield); return ok }() {
		return SkillResult{Success: false, MessageToCh: "You can't do this and wield a weapon at the same time!"}
	}
	if ch.Move < 51 {
		return SkillResult{Success: false, MessageToCh: "You haven't the energy to do this!"}
	}
	ch.Move -= 51
	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())
	// #nosec G404
	percent := ((7 - (target.GetAC()/10))*2) + (rand.Intn(101) + 1)
	prob := ch.GetSkill(SkillNeckbreak)
	if percent > prob {
		return SkillResult{
			Success: false, WaitCh: 3,
			MessageToCh: ActMessage("You try to break $S neck, but $E is too strong!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to break your neck, but can't!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to break $N's neck, but $N slips free!", chPronouns, &victPronouns, ""),
		}
	}
	dam := combat.RollDice(18, ch.Level)
	improveSkill(ch, SkillNeckbreak)
	return SkillResult{
		Success: true, Damage: dam, WaitCh: 3,
		MessageToCh: ActMessage("You snap $N's neck with a sickening crack!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n snaps your neck with a sickening crack!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n breaks $N's neck!", chPronouns, &victPronouns, ""),
	}
}

// DoAmbush implements do_ambush() from act.offensive.c lines 1454-1550.
// Cannot ambush target already fighting. Damage: damroll + weapon + level*2.6 + 10% if hidden.
func DoAmbush(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillAmbush) == 0 {
		return SkillResult{Success: false, MessageToCh: "You'd better not."}
	}
	if target.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "They're too alert for that, currently."}
	}
	ch.SendMessage("You crouch in the shadows and plan your ambush...\r\n")
	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())
	// #nosec G404
	percent := rand.Intn(131) + 1
	prob := ch.GetSkill(SkillAmbush)
	if percent > prob {
		return SkillResult{
			Success: false, WaitCh: 1,
			MessageToCh: ActMessage("You spring from the shadows but $N avoids your ambush!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n springs from the shadows but you dodge the ambush!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n springs from the shadows but fails to ambush $N!", chPronouns, &victPronouns, ""),
		}
	}
	dam := ch.GetDamroll()
	if weaponNum, weaponSides := ch.Equipment.GetWeaponDamage(); weaponNum > 0 && weaponSides > 0 {
		dam += combat.RollDice(weaponNum, weaponSides)
	}
	dam += int(float64(ch.Level) * 2.6)
	if ch.IsAffected(affHide) {
		dam += int(float64(dam) * 0.10)
	}
	improveSkill(ch, SkillAmbush)
	return SkillResult{
		Success: true, Damage: dam, WaitCh: 1, WaitTarget: 1,
		MessageToCh: ActMessage("You spring from the shadows and ambush $N!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n leaps from the shadows and ambushes you!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n leaps from the shadows to ambush $N!", chPronouns, &victPronouns, ""),
	}
}

// ---------------------------------------------------------------------------
// C-11: Parry/Dodge system — fight.c:1958-1975
// ---------------------------------------------------------------------------

// DoParry toggles parry stance on/off.
func DoParry(ch *Player) SkillResult {
	if ch.GetSkill(SkillParry) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}
	if ch.IsParrying() {
		ch.SetParry(false)
		return SkillResult{Success: true, MessageToCh: "You lower your defensive stance.\r\n"}
	}
	ch.SetParry(true)
	return SkillResult{Success: true, MessageToCh: "You move into a defensive stance, ready to parry incoming attacks.\r\n"}
}

// CheckParry checks if a defender parries an incoming attack.
// Source: fight.c:1958-1968 — number(0,10000) <= GET_SKILL(ch, SKILL_PARRY)
func CheckParry(defender *Player) bool {
	if !defender.IsParrying() || defender.GetFighting() == "" {
		return false
	}
	skill := defender.GetSkill(SkillParry)
	if skill <= 0 {
		return false
	}
	// #nosec G404 — game RNG; skill 0-100 scaled to 0-10000
	return rand.Intn(10001) <= skill*100
}

// CheckNPCDodge checks if an NPC mob dodges an attack.
// Source: fight.c:1970-1975 — number(0,100) < GET_LEVEL(ch)
func CheckNPCDodge(mob interface{ GetLevel() int; IsAffected(int) bool; GetFighting() string }) bool {
	if mob.GetFighting() == "" || !mob.IsAffected(affDodge) {
		return false
	}
	// #nosec G404
	return rand.Intn(100) < mob.GetLevel()
}
