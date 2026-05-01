package game

import (
	"fmt"
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

	// C-10: Combat skill constants (from combat_helpers.go)
	SkillDisembowel = "disembowel"
	SkillDragonKick = "dragon_kick"
	SkillTigerPunch = "tiger_punch"
	SkillShoot      = "shoot"
	SkillSubdue     = "subdue"
	SkillSleeper    = "sleeper"
	SkillNeckbreak  = "neckbreak"
	SkillAmbush     = "ambush"
	SkillParry      = "parry"
	SkillEscape     = "escape"
	SkillRetreat    = "retreat"
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
	WaitCh      int    // WAIT_STATE for attacker (PULSE_VIOLENCE ticks)
	WaitTarget  int    // WAIT_STATE for target (PULSE_VIOLENCE ticks)
}

// DoBackstab implements do_backstab() from act.offensive.c lines 172-220.
// Requires: piercing weapon wielded, target not fighting, target awake.
// Damage: weapon damage * backstab multiplier (level*0.2 + 1).
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


// ---------------------------------------------------------------------------
// C-10: Missing combat skill Do* functions — ported from act.offensive.c
// ---------------------------------------------------------------------------

// DoDisembowel implements do_disembowel() from act.offensive.c lines 222-283.
// Requires piercing weapon. Damage: weapon hit.
