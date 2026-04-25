package game

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// Skill constants — matching Dark Pawns C source
// ---------------------------------------------------------------------------

const (
	SkillBackstab  = "backstab"
	SkillFlee      = "flee"
	SkillBash      = "bash"
	SkillKick      = "kick"
	SkillTrip      = "trip"
	SkillHeadbutt  = "headbutt"
	SkillRescue    = "rescue"
	SkillSneak     = "sneak"
	SkillHide      = "hide"
	SkillSteal     = "steal"
	SkillPickLock  = "pick_lock"
	SkillCarve     = "carve"
	SkillCutthroat = "cutthroat"
	SkillStrike    = "strike"
	SkillCompare   = "compare"
	SkillScan      = "scan"
	SkillSharpen   = "sharpen"

	// Wave 2 skills (new_cmds2.c)
	SkillScrounge     = "scrounge"

	// act.other.c skills
	SkillPeek     = "peek"
	SkillStealth  = "stealth"
	SkillAppraise = "appraise"
	SkillScout    = "scout"
	SkillFirstAid     = "first_aid"
	SkillDisarm       = "disarm"
	SkillMindlink     = "mindlink"
	SkillDetect       = "detect"
	SkillSerpentKick  = "serpent_kick"
	SkillDig          = "dig"
	SkillTurn         = "turn"

	// Wave 1 cleanup skills (new_cmds.c)
	SkillMold        = "mold"
	SkillBehead      = "behead"
	SkillBearhug     = "bearhug"
	SkillSlug        = "slug"
	SkillSmackheads  = "smackheads"
	SkillBite        = "bite"
	SkillTag         = "tag"
	SkillPoint       = "point"
	SkillGroinrip    = "groinrip"
	SkillReview      = "review"
	SkillWhois       = "whois"
	SkillPalm        = "palm"
	SkillFleshAlter  = "flesh_alter"
)

// ---------------------------------------------------------------------------
// Skill level requirements by class — from src/class.c spell_level() calls
// ---------------------------------------------------------------------------

// SkillClassReq maps skill name → class → minimum level to learn.
// A class not in the map cannot learn that skill.
// Source: class.c spell_level() calls for each skill.
var SkillClassReq = map[string]map[int]int{
	SkillBackstab: {
		ClassThief:    1,
		ClassAssassin: 1,
	},
	SkillBash: {
		ClassWarrior: 3,
		ClassPaladin: 3,
		ClassRanger:  3,
	},
	SkillKick: {
		ClassWarrior:  1,
		ClassPaladin:  1,
		ClassRanger:   1,
		ClassThief:    1,
		ClassCleric:   1,
		ClassMageUser: 1,
		ClassMagus:    1,
		ClassAvatar:   1,
		ClassAssassin: 1,
		ClassNinja:    1,
		ClassPsionic:  1,
		ClassMystic:   1,
	},
	SkillTrip: {
		ClassThief:    9,
		ClassAssassin: 9,
	},
	SkillHeadbutt: {
		ClassWarrior: 5,
		ClassPaladin: 5,
		ClassRanger:  7,
	},
	SkillRescue: {
		ClassWarrior: 4,
		ClassPaladin: 3,
		ClassRanger:  5,
	},
	SkillSneak: {
		ClassThief:    2,
		ClassAssassin: 2,
	},
	SkillHide: {
		ClassThief:    5,
		ClassAssassin: 5,
		ClassRanger:   10,
	},
	SkillSteal: {
		ClassThief:    3,
		ClassAssassin: 3,
	},
	SkillPickLock: {
		ClassThief:    4,
		ClassAssassin: 4,
	},
}

// ---------------------------------------------------------------------------
// Position requirements — from interpreter.c cmd_info[] table
// ---------------------------------------------------------------------------

// SkillPosReq maps skill name → minimum position required.
// Source: interpreter.c cmd_info[] entries.
var SkillPosReq = map[string]int{
	SkillBackstab: combat.PosStanding,
	SkillFlee:     combat.PosFighting,
	SkillBash:     combat.PosFighting,
	SkillKick:     combat.PosFighting,
	SkillTrip:     combat.PosFighting,
	SkillHeadbutt: combat.PosFighting,
	SkillRescue:   combat.PosStanding,
	SkillSneak:    combat.PosStanding,
	SkillHide:     combat.PosResting,
	SkillSteal:    combat.PosStanding,
	SkillPickLock: combat.PosStanding,
}

// ---------------------------------------------------------------------------
// Helper: check if player can use a skill
// ---------------------------------------------------------------------------

// CanUseSkill checks class/level and position requirements.
func CanUseSkill(p *Player, skillName string) (bool, string) {
	classReqs, ok := SkillClassReq[skillName]
	if !ok {
		return false, "You have no idea how."
	}

	minLevel, classOk := classReqs[p.Class]
	if !classOk {
		return false, "You have no idea how."
	}
	if p.Level < minLevel {
		return false, fmt.Sprintf("You must be at least level %d to use that skill.", minLevel)
	}

	// Check position
	minPos := SkillPosReq[skillName]
	if p.GetPosition() < minPos {
		switch minPos {
		case combat.PosStanding:
			return false, "You must be standing to do that."
		case combat.PosFighting:
			return false, "You must be fighting to do that!"
		default:
			return false, "You can't do that right now."
		}
	}

	return true, ""
}

// ---------------------------------------------------------------------------
// Target finding helpers
// ---------------------------------------------------------------------------

// FindTargetInRoom finds a character (player or mob) in the same room.
func FindTargetInRoom(world *World, roomVNum int, targetName string, exclude *Player) (combat.Combatant, string, bool) {
	targetName = strings.ToLower(targetName)

	// Check mobs
	mobs := world.GetMobsInRoom(roomVNum)
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
			return mob, mob.GetShortDesc(), true
		}
	}

	// Check players
	players := world.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if exclude != nil && p.Name == exclude.Name {
			continue
		}
		if strings.Contains(strings.ToLower(p.Name), targetName) {
			return p, p.Name, true
		}
	}

	return nil, "", false
}

// ---------------------------------------------------------------------------
// Pronoun resolution for act() messages
// ---------------------------------------------------------------------------

// Pronouns holds pronoun substitutions for a character.
type Pronouns struct {
	Name string // $n / $N
	He   string // $e / $E
	Him  string // $m / $M
	His  string // $s / $S
}

// GetPronouns returns pronouns for a character based on sex.
// Sex: 0 = neutral, 1 = male, 2 = female (from structs.h)
func GetPronouns(name string, sex int) Pronouns {
	var he, him, his string
	switch sex {
	case 2: // female
		he, him, his = "she", "her", "her"
	case 1: // male
		he, him, his = "he", "him", "his"
	default: // neutral
		he, him, his = "it", "it", "its"
	}
	return Pronouns{Name: name, He: he, Him: him, His: his}
}

// ActMessage resolves pronoun codes in a message string.
// chPronouns = the actor ($n, $e, $m, $s)
// victPronouns = the target ($N, $E, $M, $S) — optional
// itemName = the item ($p) — optional
func ActMessage(msg string, chPronouns Pronouns, victPronouns *Pronouns, itemName string) string {
	result := msg
	result = strings.ReplaceAll(result, "$n", chPronouns.Name)
	result = strings.ReplaceAll(result, "$e", chPronouns.He)
	result = strings.ReplaceAll(result, "$m", chPronouns.Him)
	result = strings.ReplaceAll(result, "$s", chPronouns.His)
	if victPronouns != nil {
		result = strings.ReplaceAll(result, "$N", victPronouns.Name)
		result = strings.ReplaceAll(result, "$E", victPronouns.He)
		result = strings.ReplaceAll(result, "$M", victPronouns.Him)
		result = strings.ReplaceAll(result, "$S", victPronouns.His)
	}
	if itemName != "" {
		result = strings.ReplaceAll(result, "$p", itemName)
	}
	return result
}

// ---------------------------------------------------------------------------
// Skill implementations
// ---------------------------------------------------------------------------

// SkillResult holds the outcome of a skill use.
type SkillResult struct {
	Success     bool
	Damage      int
	MessageToCh string
	MessageToVict string
	MessageToRoom string
	StunTarget  bool   // target loses a round
	SelfStumble bool   // user falls (bash fail)
	TargetFalls bool   // target position changes to sitting
}

// DoBackstab implements do_backstab() from act.offensive.c lines 172-220.
// Requires: piercing weapon wielded, target not fighting, target awake.
// Damage: weapon damage * backstab multiplier (level*0.2 + 1).
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
	percent := rand.Intn(101) + 1 // 1-101
	skillLevel := ch.GetSkill(SkillBackstab)
	prob := skillLevel
	if prob == 0 {
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

	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   "Your deadly backstab strikes deep!",
		MessageToVict: ActMessage("$n sneaks up from behind and plunges a dagger into you!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n sneaks up from behind and backstabs $N!", chPronouns, &victPronouns, ""),
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
		}
	}

	// Success — damage = (level/2)+1
	dam := (ch.Level / 2) + 1
	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   ActMessage("You send $N flying with a powerful bash!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n sends you flying with a powerful bash!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n sends $N flying with a powerful bash!", chPronouns, &victPronouns, ""),
		TargetFalls:   true,
		StunTarget:    true,
	}
}

// DoKick implements do_kick() from act.offensive.c lines 541-576.
// Simple damage: level >> 1 (level/2).
func DoKick(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillKick) == 0 {
		return SkillResult{Success: false, MessageToCh: "You'd better leave all the martial arts to fighters."}
	}

	// Formula: percent = ((7 - (GET_AC(vict)/10)) << 1) + number(1,101)
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
	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   ActMessage("You kick $N square in the chest!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n kicks you square in the chest!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n kicks $N square in the chest!", chPronouns, &victPronouns, ""),
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
		}
	}

	// Success — damage = (level/2)+1, target falls
	dam := (ch.Level / 2) + 1
	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   ActMessage("You trip $N sending $M crashing to the ground!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n trips you sending you crashing to the ground!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n trips $N sending $M crashing to the ground!", chPronouns, &victPronouns, ""),
		TargetFalls:   true,
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
		}
		// 25% self-stun on failure
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

	return SkillResult{
		Success:     true,
		Damage:      damage,
		MessageToCh: ActMessage("You slam your forehead into $N with a sickening crack!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n slams $s forehead into you with a sickening crack!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n slams $s forehead into $N with a sickening crack!", chPronouns, &victPronouns, ""),
		StunTarget:   true,
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
	// We need to use the combat engine to handle this properly
	// For now, return success and let the caller handle combat state
	return SkillResult{
		Success:       true,
		MessageToCh:   "Banzai!  To the rescue...",
		MessageToVict: ActMessage("You are rescued by $N, you are confused!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n heroically rescues $N!", chPronouns, &victPronouns, ""),
	}
}

// ---------------------------------------------------------------------------
// Sneak / Hide / Steal state
// ---------------------------------------------------------------------------

// PlayerSneakState tracks sneak mode per player.
// In original: AFF_SNEAK affect flag.
var playerSneakState = make(map[string]bool)

// PlayerHideState tracks hide mode per player.
// In original: AFF_HIDE affect flag.
var playerHideState = make(map[string]bool)

// IsSneaking returns true if the player is in sneak mode.
func IsSneaking(name string) bool {
	return playerSneakState[name]
}

// SetSneaking sets sneak mode.
func SetSneaking(name string, val bool) {
	playerSneakState[name] = val
}

// IsHidden returns true if the player is hidden.
func IsHidden(name string) bool {
	return playerHideState[name]
}

// SetHidden sets hide mode.
func SetHidden(name string, val bool) {
	playerHideState[name] = val
}

// DoSneak implements do_sneak() from act.other.c lines 214-245.
func DoSneak(ch *Player) SkillResult {
	if ch.GetSkill(SkillSneak) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// Toggle off if already sneaking
	if IsSneaking(ch.Name) {
		SetSneaking(ch.Name, false)
		return SkillResult{Success: true, MessageToCh: "You stop sneaking."}
	}

	// Roll for success: percent = number(1,101)
	// prob = GET_SKILL(ch, SKILL_SNEAK) + dex_app_skill[GET_DEX(ch)].sneak
	// We don't have dex_app_skill table yet, use raw skill
	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillSneak)

	if percent > prob {
		return SkillResult{Success: false, MessageToCh: "You attempt to move silently, but make too much noise."}
	}

	SetSneaking(ch.Name, true)
	return SkillResult{Success: true, MessageToCh: "Okay, you'll try to move silently for a while."}
}

// DoHide implements do_hide() from act.other.c lines 247-307.
func DoHide(ch *Player) SkillResult {
	if ch.GetSkill(SkillHide) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// Toggle off if already hidden
	if IsHidden(ch.Name) {
		SetHidden(ch.Name, false)
		return SkillResult{Success: true, MessageToCh: "You step out of the shadows."}
	}

	// Roll for success
	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillHide)

	if percent > prob {
		return SkillResult{Success: false, MessageToCh: "You attempt to hide yourself, but fail."}
	}

	SetHidden(ch.Name, true)
	return SkillResult{Success: true, MessageToCh: "You blend into the shadows."}
}

// DoSteal implements do_steal() from act.other.c lines 309-560.
// Simplified: steal gold or an item from target's inventory.
func DoSteal(ch *Player, target combat.Combatant, itemName string) SkillResult {
	if ch.GetSkill(SkillSteal) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// Can't steal from yourself
	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "Come on now, that's rather stupid!"}
	}

	// Target can't be fighting
	if target.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "You can't steal from someone who's fighting!"}
	}

	chPronouns := GetPronouns(ch.Name, 1)
	victPronouns := GetPronouns(target.GetName(), 1)

	// Steal gold
	if itemName == "coins" || itemName == "gold" {
		percent := rand.Intn(101) + 1
		prob := ch.GetSkill(SkillSteal)

		if percent > prob {
			return SkillResult{
				Success:       false,
				MessageToCh:   "Oops..",
				MessageToVict: ActMessage("You discover that $n has $s hands in your wallet.", chPronouns, &victPronouns, ""),
				MessageToRoom: ActMessage("$n tries to steal gold from $N.", chPronouns, &victPronouns, ""),
			}
		}

		// Calculate gold stolen: (GET_GOLD(vict) * number(1,10)) / 100, max 1782
		// We need access to target's gold — for players we can cast, for mobs we estimate
		var gold int
		if p, ok := target.(*Player); ok {
			gold = (p.Gold * (rand.Intn(10) + 1)) / 100
			if gold > 1782 {
				gold = 1782
			}
			if gold > p.Gold {
				gold = p.Gold
			}
			p.Gold -= gold
			ch.Gold += gold
		} else {
			// Mob — steal small random amount
			gold = rand.Intn(20) + 1
			ch.Gold += gold
		}

		if gold > 1 {
			return SkillResult{
				Success:     true,
				MessageToCh: fmt.Sprintf("Bingo!  You got %d gold coins.", gold),
			}
		} else if gold == 1 {
			return SkillResult{Success: true, MessageToCh: "You manage to swipe a solitary gold coin."}
		}
		return SkillResult{Success: true, MessageToCh: "You couldn't get any gold..."}
	}

	// Steal item — simplified, only from player inventory for now
	if p, ok := target.(*Player); ok {
		// Find item in target's inventory
		item, found := p.Inventory.FindItem(itemName)
		if !found {
			return SkillResult{Success: false, MessageToCh: ActMessage("$E hasn't got that item.", chPronouns, &victPronouns, "")}
		}

		// Roll with weight penalty
		percent := rand.Intn(101) + 1
		// Heavier items are harder to steal
		// percent += GET_OBJ_WEIGHT(obj) — we don't have weight yet
		if p.Level > ch.Level {
			percent += p.Level - ch.Level
		}
		prob := ch.GetSkill(SkillSteal)

		if percent > prob {
			return SkillResult{
				Success:       false,
				MessageToCh:   ActMessage("$N catches you trying to steal something...", chPronouns, &victPronouns, ""),
				MessageToVict: ActMessage("$n tried to steal something from you!", chPronouns, &victPronouns, ""),
				MessageToRoom: ActMessage("$n tries to steal something from $N.", chPronouns, &victPronouns, ""),
			}
		}

		// Steal the item
		p.Inventory.RemoveItem(item)
		ch.Inventory.AddItem(item)
		return SkillResult{
			Success:       true,
			MessageToCh:   ActMessage("You deftly steal $p from $N's pocket!", chPronouns, &victPronouns, item.GetShortDesc()),
			MessageToVict: "",
			MessageToRoom: "",
		}
	}

	return SkillResult{Success: false, MessageToCh: "You can't steal that."}
}

// DoPickLock implements do_pick() — simplified version.
// In original: act.movement.c do_gen_door() with SCMD_PICK.
// For now, just a skill check with messaging.
func DoPickLock(ch *Player) SkillResult {
	if ch.GetSkill(SkillPickLock) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// This is a placeholder — actual pick lock logic is in door_commands.go
	// which handles the full door/unlock logic.
	return SkillResult{Success: true, MessageToCh: "You attempt to pick the lock..."}
}

// DoCarve implements do_carve() — carve food from a corpse.
func DoCarve(ch *Player, targetName string, world *World) SkillResult {
	// Find target corpse in room
	objects := world.GetItemsInRoom(ch.GetRoomVNum())
	var corpse *ObjectInstance
	for _, obj := range objects {
		if obj.Prototype.TypeFlag == 9 && strings.Contains(strings.ToLower(obj.GetShortDesc()), strings.ToLower(targetName)) {
			corpse = obj
			break
		}
	}

	if corpse == nil {
		return SkillResult{Success: false, MessageToCh: "There is nothing like that here."}
	}

	if ch.GetSkill(SkillCarve) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// Create food item
	food := &ObjectInstance{
		VNum:    corpse.VNum,
		RoomVNum: ch.GetRoomVNum(),
	}
	if food.CustomData == nil {
		food.CustomData = make(map[string]interface{})
	}
	food.CustomData["short_desc_override"] = "some carved meat from " + corpse.GetShortDesc()

	if err := ch.Inventory.AddItem(food); err != nil {
		world.AddItemToRoom(food, ch.GetRoomVNum())
	}

	// Remove corpse from room
	world.RemoveItemFromRoom(corpse, ch.GetRoomVNum())

	return SkillResult{
		Success:     true,
		MessageToCh: fmt.Sprintf("You carve some meat from %s.", corpse.GetShortDesc()),
	}
}

// DoCutthroat implements do_cutthroat() — attempt instant kill from behind.
func DoCutthroat(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillCutthroat) == 0 {
		return SkillResult{Success: false, MessageToCh: "You don't know how!"}
	}

	if target.GetHP() <= 0 {
		return SkillResult{Success: false, MessageToCh: "They're already dead!"}
	}

	// Skill check: D100 vs skill
	roll := rand.Intn(100) + 1
	if roll > ch.GetSkill(SkillCutthroat) {
		return SkillResult{
			Success:     false,
			MessageToCh: "Your attempt fails!",
		}
	}

	// Instant kill: set target to -1 HP
	damage := target.GetHP() + 1
	target.TakeDamage(damage)

	return SkillResult{
		Success:     true,
		Damage:      damage,
		MessageToCh: "You slash their throat!",
		MessageToVict: "Your throat is slashed!",
		MessageToRoom: fmt.Sprintf("%s slashes %s's throat!", ch.Name, target.GetName()),
	}
}

// DoStrike implements do_strike() — quick attack.
func DoStrike(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillStrike) == 0 {
		return SkillResult{Success: false, MessageToCh: "You don't know how!"}
	}

	// Simple damage based on level
	damage := rand.Intn(ch.GetLevel()) + 1

	return SkillResult{
		Success:     true,
		Damage:      damage,
		MessageToCh: fmt.Sprintf("You strike %s!", target.GetName()),
		MessageToVict: fmt.Sprintf("%s strikes you!", ch.Name),
		MessageToRoom: fmt.Sprintf("%s strikes %s!", ch.Name, target.GetName()),
	}
}

// DoCompare implements do_compare() — compare weapons or armor.
func DoCompare(ch *Player, objName1, objName2 string, compareToEquipped bool) SkillResult {
	// Find the first object
	obj1, found := findItemByName(ch, objName1)
	if !found {
		return SkillResult{Success: false, MessageToCh: "You don't have that item."}
	}

	// Find the second object
	if compareToEquipped {
		// Compare against equipped weapon
		result := fmt.Sprintf("Comparing %s with your weapon...", obj1.GetShortDesc())
		return SkillResult{Success: true, MessageToCh: result}
	}

	obj2, found := findItemByName(ch, objName2)
	if !found {
		return SkillResult{Success: false, MessageToCh: "You don't have that item."}
	}

	result := fmt.Sprintf("%s vs %s: comparing...", obj1.GetShortDesc(), obj2.GetShortDesc())
	return SkillResult{Success: true, MessageToCh: result}
}

// DoScan implements do_scan() — scan surrounding rooms.
func DoScan(ch *Player, world *World) SkillResult {
	if ch.GetSkill(SkillScan) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	// Get current room exits
	room := world.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil {
		return SkillResult{Success: false, MessageToCh: "You are in a void."}
	}

	var scanResult string
	scanResult = "You scan the area...\r\n"

	for dir, exit := range room.Exits {
		if exit.ToRoom > 0 {
			exitRoom := world.GetRoomInWorld(exit.ToRoom)
			if exitRoom != nil {
				exitName := exitRoom.Name
				// Check for players in that room
				players := world.GetPlayersInRoom(exit.ToRoom)
				if len(players) > 0 {
					for _, p := range players {
						scanResult += fmt.Sprintf("%-5s - %s is there.\r\n", strings.ToUpper(dir), p.Name)
					}
				} else {
					scanResult += fmt.Sprintf("%-5s - %s (empty)\r\n", strings.ToUpper(dir), exitName)
				}
			}
		}
	}

	if scanResult == "You scan the area...\r\n" {
		scanResult += "Nothing interesting."
	}

	return SkillResult{Success: true, MessageToCh: scanResult}
}

// DoSharpen implements do_sharpen() — sharpen a weapon.
func DoSharpen(ch *Player, objName string) SkillResult {
	if ch.GetSkill(SkillSharpen) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how."}
	}

	obj, found := findItemByName(ch, objName)
	if !found {
		return SkillResult{Success: false, MessageToCh: "You don't have that item."}
	}

	// Check it's a weapon
	if obj.Prototype.TypeFlag != 0 {
		return SkillResult{Success: false, MessageToCh: "You can only sharpen weapons."}
	}

	// Simple sharpen: success based on skill level
	roll := rand.Intn(100) + 1
	if roll <= ch.GetSkill(SkillSharpen) {
		return SkillResult{
			Success:     true,
			MessageToCh: fmt.Sprintf("You sharpen %s. It looks more deadly!", obj.GetShortDesc()),
		}
	}

	return SkillResult{
		Success:     false,
		MessageToCh: "You fail to sharpen it properly.",
	}
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

// findItemByName searches a player's inventory and equipment for an item matching name.
func findItemByName(ch *Player, name string) (*ObjectInstance, bool) {
	name = strings.ToLower(name)

	// Check inventory
	for _, obj := range ch.Inventory.Items {
		if obj != nil && strings.Contains(strings.ToLower(obj.GetShortDesc()), name) {
			return obj, true
		}
	}

	// Check equipment
	for _, obj := range ch.Equipment.Slots {
		if obj != nil && strings.Contains(strings.ToLower(obj.GetShortDesc()), name) {
			return obj, true
		}
	}

	return nil, false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// Wave 1 cleanup — remaining commands from new_cmds.c
// DoMold, DoBehead, DoBearhug, DoSlug, DoSmackheads, DoBite, DoTag,
// DoPoint, DoGroinrip, DoReview, DoWhois, DoPalm, DoFleshAlter
// ---------------------------------------------------------------------------

// DoMold implements do_mold() — rename and redescribe a clay item.
func DoMold(ch *Player, objName, newName, newDesc string) SkillResult {
	obj, found := findItemByName(ch, objName)
	if !found {
		return SkillResult{Success: false, MessageToCh: "You don't have one of those.\r\n"}
	}

	name := strings.ToLower(obj.GetKeywords())
	if !strings.Contains(name, "clay") && !strings.Contains(name, "playdough") && !strings.Contains(name, "halo") {
		return SkillResult{Success: false, MessageToCh: "You do not have anything to mold!\r\n"}
	}

	if newName == "" || newDesc == "" {
		return SkillResult{Success: false, MessageToCh: "You must specify a name and a description.\r\n"}
	}

	// Store custom mold data
	if obj.CustomData == nil {
		obj.CustomData = make(map[string]interface{})
	}
	obj.CustomData["mold_name"] = newName
	obj.CustomData["mold_desc"] = newDesc

	return SkillResult{
		Success:     true,
		MessageToCh: fmt.Sprintf("The material magically hardens when you create %s.\r\n", newDesc),
	}
}

// DoBehead implements do_behead() — behead a corpse.
func DoBehead(ch *Player, targetName string, world *World) SkillResult {
	// Check if target is a living character
	target, _, found := FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
	if found && target != nil {
		return SkillResult{Success: false, MessageToCh: "You kill it first and THEN you behead it!\r\n"}
	}

	// Look for corpse object in room
	room := world.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil {
		return SkillResult{MessageToCh: "You are in a void.\r\n"}
	}

	items := world.GetItemsInRoom(ch.GetRoomVNum())
	var corpse *ObjectInstance
	targetLower := strings.ToLower(targetName)
	for _, item := range items {
		iname := strings.ToLower(item.GetKeywords())
		if strings.Contains(iname, "corpse") && strings.Contains(iname, targetLower) {
			corpse = item
			break
		}
	}

	if corpse == nil {
		// Fallback: find any corpse matching name
		for _, item := range items {
			iname := strings.ToLower(item.GetKeywords())
			if strings.Contains(iname, "corpse") {
				corpse = item
				break
			}
		}
	}

	if corpse == nil {
		return SkillResult{Success: false, MessageToCh: fmt.Sprintf("You can't seem to find a %s to behead!\r\n", targetName)}
	}

	if strings.Contains(strings.ToLower(corpse.GetKeywords()), "headless") {
		return SkillResult{Success: false, MessageToCh: "You can't behead something without a head!\r\n"}
	}

	// Check if it's a container (c-style: ITEM_CONTAINER with val[3] == 1 = corpse)
	// For now, just check it's a corpse object
	if !strings.Contains(strings.ToLower(corpse.GetKeywords()), "corpse") {
		return SkillResult{Success: false, MessageToCh: "You can't behead that!\r\n"}
	}

	// Determine weapon type for messaging
	wielded := false
	slashWeapon := false
	if ch.Equipment != nil && len(ch.Equipment.Slots) > 0 {
		weapon := ch.Equipment.Slots[0] // WEAR_WIELD = slot 0
		if weapon != nil {
			wielded = true
			// Check if weapon type is slash (value[3] == 3)
			slashWeapon = true // simplified — assume equipped weapons are slash-able
		}
	}

	var msgToCh, msgToRoom string
	if wielded && slashWeapon {
		msgToCh = fmt.Sprintf("You behead %s!", corpse.GetShortDesc())
		msgToRoom = fmt.Sprintf("%s beheads %s!", ch.Name, corpse.GetShortDesc())
	} else {
		msgToCh = fmt.Sprintf("You rip the head off %s with your bare hands!", corpse.GetShortDesc())
		msgToRoom = fmt.Sprintf("%s rips the head off %s with %s bare hands!", ch.Name, corpse.GetShortDesc(), heShe(ch.GetSex()))
	}

	// Create head object (proto vnum 16)
	_ = world.GetItemsInRoom(ch.GetRoomVNum()) // room items ref

	// Since we can't easily create objects from proto, store modified name on corpse
	// and use the corpse's short desc for the room message

	// Dump corpse contents and remove it
	// In a full port we'd create head + headless_corpse objects
	// For now, mark the corpse as beheaded and dump its contents
	world.RemoveItemFromRoom(corpse, ch.GetRoomVNum())

	// Create head (vnum 16) and headless corpse (vnum 17) objects
	headObj, err := world.SpawnObject(16, ch.GetRoomVNum())
	if err == nil && headObj != nil {
		headObj.CustomData["short_desc"] = fmt.Sprintf("the severed head of %s", ch.Name)
		headObj.CustomData["name"] = fmt.Sprintf("head %s", ch.Name)
	}
	headlessCorpseObj, err := world.SpawnObject(17, ch.GetRoomVNum())
	if err == nil && headlessCorpseObj != nil {
		headlessCorpseObj.CustomData["short_desc"] = fmt.Sprintf("the headless corpse of %s", ch.Name)
		headlessCorpseObj.CustomData["name"] = fmt.Sprintf("corpse headless %s", ch.Name)
	}

	return SkillResult{
		Success:      true,
		MessageToCh:  msgToCh + "\r\n",
		MessageToRoom: msgToRoom + "\r\n",
	}
}

// DoBearhug implements do_bearhug() — bare-handed squeeze attack.
func DoBearhug(ch *Player, target combat.Combatant, world *World) SkillResult {
	if ch.GetSkill(SkillBearhug) == 0 {
		return SkillResult{Success: false, MessageToCh: "You'd better leave all the martial arts to fighters.\r\n"}
	}

	if ch.GetMove() <= 0 {
		return SkillResult{Success: false, MessageToCh: "You are too exhausted!\r\n"}
	}

	if ch.Equipment != nil && len(ch.Equipment.Slots) > 0 && ch.Equipment.Slots[0] != nil {
		return SkillResult{Success: false, MessageToCh: "You need to be bare handed to get a good grip.\r\n"}
	}

	percent := rand.Intn(150) + 1 // 1-150; 101+ is complete failure

	// Immortals always succeed, sleeping targets always hit
	if ch.GetLevel() > 60 {
		percent = 101
	}

	prob := ch.GetSkill(SkillBearhug)

	if percent > prob {
		return SkillResult{
			Success:      true,
			Damage:       0,
			MessageToCh:  "You try to bear hug but miss!\r\n",
			MessageToVict: "$n tries to bear hug you!\r\n",
			MessageToRoom: fmt.Sprintf("%s tries to bear hug %s!\r\n", ch.Name, target.GetName()),
		}
	}

	dam := ch.GetLevel() + (ch.GetLevel() / 2) // level * 1.5

	return SkillResult{
		Success:      true,
		Damage:       dam,
		MessageToCh:  "You squeeze your victim in a crushing bear hug!\r\n",
		MessageToVict: "You are crushed in a powerful bear hug!\r\n",
		MessageToRoom: fmt.Sprintf("%s crushes %s in a powerful bear hug!\r\n", ch.Name, target.GetName()),
	}
}

// DoSlug implements do_slug() — punch attack.
func DoSlug(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillSlug) == 0 {
		return SkillResult{Success: false, MessageToCh: "You couldn't slug your way out of a wet paper bag.\r\n"}
	}

	if ch.Equipment != nil && len(ch.Equipment.Slots) > 0 && ch.Equipment.Slots[0] != nil {
		return SkillResult{Success: false, MessageToCh: "You can't make a fist while wielding a weapon!\r\n"}
	}

	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillSlug)

	if percent > prob {
		return SkillResult{
			Success:      true,
			Damage:       0,
			MessageToCh:  "You swing wildly and miss!\r\n",
			MessageToVict: "$n swings a fist at you and misses!\r\n",
			MessageToRoom: fmt.Sprintf("%s swings a fist at %s and misses!\r\n", ch.Name, target.GetName()),
		}
	}

	dam := (ch.GetLevel() * (rand.Intn(4) + 1)) / 2
	return SkillResult{
		Success:      true,
		Damage:       dam,
		MessageToCh:  "You slug your victim with a solid punch!\r\n",
		MessageToVict: "You are slugged hard!\r\n",
		MessageToRoom: fmt.Sprintf("%s slugs %s!\r\n", ch.Name, target.GetName()),
	}
}

// DoSmackheads implements do_smackheads() — grab two NPCs and smack them together.
func DoSmackheads(ch *Player, victim1Name, victim2Name string, world *World) SkillResult {
	if ch.GetSkill(SkillSmackheads) == 0 {
		return SkillResult{Success: false, MessageToCh: "The only heads you're gonna smack are yours and Rosie's.\r\n"}
	}

	if victim1Name == victim2Name {
		return SkillResult{Success: false, MessageToCh: "Looks like the gang's not all here...\r\n"}
	}

	vill, _, found1 := FindTargetInRoom(world, ch.GetRoomVNum(), victim1Name, ch)
	vil2, _, found2 := FindTargetInRoom(world, ch.GetRoomVNum(), victim2Name, ch)
	if !found1 || !found2 {
		return SkillResult{Success: false, MessageToCh: "Looks like the gang's not all here...\r\n"}
	}

	// Check we're not targeting ourselves
	if vill.GetName() == ch.Name || vil2.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "We call that 'headbutt' around here, son...\r\n"}
	}

	if ch.Equipment != nil && len(ch.Equipment.Slots) > 0 && ch.Equipment.Slots[0] != nil {
		return SkillResult{Success: false, MessageToCh: "You need your hands free to smack some heads!\r\n"}
	}

	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillSmackheads)

	if percent > prob {
		// Failure — victims duck
		msgToCh := fmt.Sprintf("%s and %s slip out of your hands!", vill.GetName(), vil2.GetName())
		return SkillResult{
			Success:      true,
			MessageToCh:  msgToCh + "\r\n",
			MessageToRoom: fmt.Sprintf("%s and %s duck as %s lunges at them!\r\n", vill.GetName(), vil2.GetName(), ch.Name),
		}
	}

	// Success — smack them together
	dam := 3 * ch.GetLevel()
	return SkillResult{
		Success:      true,
		Damage:       dam,
		MessageToCh:  fmt.Sprintf("You grab the heads of %s and %s and bang them together with a sickening *SMACK*.\r\n", vill.GetName(), vil2.GetName()),
		MessageToRoom: fmt.Sprintf("%s grabs the heads of %s and %s and bangs them together with a sickening *SMACK*.\r\n", ch.Name, vill.GetName(), vil2.GetName()),
	}
}

// DoBite implements do_bite() — vampire/werewolf bite attack.
func DoBite(ch *Player, target combat.Combatant) SkillResult {
	// Non-supernatural bite (love bite)
	dam := ch.GetLevel()
	if dam > 15 {
		dam = 15
	}

	return SkillResult{
		Success:      true,
		Damage:       dam,
		MessageToCh:  "You bite your victim!\r\n",
		MessageToVict: "$n bites you!\r\n",
		MessageToRoom: fmt.Sprintf("%s bites %s!\r\n", ch.Name, target.GetName()),
	}
}

// DoTag implements do_tag() — tag someone as "it".
func DoTag(ch *Player, targetName string, world *World) SkillResult {
	if targetName == "" {
		return SkillResult{Success: false, MessageToCh: "Tag who?\r\n"}
	}

	target, _, found := FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
	if !found {
		return SkillResult{Success: false, MessageToCh: "They don't seem to be here.\r\n"}
	}

	// Self-tag starts the game
	if target.GetName() == ch.Name {
		return SkillResult{Success: true, MessageToCh: "Let the game begin!\r\n"}
	}

	return SkillResult{
		Success:      true,
		MessageToCh:  fmt.Sprintf("You tap %s and scream, 'TAG! You're it!'\r\n", target.GetName()),
		MessageToVict: fmt.Sprintf("%s taps you and screams, 'TAG! You're it!'\r\n", ch.Name),
		MessageToRoom: fmt.Sprintf("%s taps %s and screams, 'TAG! You're it!'\r\n", ch.Name, target.GetName()),
	}
}

// DoPoint implements do_point() — point at someone or something.
func DoPoint(ch *Player, targetName string, world *World) SkillResult {
	if targetName == "" {
		return SkillResult{
			Success:      true,
			MessageToCh:  "You point around the room.\r\n",
			MessageToRoom: fmt.Sprintf("%s points around the room.\r\n", ch.Name),
		}
	}

	target, _, found := FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
	if !found {
		// Point at self
		if strings.EqualFold(targetName, "self") || strings.EqualFold(targetName, "me") || strings.EqualFold(targetName, ch.Name) {
			return SkillResult{
				Success:      true,
				MessageToCh:  "You point at yourself.\r\n",
				MessageToRoom: fmt.Sprintf("%s points at %s.\r\n", ch.Name, himHer(ch.GetSex())),
			}
		}
		// Point at nothing specific
		return SkillResult{
			Success:      true,
			MessageToCh:  "You point around the room.\r\n",
			MessageToRoom: fmt.Sprintf("%s points around the room.\r\n", ch.Name),
		}
	}

	if target.GetName() == ch.Name {
		return SkillResult{
			Success:      true,
			MessageToCh:  "You point at yourself.\r\n",
			MessageToRoom: fmt.Sprintf("%s points at %s.\r\n", ch.Name, himHer(ch.GetSex())),
		}
	}

	// Check if wielding a weapon
	if ch.Equipment != nil && len(ch.Equipment.Slots) > 0 && ch.Equipment.Slots[0] != nil {
		weapon := ch.Equipment.Slots[0]
		return SkillResult{
			Success:      true,
			MessageToCh:  fmt.Sprintf("You point %s at %s.\r\n", weapon.GetShortDesc(), target.GetName()),
			MessageToVict: fmt.Sprintf("%s points %s at you.\r\n", ch.Name, weapon.GetShortDesc()),
			MessageToRoom: fmt.Sprintf("%s points %s at %s.\r\n", ch.Name, weapon.GetShortDesc(), target.GetName()),
		}
	}

	return SkillResult{
		Success:      true,
		MessageToCh:  fmt.Sprintf("You point at %s.\r\n", target.GetName()),
		MessageToVict: fmt.Sprintf("%s points at you.\r\n", ch.Name),
		MessageToRoom: fmt.Sprintf("%s points at %s.\r\n", ch.Name, target.GetName()),
	}
}

// DoGroinrip implements do_groinrip() — low blow.
func DoGroinrip(ch *Player, target combat.Combatant, world *World) SkillResult {
	if ch.GetSkill(SkillGroinrip) == 0 {
		return SkillResult{Success: false, MessageToCh: "You're not trained in martial arts!\r\n"}
	}

	percent := rand.Intn(121) + 1 // 0-120; 101+ is complete failure

	// Immortals always succeed
	if ch.GetLevel() > 60 {
		percent = 0
	}

	prob := ch.GetSkill(SkillGroinrip)

	if percent < prob {
		// Success
		dam := ch.GetLevel()
		return SkillResult{
			Success:      true,
			Damage:       dam,
			MessageToCh:  "You grab your victim's groin and twist!\r\n",
			MessageToVict: "You are grabbed in the groin and twisted! The pain is unbearable!\r\n",
			MessageToRoom: fmt.Sprintf("%s falls to %s knees, clutching %s groin and throwing up everywhere!\r\n", target.GetName(), hisHer(ch.GetSex()), hisHer(ch.GetSex())),
		}
	}

	// Miss
	return SkillResult{
		Success:      true,
		Damage:       0,
		MessageToCh:  "You try to grab your victim's groin but miss!\r\n",
		MessageToVict: "$n tries to grab your groin!\r\n",
		MessageToRoom: fmt.Sprintf("%s tries to grab %s's groin!\r\n", ch.Name, target.GetName()),
	}
}

// DoReview implements do_review() — show recent gossip history.
func DoReview(ch *Player) SkillResult {
	// Simple placeholder — returns a message that review was requested
	return SkillResult{
		Success:     true,
		MessageToCh: "Review: (Recent gossip history)\r\n(Review system not yet implemented)\r\n",
	}
}

// DoWhois implements do_whois() — look up player info.
func DoWhois(ch *Player, targetName string) SkillResult {
	if targetName == "" {
		return SkillResult{Success: false, MessageToCh: "For whom do you wish to search?\r\n"}
	}

	return SkillResult{
		Success:     true,
		MessageToCh: fmt.Sprintf("[Looking up %s...]\r\n(Player database lookup not yet connected)\r\n", targetName),
	}
}

// DoPalm implements do_palm() — conceal a small object up your sleeve.
func DoPalm(ch *Player, objName string, world *World) SkillResult {
	if objName == "" {
		return SkillResult{Success: false, MessageToCh: "Palm what?\r\n"}
	}

	// Find item in room
	items := world.GetItemsInRoom(ch.GetRoomVNum())
	var targetItem *ObjectInstance
	targetLower := strings.ToLower(objName)
	for _, item := range items {
		iname := strings.ToLower(item.GetKeywords())
		if strings.Contains(iname, targetLower) {
			targetItem = item
			break
		}
	}

	if targetItem == nil {
		return SkillResult{Success: false, MessageToCh: "You don't see that here.\r\n"}
	}

	// Check weight <= 1 (small object)
	if targetItem.GetWeight() > 1 {
		return SkillResult{Success: false, MessageToCh: "That's too big to palm!\r\n"}
	}

	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillPalm)

	if prob > percent {
		// Success — add to inventory
		world.RemoveItemFromRoom(targetItem, ch.GetRoomVNum())
		_ = ch.Inventory.AddItem(targetItem)
		return SkillResult{
			Success:      true,
			MessageToCh:  "You palm the item skillfully.\r\n",
			MessageToRoom: fmt.Sprintf("%s deftly palms something.\r\n", ch.Name),
		}
	}

	// Failure
	world.RemoveItemFromRoom(targetItem, ch.GetRoomVNum())
	return SkillResult{
		Success:      true,
		MessageToCh:  fmt.Sprintf("You try to palm %s but fumble it!\r\n", targetItem.GetShortDesc()),
		MessageToRoom: fmt.Sprintf("%s fumbles with %s!\r\n", ch.Name, targetItem.GetShortDesc()),
	}
}

// DoFleshAlter implements do_flesh_alter() — transform your hand into a weapon.
func DoFleshAlter(ch *Player) SkillResult {
	if ch.GetSkill(SkillFleshAlter) == 0 {
		return SkillResult{Success: false, MessageToCh: "You know nothing of altering your flesh!\r\n"}
	}

	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillFleshAlter)

	if percent > prob {
		return SkillResult{
			Success:     true,
			MessageToCh: "You lose your concentration!\r\n",
		}
	}

	// Toggle flesh alter state
	return SkillResult{
		Success:      true,
		MessageToCh:  "Your hand turns into a weapon!\r\n",
		MessageToRoom: fmt.Sprintf("%s's hand turns into a weapon!\r\n", ch.Name),
	}
}

// heShe returns "he" / "she" / "it" based on sex.
func heShe(sex int) string {
	switch sex {
	case 1:
		return "he"
	case 2:
		return "she"
	default:
		return "it"
	}
}

// himHer returns "himself" / "herself" / "itself" based on sex.
func himHer(sex int) string {
	switch sex {
	case 1:
		return "himself"
	case 2:
		return "herself"
	default:
		return "itself"
	}
}

// hisHer returns "his" / "her" / "its" based on sex.
func hisHer(sex int) string {
	switch sex {
	case 1:
		return "his"
	case 2:
		return "her"
	default:
		return "its"
	}
}
