package game

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// DoBackstab implements do_backstab() from act.offensive.c lines 165-235.
//
// Gates (mirrors DoCircle in this file, which already ports the C gates):
//  1. Skill known
//  2. Not targeting self
//  3. Piercing weapon wielded (TYPE_PIERCE — obj Values[3] == 11)
//  4. Not mounted
//  5. Target not already fighting
//  6. MOB_AWARE awake mobs notice and retaliate (start combat, no damage)
//  7. Skill roll; on miss, still initiate combat (C: damage(ch, vict, 0, SKILL_BACKSTAB))
//
// Damage: (str_app.todam + damroll + weapon_dice) * backstab_mult(level).
// Source: fight.c damage() + class.c backstab_mult().
func DoBackstab(ch *Player, target combat.Combatant, world *World) SkillResult {
	if target == nil {
		return SkillResult{Success: false, MessageToCh: "Backstab who?"}
	}

	// 1. Skill requirement
	if ch.GetSkill(SkillBackstab) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// 2. Self-check — act.offensive.c:185
	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "How can you sneak up on yourself?"}
	}

	// 3. Must wield a piercing weapon — act.offensive.c:189,194
	weapon, ok := ch.Equipment.GetItemInSlot(SlotWield)
	if !ok || weapon == nil {
		return SkillResult{Success: false, MessageToCh: "You need to wield a weapon to make it a success."}
	}
	if weapon.Prototype.Values[3] != 11 { // TYPE_PIERCE - TYPE_HIT
		return SkillResult{Success: false, MessageToCh: "Only piercing weapons can be used for backstabbing."}
	}

	// 4. Mounted check — act.offensive.c:206
	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}

	// 5. Target must not be fighting — act.offensive.c:209
	if target.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "You can't backstab a fighting person -- they're too alert!"}
	}

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	// 6. MOB_AWARE awake mobs notice the attempt and retaliate — act.offensive.c:212
	if mob, ok := target.(*MobInstance); ok && mob.HasMobFlag(MobFlagAware) && target.GetPosition() > combat.PosSleeping {
		if target.GetFighting() == "" {
			target.SetFighting(ch.Name)
		}
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("$e notices you lunging at $m!", victPronouns, &chPronouns, ""),
			MessageToVict: ActMessage("You notice $N lunging at you!", victPronouns, &chPronouns, ""),
			MessageToRoom: ActMessage("$n notices $N lunging at $m!", victPronouns, &chPronouns, ""),
			StartCombat:   true,
		}
	}

	// 7. Roll for success — act.offensive.c:220
	// #nosec G404 — game RNG, not cryptographic
	percent := rand.IntN(101) + 1 // 1-101
	skillLevel := ch.GetSkill(SkillBackstab)
	prob := skillLevel

	if target.GetPosition() > combat.PosSleeping && percent > prob {
		// Miss — C still calls damage(ch, vict, 0, SKILL_BACKSTAB), which starts
		// combat. Flag the caller to initiate via the combat engine.
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to backstab $N, but $E notices you!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to backstab you, but you notice $m in time!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to backstab $N, but fails.", chPronouns, &victPronouns, ""),
			StartCombat:   true,
			WaitCh:        1,
		}
	}

	// Skill roll passed — but a passed backstab roll means "you get to
	// attempt the backstab," NOT "you automatically hit." C then calls
	// hit(ch, vict, SKILL_BACKSTAB) (act.offensive.c:226-229), which runs the
	// full THAC0 d20 to-hit check (fight.c:1825-1830) that can still miss.
	// Go previously skipped this roll — every successful backstab landed.
	// (DP-1033)
	if !combat.CalculateHitChance(ch, target, combat.HitModifiers{}) {
		improveSkill(ch, SkillBackstab)
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to backstab $N, but miss!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to backstab you, but misses!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to backstab $N, but misses.", chPronouns, &victPronouns, ""),
			StartCombat:   true,
			WaitCh:        1,
		}
	}

	// Hit — calculate damage
	// C: dam = str_app[STRENGTH_APPLY_INDEX(ch)].todam + GET_DAMROLL(ch) + weapon_dice
	//     dam *= backstab_mult(GET_LEVEL(ch))
	weaponNum, weaponSides := ch.Equipment.GetWeaponDamage()
	weaponDam := combat.RollDice(weaponNum, weaponSides)
	dam := weaponDam + ch.GetDamroll() + ch.GetStrToDam()
	mult := combat.BackstabMult(ch.GetLevel())
	dam = int(float64(dam) * mult)

	improveSkill(ch, SkillBackstab)

	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   "Your deadly backstab strikes deep!",
		MessageToVict: ActMessage("$n sneaks up from behind and plunges a dagger into you!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n sneaks up from behind and backstabs $N!", chPronouns, &victPronouns, ""),
		WaitCh:        1, // PULSE_VIOLENCE
	}
}

// DoBash implements do_bash() from act.offensive.c lines 419-490.
// Strength-based check. On success: damage + target sits + stunned.
// On failure: user sits.
func DoBash(ch *Player, target combat.Combatant, world *World) SkillResult {
	if ch.GetSkill(SkillBash) == 0 {
		return SkillResult{Success: false, MessageToCh: "You'd better leave all the martial arts to fighters."}
	}

	// Peaceful room — act.offensive.c:435-439
	if world != nil && world.roomHasFlag(ch.GetRoom(), "peaceful") {
		return SkillResult{Success: false, MessageToCh: "This room just has such a peaceful, easy feeling...\r\n"}
	}

	// Self-target — act.offensive.c:450-454
	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "Aren't we funny today...\r\n"}
	}

	// Target must be standing or fighting — act.offensive.c:456-459
	if target.GetPosition() < combat.PosFighting {
		return SkillResult{Success: false, MessageToCh: "You can't bash someone who's sitting already!"}
	}

	// Mounted — act.offensive.c:461-465
	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}

	// Check move points
	if !ch.SpendMove(10) {
		return SkillResult{Success: false, MessageToCh: "You haven't the energy!"}
	}

	// Bash formula: percent = ((5 - (GET_AC(vict)/10)) << 1) + number(1,101)
	// prob = GET_SKILL(ch, SKILL_BASH)
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := ((5 - (target.GetAC() / 10)) * 2) + (rand.IntN(101) + 1)
	prob := ch.GetSkill(SkillBash)

	// MOB_NOBASH force-fail unless the caster is an immortal — act.offensive.c:478
	if mob, ok := target.(*MobInstance); ok && mob.HasMobFlag(MobFlagNobash) && ch.GetLevel() < LVL_IMMORT {
		percent = 101
	}
	// Sleeping-target / immortal-caster auto-success — act.offensive.c:479-480.
	// In practice this is unreachable when the target is asleep (the
	// PosFighting gate above already rejects it first, matching the C
	// original's own dead branch), but the immortal-caster half fires.
	if target.GetPosition() <= combat.PosSleeping || ch.GetLevel() >= LVL_IMMORT {
		percent = 0
	}

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	// C applies `if (!IS_NPC(ch)) WAIT_STATE(ch, PULSE_VIOLENCE*2)` unconditionally
	// after either branch; ch is always a player here, so WaitCh is always 2.
	if percent > prob {
		// Failure
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to bash $N, but miss and fall!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to bash you, but misses and falls!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to bash $N, but misses and falls!", chPronouns, &victPronouns, ""),
			SelfStumble:   true,
			WaitCh:        2, // PULSE_VIOLENCE * 2 (act.offensive.c:489)
		}
	}

	// Success — damage = (level/2)+1
	dam := (ch.GetLevel() / 2) + 1
	improveSkill(ch, SkillBash)

	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   ActMessage("You send $N flying with a powerful bash!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n sends you flying with a powerful bash!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n sends $N flying with a powerful bash!", chPronouns, &victPronouns, ""),
		TargetFalls:   true,
		StunTarget:    true,
		WaitCh:        2, // PULSE_VIOLENCE * 2 (act.offensive.c:489)
		WaitTarget:    2, // PULSE_VIOLENCE * 2 (act.offensive.c:485)
	}
}

// DoKick implements do_kick() from act.offensive.c lines 587-634.
// Simple damage: level >> 1 (level/2).
func DoKick(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillKick) == 0 {
		return SkillResult{Success: false, MessageToCh: "You'd better leave all the martial arts to fighters."}
	}

	// Self-target — act.offensive.c:610-614
	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "Aren't we funny today...\r\n"}
	}

	// Mounted — act.offensive.c:615-619
	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}

	// Formula: percent = ((7 - (GET_AC(vict)/10)) << 1) + number(1,101)
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := ((7 - (target.GetAC() / 10)) * 2) + (rand.IntN(101) + 1)
	prob := ch.GetSkill(SkillKick)

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	// C: WAIT_STATE(ch, PULSE_VIOLENCE+2) sits outside the if/else, so both
	// branches get WaitCh=3 — act.offensive.c:633.
	if percent > prob {
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to kick $N, but miss!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to kick you, but misses!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to kick $N, but misses!", chPronouns, &victPronouns, ""),
			WaitCh:        3,
		}
	}

	dam := ch.GetLevel() >> 1 // level / 2
	improveSkill(ch, SkillKick)

	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   ActMessage("You kick $N square in the chest!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n kicks you square in the chest!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n kicks $N square in the chest!", chPronouns, &victPronouns, ""),
		WaitCh:        3, // PULSE_VIOLENCE + 2
	}
}

// DoTrip implements do_trip() from new_cmds.c lines 735-815.
// Dexterity check. On success: target falls (sitting).
func DoTrip(ch *Player, target combat.Combatant, world *World) SkillResult {
	if ch.GetSkill(SkillTrip) == 0 {
		return SkillResult{Success: false, MessageToCh: "You'd better leave the sneaky stuff to the thieves."}
	}

	// Peaceful room — new_cmds.c:749-753
	if world != nil && world.roomHasFlag(ch.GetRoom(), "peaceful") {
		return SkillResult{Success: false, MessageToCh: "This room just has such a peaceful, easy feeling...\r\n"}
	}

	// Mounted — new_cmds.c:765-769
	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}

	// Self-target — new_cmds.c:771-775
	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "You trip over your shoe laces...\r\n"}
	}

	// Can't trip flying targets — new_cmds.c:776-779
	if flyer, ok := target.(interface{ IsAffected(int) bool }); ok && flyer.IsAffected(affFly) {
		return SkillResult{Success: false, MessageToCh: "You can't trip something that's FLYING!"}
	}

	if target.GetPosition() <= combat.PosSleeping {
		return SkillResult{Success: false, MessageToCh: "What's the point of doing that now?"}
	}

	// Formula: percent = number(1,121) + MAX(GET_LEVEL(vict)-GET_LEVEL(ch),0)
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := rand.IntN(121) + 1

	// Immortal-victim force-fail — new_cmds.c:782
	if target.GetLevel() >= LVL_IMMORT {
		percent = 101
	}
	// MOB_NOBASH force-fail (NPC victims only) — new_cmds.c:783
	if target.IsNPC() {
		if mob, ok := target.(*MobInstance); ok && mob.HasMobFlag(MobFlagNobash) {
			percent = 101
		}
	}

	prob := ch.GetSkill(SkillTrip)
	percent += max(target.GetLevel()-ch.GetLevel(), 0)

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	if percent > prob {
		// Failure — user falls. C: WAIT_STATE(ch, PULSE_VIOLENCE) — new_cmds.c:807
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to trip $N, but fail and fall!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to trip you, but fails and falls!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to trip $N, but fails and falls!", chPronouns, &victPronouns, ""),
			SelfStumble:   true,
			WaitCh:        1,
		}
	}

	// Success — damage = (level/2)+1, target falls. C only sets a wait state
	// on the victim (WAIT_STATE(victim, PULSE_VIOLENCE)); ch gets none —
	// new_cmds.c:811-813.
	dam := (ch.GetLevel() / 2) + 1
	improveSkill(ch, SkillTrip)

	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   ActMessage("You trip $N sending $M crashing to the ground!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n trips you sending you crashing to the ground!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n trips $N sending $M crashing to the ground!", chPronouns, &victPronouns, ""),
		TargetFalls:   true,
		WaitTarget:    1,
	}
}

// DoHeadbutt implements do_headbutt() from new_cmds.c lines 368-460.
// Damage is flat GET_LEVEL(ch) — a successful hit costs the caster HP as
// recoil (level/4, or level/3 wearing a helm), gated by an HP check that
// refuses the attempt outright if the recoil could be fatal.
func DoHeadbutt(ch *Player, target combat.Combatant, world *World) SkillResult {
	// Peaceful room — checked before even the skill gate — new_cmds.c:378-381
	if world != nil && world.roomHasFlag(ch.GetRoom(), "peaceful") {
		return SkillResult{Success: false, MessageToCh: "The Gods prevent thy violent act.\r\n"}
	}

	if ch.GetSkill(SkillHeadbutt) == 0 {
		return SkillResult{Success: false, MessageToCh: "You aren't qualified to headbutt anyone!\r\n"}
	}

	// Mounted — new_cmds.c:388-392
	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}

	// Self-target — new_cmds.c:405-408
	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "You bang your head into the nearest wall...\r\n"}
	}

	// Can't headbutt a non-NPC immortal — caster is thrown down — new_cmds.c:410-416
	if target.GetLevel() >= LVL_IMMORT && !target.IsNPC() {
		return SkillResult{
			Success:     false,
			MessageToCh: "How dare you try to headbutt a god!\r\nYou are thrown across the room...\r\n",
			SelfStumble: true,
		}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := rand.IntN(121) + 1 // 1-121; new_cmds.c:422

	// Sleeping-target / immortal-caster auto-success — new_cmds.c:424-426.
	// Note: strictly-greater-than LEVEL_IMMORT, unlike bash's >=.
	if target.GetPosition() <= combat.PosSleeping || ch.GetLevel() > LVL_IMMORT {
		percent = 0
	}
	// MOB_NOBASH auto-success (not force-fail, unlike bash/trip) — new_cmds.c:428
	if mob, ok := target.(*MobInstance); ok && mob.HasMobFlag(MobFlagNobash) {
		percent = 0
	}

	// HP gate — refuses outright if the recoil could kill the caster — new_cmds.c:429-433
	if ch.GetLevel()/2 > ch.GetHP() {
		return SkillResult{Success: false, MessageToCh: "But that could kill you!\r\n"}
	}

	skillLevel := ch.GetSkill(SkillHeadbutt)
	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	// C: WAIT_STATE(ch, PULSE_VIOLENCE*3) sits outside the hit/miss if/else —
	// both branches get WaitCh=3 — new_cmds.c:459.
	if percent > skillLevel {
		// Miss — new_cmds.c:435-437
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to headbutt $N but miss!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to headbutt you but misses!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to headbutt $N but misses!", chPronouns, &victPronouns, ""),
			WaitCh:        3,
		}
	}

	// Hit — recoil damage to caster based on helm, then flat level damage to
	// victim, victim sits (not stunned) if above POS_STUNNED — new_cmds.c:438-455.
	recoil := ch.GetLevel() / 4
	if _, wearingHelm := ch.Equipment.GetItemInSlot(SlotHead); wearingHelm {
		recoil = ch.GetLevel() / 3
	}
	ch.TakeDamage(recoil)

	improveSkill(ch, SkillHeadbutt)

	result := SkillResult{
		Success:       true,
		Damage:        ch.GetLevel(),
		MessageToCh:   ActMessage("You slam your forehead into $N with a sickening crack!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n slams $s forehead into you with a sickening crack!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n slams $s forehead into $N with a sickening crack!", chPronouns, &victPronouns, ""),
		WaitCh:        3,
	}
	if target.GetPosition() > combat.PosStunned {
		result.TargetFalls = true
	}
	return result
}

// DoRescue implements do_rescue() from act.offensive.c lines 499-581.
// Interposes between attacker and target.
func DoRescue(ch *Player, target combat.Combatant, world *World, combatEngine interface {
	StartCombat(combat.Combatant, combat.Combatant) error
	StopCombat(string)
},
) SkillResult {
	if ch.GetSkill(SkillRescue) == 0 {
		return SkillResult{Success: false, MessageToCh: "But only true warriors can do this!"}
	}

	// Can't rescue yourself
	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "What about fleeing instead?"}
	}

	// Can't rescue someone you're fighting
	if ch.GetFighting() == target.GetName() {
		return SkillResult{Success: false, MessageToCh: "How can you rescue someone you are trying to kill?"}
	}

	// Mounted — act.offensive.c:531-535
	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!"}
	}

	// Peaceful room — outlaws are exempt (they can fight anywhere) — act.offensive.c:537-542
	if world != nil && world.roomHasFlag(ch.GetRoom(), "peaceful") && ch.GetFlags()&(1<<PlrOutlaw) == 0 {
		return SkillResult{Success: false, MessageToCh: "This room just has such a peaceful, easy feeling...\r\n"}
	}

	// Find who is fighting the target
	var attacker combat.Combatant
	// Check players
	players := world.GetPlayersInRoom(ch.GetRoom())
	for _, p := range players {
		if p.GetFighting() == target.GetName() && p.Name != ch.Name {
			attacker = p
			break
		}
	}
	// Check mobs
	if attacker == nil {
		mobs := world.GetMobsInRoom(ch.GetRoom())
		for _, m := range mobs {
			if m.GetFighting() == target.GetName() {
				attacker = m
				break
			}
		}
	}

	if attacker == nil {
		victPronouns := GetPronouns(target.GetName(), target.GetSex())
		return SkillResult{Success: false, MessageToCh: ActMessage("But nobody is fighting $N!", GetPronouns(ch.Name, ch.GetSex()), &victPronouns, "")}
	}

	// Roll for success
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := rand.IntN(101) + 1
	prob := ch.GetSkill(SkillRescue)

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	if percent > prob {
		// C sets no wait state at all on a failed rescue.
		return SkillResult{
			Success:     false,
			MessageToCh: "You fail the rescue!",
		}
	}

	// Success — stop fighting for all three parties, then interpose ch between
	// attacker and target: attacker turns onto ch, ch turns onto attacker.
	// act.offensive.c:569-579.
	improveSkill(ch, SkillRescue)

	combatEngine.StopCombat(target.GetName())
	combatEngine.StopCombat(attacker.GetName())
	combatEngine.StopCombat(ch.Name)
	// Errors here mean one side is already paired (shouldn't happen right
	// after the StopCombat calls above); nothing more we can do but proceed.
	_ = combatEngine.StartCombat(ch, attacker)

	return SkillResult{
		Success:       true,
		MessageToCh:   "Banzai!  To the rescue...",
		MessageToVict: ActMessage("You are rescued by $N, you are confused!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n heroically rescues $N!", chPronouns, &victPronouns, ""),
		WaitTarget:    2, // WAIT_STATE(vict, 2*PULSE_VIOLENCE) — act.offensive.c:579; no WaitCh in C
	}
}

// ---------------------------------------------------------------------------
// Sneak / Hide / Steal state
// ---------------------------------------------------------------------------
// Sneak and hide state are stored via Player.Affects bit vector using
// affSneak (0) and affHide (1) constants from act_movement.go.
// Player.mu protects all access. No global maps needed.

// DoSneak implements do_sneak() from act.other.c lines 214-245.

// DoSpike implements do_spike() from src/new_cmds.c lines 1098-1188.
// subcmd 0 = spike (werewolf), 1 = stake (vampire).
func DoSpike(ch *Player, target combat.Combatant, subcmd int, world *World) SkillResult {
	weaponName := "spike"
	if subcmd == 1 {
		weaponName = "stake"
	}

	if ch.GetFighting() != "" {
		return SkillResult{
			Success:     false,
			MessageToCh: fmt.Sprintf("You can't %s someone while fighting!\r\n", weaponName),
		}
	}

	if target == nil {
		return SkillResult{
			Success:     false,
			MessageToCh: fmt.Sprintf("Whom do you wish to %s?\r\n", weaponName),
		}
	}

	weapon, ok := ch.Equipment.GetItemInSlot(SlotWield)
	if !ok || weapon == nil || !strings.Contains(strings.ToLower(weapon.GetShortDesc()), weaponName) {
		return SkillResult{
			Success:     false,
			MessageToCh: fmt.Sprintf("You need to wield a %s to succeed!\r\n", weaponName),
		}
	}

	if world != nil && world.roomHasFlag(ch.GetRoom(), "peaceful") {
		return SkillResult{Success: false, MessageToCh: "You can't commit murder in this holy place!\r\n"}
	}

	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "The monster in you won't let you suicide!\r\n"}
	}

	// Targets must expose the Player interface to check affects/PLR flags.
	tp, ok := target.(*Player)
	if !ok {
		return SkillResult{Success: false, MessageToCh: fmt.Sprintf("You can't %s that!\r\n", weaponName)}
	}

	if subcmd == 0 && !tp.IsAffected(affWerewolf) {
		return SkillResult{Success: false, MessageToCh: "Spiking is only for werewolves..\r\n"}
	}
	if subcmd == 1 && !tp.IsAffected(affVampire) {
		return SkillResult{Success: false, MessageToCh: "Staking is only for vampires..\r\n"}
	}

	if subcmd == 0 && ch.GetFlags()&(1<<PlrWerewolf) != 0 {
		return SkillResult{Success: false, MessageToCh: "You can't destroy your own kind!\r\n"}
	}
	if subcmd == 1 && ch.GetFlags()&(1<<PlrVampire) != 0 {
		return SkillResult{Success: false, MessageToCh: "You can't destroy your own kind!\r\n"}
	}

	if tp.GetLevel() >= LVL_IMMORT && ch.GetLevel() < LVL_IMMORT {
		return SkillResult{Success: false, MessageToCh: "Yeah, right.\r\n"}
	}

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(tp.GetName(), tp.GetSex())

	// Success if attacker level > victim, level gap < random(0, LVL_IMMORT), or victim asleep.
	// #nosec G404 — game RNG, not cryptographic
	if ch.GetLevel() > tp.GetLevel() ||
		tp.GetLevel()-ch.GetLevel() < rand.IntN(LVL_IMMORT) ||
		tp.GetPosition() <= combat.PosSleeping {
		// Remove vampire/werewolf PLR flag from the victim so raw_kill can proceed.
		if tp.GetFlags()&(1<<PlrVampire) != 0 {
			tp.SetPlrFlag(PlrVampire, false)
		}
		if tp.GetFlags()&(1<<PlrWerewolf) != 0 {
			tp.SetPlrFlag(PlrWerewolf, false)
		}
		// Note: C increments GET_PKS/GET_DEATHS and calls raw_kill. We rely on
		// RawKill and the existing kill-tracking callbacks.
		combat.RawKill(target, combat.TYPE_UNDEFINED)

		return SkillResult{
			Success:       true,
			MessageToCh:   ActMessage("You drive $p into $S chest!", chPronouns, &victPronouns, weapon.GetShortDesc()),
			MessageToVict: ActMessage("$n drives $p into your chest with a solid blow!", chPronouns, &victPronouns, weapon.GetShortDesc()),
			MessageToRoom: ActMessage("$n drives $p into the chest of $N!", chPronouns, &victPronouns, weapon.GetShortDesc()),
			WaitCh:        2,
		}
	}

	return SkillResult{
		Success:       false,
		MessageToCh:   ActMessage("$N twists at the last moment, and you miss!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n comes at you with a $p, but you dodge the attempt!", chPronouns, &victPronouns, weapon.GetShortDesc()),
		MessageToRoom: ActMessage("$N growls in anger as $n tries to drive a $p into $M!", chPronouns, &victPronouns, weapon.GetShortDesc()),
		WaitCh:        2,
	}
}

// DoCircle implements do_circle() from src/new_cmds.c lines 2391-2467.
// Thief-style backstab variant usable in combat. Requires a piercing weapon.
func DoCircle(ch *Player, target combat.Combatant) SkillResult {
	if target == nil {
		return SkillResult{Success: false, MessageToCh: "Circle who?\r\n"}
	}

	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "How can you stab yourself in the back?\r\n"}
	}

	// Already fighting someone who is fighting you back — too busy.
	if ch.GetFighting() != "" && ch.GetFighting() == target.GetName() && target.GetFighting() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "You're a little too busy right now!\r\n"}
	}

	weapon, ok := ch.Equipment.GetItemInSlot(SlotWield)
	if !ok || weapon == nil {
		return SkillResult{Success: false, MessageToCh: "You need to wield a weapon to make it a success.\r\n"}
	}
	if weapon.Prototype.Values[3] != 11 { // TYPE_PIERCE - TYPE_HIT
		return SkillResult{Success: false, MessageToCh: "Only piercing weapons can be used for backstabbing.\r\n"}
	}

	if ch.IsMounted() {
		return SkillResult{Success: false, MessageToCh: "Dismount first!\r\n"}
	}

	// MOB_AWARE mobs that are awake notice the attempt and retaliate.
	// C: new_cmds.c:2427 — hit(vict, ch) if the mob isn't already fighting.
	// Go has no instant-hit primitive here, so we redirect the mob onto the
	// circler and set StartCombat; the combat engine's next PerformRound
	// (bidirectional since DP-900) delivers the mob's retaliatory swing.
	if mob, ok := target.(*MobInstance); ok && mob.HasMobFlag(MobFlagAware) && target.GetPosition() > combat.PosSleeping {
		victPronouns := GetPronouns(target.GetName(), target.GetSex())
		chPronouns := GetPronouns(ch.Name, ch.GetSex())
		target.SetFighting(ch.Name)
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("$e notices you lunging at $m!", victPronouns, &chPronouns, ""),
			MessageToVict: ActMessage("You notice $N lunging at you!", victPronouns, &chPronouns, ""),
			MessageToRoom: ActMessage("$n notices $N lunging at $m!", victPronouns, &chPronouns, ""),
			StartCombat:   true,
		}
	}

	if ch.GetSkill(SkillCircle) <= 0 {
		return SkillResult{Success: false, MessageToCh: "You make a circle in the air.\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := rand.IntN(101) + 1 // 1-101; 101% is automatic failure
	prob := ch.GetSkill(SkillCircle)

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	if target.GetPosition() > combat.PosSleeping && percent > prob {
		// Miss. C: new_cmds.c:2457 — if the target is fighting, stop_fighting
		// + hit(vict, ch): the botched circle pulls the mob's aggro onto the
		// circler. Then damage(ch, vict, 0, SKILL_CIRCLE) starts combat for the
		// circler too. We retarget the mob onto the circler and set StartCombat
		// so the caller enrolls the circler; the engine delivers the mob's
		// swing next round. (C also deals the mob's hit instantly; the engine's
		// tick delivers it within one PULSE_VIOLENCE — acceptable fidelity
		// trade consistent with the rest of the Go combat engine.)
		target.SetFighting(ch.Name)
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to circle $N, but $E notices you!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to circle you, but you notice $m in time!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to circle $N, but fails.", chPronouns, &victPronouns, ""),
			StartCombat:   true,
			WaitCh:        3, // PULSE_VIOLENCE + 2
		}
	}

	// Hit — weapon damage + damroll + str-to-dam, multiplied by
	// backstab_mult(level)/3. C: hit() → damage() includes str_app.todam;
	// this was missing before (DP-906 added GetStrToDam for backstab).
	weaponNum, weaponSides := ch.Equipment.GetWeaponDamage()
	weaponDam := combat.RollDice(weaponNum, weaponSides)
	dam := weaponDam + ch.GetDamroll() + ch.GetStrToDam()
	mult := int(combat.BackstabMult(ch.GetLevel())) / 3
	if mult < 1 {
		mult = 1
	}
	dam *= mult

	improveSkill(ch, SkillCircle)

	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   ActMessage("You circle around $N and plunge your weapon into $S back!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n circles around you and plunges $s weapon into your back!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n circles around $N and plunges $s weapon into $S back!", chPronouns, &victPronouns, ""),
		WaitCh:        3, // PULSE_VIOLENCE + 2
	}
}

// DoCharge implements do_charge() from src/new_cmds.c lines 880-955.
// Warrior/paladin/ranger charge attack requiring a sword or lance.
func DoCharge(ch *Player, target combat.Combatant) SkillResult {
	if target == nil {
		return SkillResult{Success: false, MessageToCh: "Great! Fine! Charge who?!?!\r\n"}
	}

	if ch.GetSkill(SkillCharge) == 0 {
		return SkillResult{Success: false, MessageToCh: "You couldn't charge if you wanted to!\r\n"}
	}

	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "You charge headlong into the ground, impressing everyone..\r\n"}
	}

	weapon, ok := ch.Equipment.GetItemInSlot(SlotWield)
	if !ok || weapon == nil {
		return SkillResult{Success: false, MessageToCh: "You're barehanded, try it with a sword or lance next time.\r\n"}
	}

	wpnType := weapon.Prototype.Values[3]
	if wpnType != 3 && wpnType != 12 { // sword (TYPE_SLASH) or lance
		return SkillResult{Success: false, MessageToCh: "You need sword or a lance to run 'em through!\r\n"}
	}

	// Route through combat.Roller so tests can seed/script the success roll
	// (math/rand/v2's global source cannot be seeded). #nosec G404 — game RNG.
	percent := ((5 - (target.GetAC() / 10)) * 2) + (combat.GetRoller().IntN(101) + 1)
	if ch.IsMounted() {
		percent += 5
	}
	if mob, ok := target.(*MobInstance); ok && mob.HasMobFlag(MobFlagNobash) {
		percent += 25
	}

	prob := ch.GetSkill(SkillCharge)

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	if percent > prob {
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You charge at $N, but lose your balance and fall!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n charges at you, but loses $s balance and falls!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n charges at $N, but loses $s balance and falls!", chPronouns, &victPronouns, ""),
			SelfStumble:   !ch.IsMounted(),
			WaitCh:        2, // PULSE_VIOLENCE * 2
		}
	}

	weaponNum, weaponSides := ch.Equipment.GetWeaponDamage()
	dam := 2 * combat.RollDice(weaponNum, weaponSides)
	if ch.IsMounted() {
		dam += 50
	}

	improveSkill(ch, SkillCharge)

	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   ActMessage("You charge at $N and run $M through with your weapon!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n charges at you and runs you through with $s weapon!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n charges at $N and runs $M through with $s weapon!", chPronouns, &victPronouns, ""),
		WaitCh:        2, // PULSE_VIOLENCE * 2
	}
}
