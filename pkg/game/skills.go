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
	SkillScrounge = "scrounge"

	// act.other.c skills
	SkillPeek        = "peek"
	SkillStealth     = "stealth"
	SkillAppraise    = "appraise"
	SkillScout       = "scout"
	SkillFirstAid    = "first_aid"
	SkillDisarm      = "disarm"
	SkillMindlink    = "mindlink"
	SkillDetect      = "detect"
	SkillSerpentKick = "serpent_kick"
	SkillDig         = "dig"
	SkillTurn        = "turn"

	// Wave 1 cleanup skills (new_cmds.c)
	SkillMold       = "mold"
	SkillBehead     = "behead"
	SkillBearhug    = "bearhug"
	SkillSlug       = "slug"
	SkillSmackheads = "smackheads"
	SkillBite       = "bite"
	SkillTag        = "tag"
	SkillPoint      = "point"
	SkillGroinrip   = "groinrip"
	SkillReview     = "review"
	SkillWhois      = "whois"
	SkillPalm       = "palm"
	SkillFleshAlter = "flesh_alter"
	SkillKabuki     = "kabuki" // src/act.other.c do_hide SCMD_KABUKI — distinct from SkillHide
	SkillShadow     = "shadow" // src/act.movement.c do_follow SCMD shadow (the quiet-follow skill)

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

	// Fidelity port gaps (new_cmds.c)
	SkillSpike  = "spike"
	SkillStake  = "stake"
	SkillCircle = "circle"
	SkillCharge = "charge"
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
	SkillCircle: {
		ClassThief:    15,
		ClassAssassin: 8,
	},
	SkillCharge: {
		ClassWarrior: 23,
		ClassPaladin: 23,
		ClassRanger:  22,
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
	SkillCircle:   combat.PosFighting,
	SkillCharge:   combat.PosFighting,
}

// ---------------------------------------------------------------------------
// Helper: check if player can use a skill
// ---------------------------------------------------------------------------

// SkillUnknownMsg holds the byte-exact message a Wave-1 combat command sends
// when the actor does not know the skill. C gates skill USE purely on
// `if (!GET_SKILL(ch, SKILL_X))` (act.offensive.c / new_cmds.c), emitting each
// command's own hardcoded line — there is no class check and no generic level
// message. These are ported per-command from the C do_ functions (R1/R4, DP-1206).
//
// Each message carries its OWN C-exact terminator — usually "\r\n", but
// do_cutthroat (new_cmds.c:561) and do_slug (new_cmds2.c:829) use "\n\r".
// Handlers send the message AS-IS (SendMessage(msg), no append): the oracle
// normalizes terminators so a uniform "+\r\n" append passed green while
// silently rewriting those two "\n\r" quirks (the cutthroat #532/#537 and slug
// #535 findings). Storing the terminator per-message kills that bug class.
// Backstab's message is "You have no idea how." (verified at act.offensive.c:174,
// the registered do_backstab) — NOT the "sneaky stuff" line (that's do_trip).
// Disarm: CmdDisarm never calls CanUseSkill and DoDisarm already gates faithfully
// on GetSkill==0 with the byte-exact message (skills2.go:156).
var SkillUnknownMsg = map[string]string{
	SkillBackstab:   "You have no idea how.\r\n",
	SkillBash:       "You'd better leave all the martial arts to fighters.\r\n",
	SkillKick:       "You'd better leave all the martial arts to fighters.\r\n",
	SkillTrip:       "You'd better leave the sneaky stuff to the thieves.\r\n",
	SkillHeadbutt:   "You aren't qualified to headbutt anyone!\r\n",
	SkillRescue:     "But only true warriors can do this!\r\n",
	SkillDisarm:     "You'd better leave all the martial arts to fighters.\r\n",
	SkillCharge:     "You couldn't charge if you wanted to!\r\n",
	SkillDisembowel: "You have no idea how.\r\n",
	SkillCutthroat:  "You're not trained in slitting throats!\n\r", // C: \n\r (new_cmds.c:561)
	SkillNeckbreak:  "What's that, idiot-san?\r\n",
	SkillSlug:       "You couldn't slug your way out of a wet paper bag.\n\r", // C: \n\r (new_cmds2.c:829)
	SkillStrike:     "Yeah, right.\r\n",
}

// CanUseSkill checks whether a player can use a skill. For Wave-1 combat skills
// (those in SkillUnknownMsg) it gates faithfully on GetSkill()==0 with the
// command's exact C message — no class or level check (DP-1206). For all other
// skills it keeps the legacy class/level/position behavior verbatim until their
// Wave-2 audit. The position block is shared by both paths and is unchanged
// (position bytes/ordering is a separate finding).
func CanUseSkill(p *Player, skillName string) (bool, string) {
	if msg, audited := SkillUnknownMsg[skillName]; audited {
		// FAITHFUL path (DP-1206): C gates on !GET_SKILL, no class/level.
		if p.GetSkill(skillName) == 0 {
			return false, msg
		}
		return skillPositionGate(p, skillName)
	}

	// LEGACY path — un-audited skills keep today's class/level behavior verbatim.
	classReqs, ok := SkillClassReq[skillName]
	if !ok {
		return false, "You have no idea how.\r\n"
	}

	minLevel, classOk := classReqs[p.Class]
	if !classOk {
		return false, "You have no idea how.\r\n"
	}
	if p.Level < minLevel {
		return false, fmt.Sprintf("You must be at least level %d to use that skill.\r\n", minLevel)
	}

	return skillPositionGate(p, skillName)
}

// skillPositionGate enforces the skill's minimum position and returns the
// existing (unchanged) position messages. Shared by both CanUseSkill paths so
// the audited fork does not perturb position bytes (a separate finding).
func skillPositionGate(p *Player, skillName string) (bool, string) {
	minPos := SkillPosReq[skillName]
	if p.GetPosition() < minPos {
		switch minPos {
		case combat.PosStanding:
			return false, "You must be standing to do that.\r\n"
		case combat.PosFighting:
			return false, "You must be fighting to do that!\r\n"
		default:
			return false, "You can't do that right now.\r\n"
		}
	}
	return true, ""
}

// ---------------------------------------------------------------------------
// Target finding helpers
// ---------------------------------------------------------------------------

// FindTargetInRoom finds a character (player or mob) in the same room. It is a
// thin wrapper over World.ResolveCharInRoom (DP-907) so every command that
// takes an in-room character target — consider, kick, backstab, bash, trip,
// rescue, steal, ... — resolves through the single canonical get_char_room_vis
// port (keyword-list abbreviation matching, ordinals like 2.guard, self/me,
// CAN_SEE). Before DP-907 this used a substring match against the ShortDesc,
// which is why `consider postman` and `kick postman` disagreed in the same
// room.
//
// `exclude` is retained for signature compatibility and supplies the viewer
// for visibility checks. `roomVNum` is taken from `ch.GetRoom()` by
// ResolveCharInRoom, so the explicit room is only authoritative when it
// matches ch's room; this matches every existing caller.
func FindTargetInRoom(world *World, roomVNum int, targetName string, exclude *Player) (combat.Combatant, string, bool) {
	ch := exclude
	if ch == nil {
		// No viewer supplied: fall back to a match without visibility/ordinal
		// semantics by using a throwaway viewer located in roomVNum. This path
		// is not used by any current caller (all pass ch), but keeps the
		// function safe if it ever is.
		ch = &Player{}
		ch.RoomVNum = roomVNum
	}
	if ch.RoomVNum == 0 {
		// A zero-valued RoomVNum (unset) would resolve against room 0; honour
		// the explicit roomVNum the caller passed instead.
		ch.RoomVNum = roomVNum
	}
	tgt, ok := world.ResolveCharInRoom(ch, targetName)
	if !ok {
		return nil, "", false
	}
	return tgt.Combatant, charDisplayName(tgt), true
}

// charDisplayName returns the player name or mob ShortDesc for a resolved
// target — the second return value FindTargetInRoom historically carried.
func charDisplayName(t CharTarget) string {
	switch {
	case t.Player != nil:
		return t.Player.Name
	case t.Mob != nil:
		return t.Mob.GetShortDesc()
	}
	return ""
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
// Sex: 0 = male, 1 = female, 2 = neutral (matching Player.Sex / MobInstance.GetSex)
func GetPronouns(name string, sex int) Pronouns {
	var he, him, his string
	switch sex {
	case 1: // female
		he, him, his = "she", "her", "her"
	case 0: // male
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
	Success       bool
	Damage        int
	MessageToCh   string
	MessageToVict string
	MessageToRoom string
	StunTarget    bool // target loses a round
	SleepTarget   bool // target is put to sleep (PosSleeping + AFF_SLEEP)
	SelfStumble   bool // user falls (bash fail)
	TargetFalls   bool // target position changes to sitting
	WaitCh        int  // WAIT_STATE for attacker (PULSE_VIOLENCE ticks)
	WaitTarget    int  // WAIT_STATE for target (PULSE_VIOLENCE ticks)
	// StartCombat signals the caller to initiate combat even when the skill
	// deals no damage (miss / zero-damage hit). C: skills like backstab call
	// damage(ch, vict, 0, SKILL) on a miss, which starts combat via set_fighting.
	// The caller (sendSkillResult) routes this through the combat engine.
	StartCombat bool

	// SkillMsgType, when non-zero, routes the combat message through the
	// skill_message path (fight.c:1023-1092) instead of emitting MessageToCh/
	// Vict/Room directly. It holds the lib/misc/messages attack-type key (e.g.
	// 131 for the Backstab set) — NOT the Go-internal SKILL_* enum. When set,
	// the caller (sendSkillResult) draws Dice(1,N) and emits the selected set's
	// char/vict/room text via the combat engine, mirroring C's damage() path,
	// and MessageToCh/Vict/Room are ignored. R4 (no invented strings) + R3
	// (the Dice draw must happen in order).
	SkillMsgType int

	// DeferredImprove lists the skills to run improveSkill() on AFTER the
	// skill_message/damage step, matching C's order (skill_message draws its
	// dice inside damage()/hit(), THEN improve_skill runs). Ordered; repeat an
	// entry for a skill C improves twice (headbutt). DP-1212 / R3b.
	DeferredImprove []string
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
