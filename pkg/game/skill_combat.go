package game

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func DoBackstab(ch *Player, target combat.Combatant, world *World) SkillResult {
	// Check skill requirement
	if ch.GetSkill(SkillBackstab) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// Must wield a weapon
	weaponNum, weaponSides := ch.Equipment.GetWeaponDamage()
	if weaponNum <= 0 || weaponSides <= 0 {
		return SkillResult{Success: false, MessageToCh: "You need to wield a weapon to make it a success."}
	}

	// Target must not be fighting
	if target.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "You can't backstab a fighting person -- they're too alert!"}
	}

	// Roll for success
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := rand.IntN(101) + 1 // 1-101
	skillLevel := ch.GetSkill(SkillBackstab)
	prob := skillLevel
	if prob == 0 {
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		prob = rand.IntN(51) + 50 // 50-100 fallback
	}

	chPronouns := GetPronouns(ch.Name, ch.GetSex()) // default male for now
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	if target.GetPosition() > combat.PosSleeping && percent > prob {
		// Miss
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to backstab $N, but $E notices you!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to backstab you, but you notice $m in time!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to backstab $N, but fails.", chPronouns, &victPronouns, ""),
		}
	}

	// Hit — calculate damage
	// Source: fight.c + backstab_mult() from class.c
	// C: dam = str_app[...].todam + GET_DAMROLL(ch) + weapon_dice
	//     dam *= backstab_mult(GET_LEVEL(ch))
	weaponDam := combat.RollDice(weaponNum, weaponSides)
	dam := weaponDam + ch.GetDamroll()
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

// DoBash implements do_bash() from act.offensive.c lines 423-478.
// Strength-based check. On success: damage + target sits + stunned.
// On failure: user sits.
func DoBash(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillBash) == 0 {
		return SkillResult{Success: false, MessageToCh: "You'd better leave all the martial arts to fighters."}
	}

	// Target must be standing or fighting
	if target.GetPosition() < combat.PosFighting {
		return SkillResult{Success: false, MessageToCh: "You can't bash someone who's sitting already!"}
	}

	// Check move points
	if ch.GetMove() < 10 {
		return SkillResult{Success: false, MessageToCh: "You haven't the energy!"}
	}
	ch.SetMove(ch.GetMove() - 10)

	// Bash formula: percent = ((5 - (GET_AC(vict)/10)) << 1) + number(1,101)
	// prob = GET_SKILL(ch, SKILL_BASH)
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := ((5 - (target.GetAC() / 10)) * 2) + (rand.IntN(101) + 1)
	prob := ch.GetSkill(SkillBash)

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	if percent > prob {
		// Failure
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to bash $N, but miss and fall!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to bash you, but misses and falls!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to bash $N, but misses and falls!", chPronouns, &victPronouns, ""),
			SelfStumble:   true,
			WaitCh:        1, // PULSE_VIOLENCE
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
		WaitCh:        2, // PULSE_VIOLENCE * 2 (heavy move)
		WaitTarget:    2,
	}
}

// DoKick implements do_kick() from act.offensive.c lines 541-576.
// Simple damage: level >> 1 (level/2).
func DoKick(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillKick) == 0 {
		return SkillResult{Success: false, MessageToCh: "You'd better leave all the martial arts to fighters."}
	}

	// Formula: percent = ((7 - (GET_AC(vict)/10)) << 1) + number(1,101)
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := ((7 - (target.GetAC() / 10)) * 2) + (rand.IntN(101) + 1)
	prob := ch.GetSkill(SkillKick)

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	if percent > prob {
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to kick $N, but miss!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to kick you, but misses!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to kick $N, but misses!", chPronouns, &victPronouns, ""),
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
		WaitCh:        1, // PULSE_VIOLENCE
	}
}

// DoTrip implements do_trip() from new_cmds.c lines 728-792.
// Dexterity check. On success: target falls (sitting).
func DoTrip(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillTrip) == 0 {
		return SkillResult{Success: false, MessageToCh: "You'd better leave the sneaky stuff to the thieves."}
	}

	// Can't trip flying targets
	// (In original: IS_AFFECTED(vict, AFF_FLY) — we don't have affects yet, skip)

	if target.GetPosition() <= combat.PosSleeping {
		return SkillResult{Success: false, MessageToCh: "What's the point of doing that now?"}
	}

	// Formula: percent = number(1,121) + MAX(GET_LEVEL(vict)-GET_LEVEL(ch),0)
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := rand.IntN(121) + 1
	percent += max(target.GetLevel()-ch.GetLevel(), 0)
	prob := ch.GetSkill(SkillTrip)

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	if percent > prob {
		// Failure — user falls
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to trip $N, but fail and fall!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to trip you, but fails and falls!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to trip $N, but fails and falls!", chPronouns, &victPronouns, ""),
			SelfStumble:   true,
			WaitCh:        1,
		}
	}

	// Success — damage = (level/2)+1, target falls
	dam := (ch.GetLevel() / 2) + 1
	improveSkill(ch, SkillTrip)

	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   ActMessage("You trip $N sending $M crashing to the ground!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n trips you sending you crashing to the ground!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n trips $N sending $M crashing to the ground!", chPronouns, &victPronouns, ""),
		TargetFalls:   true,
		WaitCh:        1,
	}
}

// DoHeadbutt implements headbutt — high damage melee with self-stun risk.
// Formula: hitroll = DAMAGE_ROLL(skill_level) - 10, damage = DAMAGE_ROLL(skill_level) + 4.
// On miss: 25% chance attacker takes half damage and is stunned 1 round.
func DoHeadbutt(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillHeadbutt) == 0 {
		return SkillResult{Success: false, MessageToCh: "You'd better leave all the martial arts to fighters."}
	}

	if target.GetPosition() <= combat.PosSleeping {
		return SkillResult{Success: false, MessageToCh: "What's the point of doing that now?"}
	}

	// Check move points
	if ch.GetMove() < 15 {
		return SkillResult{Success: false, MessageToCh: "You haven't the energy!"}
	}
	ch.SetMove(ch.GetMove() - 15)

	skillLevel := ch.GetSkill(SkillHeadbutt)
	damage := (skillLevel/2 + 1) + 4 // higher base damage

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := rand.IntN(101) + 1

	chPronouns := GetPronouns(ch.Name, ch.GetSex())
	victPronouns := GetPronouns(target.GetName(), target.GetSex())

	if percent > skillLevel {
		// Miss
		result := SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to headbutt $N but miss!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to headbutt you but misses!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to headbutt $N but misses!", chPronouns, &victPronouns, ""),
			WaitCh:        1,
		}
		// 25% self-stun on failure
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		if rand.IntN(4) == 0 {
			selfDam := damage / 2
			if selfDam < 1 {
				selfDam = 1
			}
			ch.TakeDamage(selfDam)
			result.SelfStumble = true
			result.MessageToCh += " You crack your skull against thin air and see stars!\r\n"
		}
		return result
	}

	// Hit — success
	improveSkill(ch, SkillHeadbutt)

	return SkillResult{
		Success:       true,
		Damage:        damage,
		MessageToCh:   ActMessage("You slam your forehead into $N with a sickening crack!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n slams $s forehead into you with a sickening crack!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n slams $s forehead into $N with a sickening crack!", chPronouns, &victPronouns, ""),
		StunTarget:    true,
		WaitCh:        2,
	}
}

// DoRescue implements do_rescue() from act.offensive.c lines 480-539.
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
		return SkillResult{
			Success:     false,
			MessageToCh: "You fail the rescue!",
		}
	}

	// Success — stop fighting for all, start ch vs attacker
	improveSkill(ch, SkillRescue)

	return SkillResult{
		Success:       true,
		MessageToCh:   "Banzai!  To the rescue...",
		MessageToVict: ActMessage("You are rescued by $N, you are confused!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n heroically rescues $N!", chPronouns, &victPronouns, ""),
		WaitCh:        1,
		WaitTarget:    2,
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
	if mob, ok := target.(*MobInstance); ok && mob.HasMobFlag(MobFlagAware) && target.GetPosition() > combat.PosSleeping {
		victPronouns := GetPronouns(target.GetName(), target.GetSex())
		chPronouns := GetPronouns(ch.Name, ch.GetSex())
		if target.GetFighting() == "" {
			target.SetFighting(ch.Name)
		}
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("$e notices you lunging at $m!", victPronouns, &chPronouns, ""),
			MessageToVict: ActMessage("You notice $N lunging at you!", victPronouns, &chPronouns, ""),
			MessageToRoom: ActMessage("$n notices $N lunging at $m!", victPronouns, &chPronouns, ""),
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
		// Miss. If the target is fighting, they stop and turn on the attacker.
		if target.GetFighting() != "" {
			target.SetFighting("")
			target.SetFighting(ch.Name)
		}
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to circle $N, but $E notices you!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to circle you, but you notice $m in time!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to circle $N, but fails.", chPronouns, &victPronouns, ""),
			WaitCh:        3, // PULSE_VIOLENCE + 2
		}
	}

	// Hit — weapon damage + damroll, multiplied by backstab_mult(level)/3.
	weaponNum, weaponSides := ch.Equipment.GetWeaponDamage()
	weaponDam := combat.RollDice(weaponNum, weaponSides)
	dam := weaponDam + ch.GetDamroll()
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

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	percent := ((5 - (target.GetAC() / 10)) * 2) + (rand.IntN(101) + 1)
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
