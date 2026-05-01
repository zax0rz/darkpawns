package game

import (
	"math/rand"
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
	percent := rand.Intn(101) + 1 // 1-101
	skillLevel := ch.GetSkill(SkillBackstab)
	prob := skillLevel
	if prob == 0 {
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		prob = rand.Intn(51) + 50 // 50-100 fallback
	}

	chPronouns := GetPronouns(ch.Name, 1) // default male for now
	victPronouns := GetPronouns(target.GetName(), 1)

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
	weaponDam := combat.RollDice(weaponNum, weaponSides)
	mult := backstabMult(ch.Level)
	dam := int(float64(weaponDam) * mult)

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

// backstabMult mirrors backstab_mult() from class.c lines 720-729.
func backstabMult(level int) float64 {
	if level <= 0 {
		return 1.0
	}
	if level >= 31 {
		return 20.0
	}
	return float64(level)*0.2 + 1.0
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
	if ch.Move < 10 {
		return SkillResult{Success: false, MessageToCh: "You haven't the energy!"}
	}
	ch.Move -= 10

	// Bash formula: percent = ((5 - (GET_AC(vict)/10)) << 1) + number(1,101)
	// prob = GET_SKILL(ch, SKILL_BASH)
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := ((5 - (target.GetAC() / 10)) * 2) + (rand.Intn(101) + 1)
	prob := ch.GetSkill(SkillBash)

	chPronouns := GetPronouns(ch.Name, 1)
	victPronouns := GetPronouns(target.GetName(), 1)

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
	dam := (ch.Level / 2) + 1
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
	percent := ((7 - (target.GetAC() / 10)) * 2) + (rand.Intn(101) + 1)
	prob := ch.GetSkill(SkillKick)

	chPronouns := GetPronouns(ch.Name, 1)
	victPronouns := GetPronouns(target.GetName(), 1)

	if percent > prob {
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to kick $N, but miss!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to kick you, but misses!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to kick $N, but misses!", chPronouns, &victPronouns, ""),
		}
	}

	dam := ch.Level >> 1 // level / 2
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
	percent := rand.Intn(121) + 1
	percent += max(target.GetLevel()-ch.Level, 0)
	prob := ch.GetSkill(SkillTrip)

	chPronouns := GetPronouns(ch.Name, 1)
	victPronouns := GetPronouns(target.GetName(), 1)

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
	dam := (ch.Level / 2) + 1
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
	if ch.Move < 15 {
		return SkillResult{Success: false, MessageToCh: "You haven't the energy!"}
	}
	ch.Move -= 15

	skillLevel := ch.GetSkill(SkillHeadbutt)
	hitRoll := (skillLevel/2 + 1) - 10 // DAMAGE_ROLL approximation minus accuracy penalty
	if hitRoll < 1 {
		hitRoll = 1
	}
	damage := (skillLevel/2 + 1) + 4 // higher base damage

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(101) + 1

	chPronouns := GetPronouns(ch.Name, 1)
	victPronouns := GetPronouns(target.GetName(), 1)

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
		if rand.Intn(4) == 0 {
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
		Success:     true,
		Damage:      damage,
		MessageToCh: ActMessage("You slam your forehead into $N with a sickening crack!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n slams $s forehead into you with a sickening crack!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n slams $s forehead into $N with a sickening crack!", chPronouns, &victPronouns, ""),
		StunTarget:   true,
		WaitCh:       2,
	}
}

// DoRescue implements do_rescue() from act.offensive.c lines 480-539.
// Interposes between attacker and target.
func DoRescue(ch *Player, target combat.Combatant, world *World, combatEngine interface{ StartCombat(combat.Combatant, combat.Combatant) error; StopCombat(string) }) SkillResult {
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
		victPronouns := GetPronouns(target.GetName(), 1)
		return SkillResult{Success: false, MessageToCh: ActMessage("But nobody is fighting $N!", GetPronouns(ch.Name, 1), &victPronouns, "")}
	}

	// Roll for success
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillRescue)

	chPronouns := GetPronouns(ch.Name, 1)
	victPronouns := GetPronouns(target.GetName(), 1)

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
