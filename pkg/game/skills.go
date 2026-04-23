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
	SkillBash      = "bash"
	SkillKick      = "kick"
	SkillTrip      = "trip"
	SkillRescue    = "rescue"
	SkillSneak     = "sneak"
	SkillHide      = "hide"
	SkillSteal     = "steal"
	SkillPickLock  = "pick_lock"
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
	SkillBash:     combat.PosFighting,
	SkillKick:     combat.PosFighting,
	SkillTrip:     combat.PosFighting,
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

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
