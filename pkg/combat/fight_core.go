// Package combat — fight_core.go
// Port of src/fight.c from the Dark Pawns C codebase.
package combat

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Game-layer hooks
//
// These package-level hooks are used by the legacy skill/spell/DamMessage path
// in fight_core.go. They are wired during initialization:
//   - BroadcastMessage, SendToCharFunc, DoFlee, DoRetreat in pkg/session/manager.go
//   - SkillMessageFunc in pkg/combat/skill_messages.go (via InitSkillMessages)
//   - NowUnix defaults to a zero-returning function below
//   - GetExp is optional; callers must nil-guard it
//
// The primary combat path is CombatEngine struct callbacks (BroadcastFunc,
// MessageFunc, DeathFunc, etc.). These package-level vars remain for legacy
// fight_core functions that do not receive an engine reference.
//
// All hooks are nil-guarded at every call site.
// ---------------------------------------------------------------------------

var (
	BroadcastMessage func(roomVNum int, msg string, exclude string)
	SendToCharFunc   func(name string, msg string)
	DoFlee           func(name string)
	DoRetreat        func(name string)
	SkillMessageFunc func(dam int, ch, vict string, attackType int, roomVNum int) bool
	GetExp           func(name string) int
	NowUnix          = func() int64 { return 0 }
)

// ---------------------------------------------------------------------------
// attack_hit_text — weapon attack names (fight.c:63-81)
// ---------------------------------------------------------------------------

type AttackHitText struct {
	Singular string
	Plural   string
}

var AttackHitTexts = []AttackHitText{
	0:  {"hit", "hits"},
	1:  {"sting", "stings"},
	2:  {"whip", "whips"},
	3:  {"slash", "slashes"},
	4:  {"bite", "bites"},
	5:  {"bludgeon", "bludgeons"},
	6:  {"crush", "crushes"},
	7:  {"pound", "pounds"},
	8:  {"claw", "claws"},
	9:  {"maul", "mauls"},
	10: {"thrash", "thrashes"},
	11: {"pierce", "pierces"},
	12: {"blast", "blasts"},
	13: {"punch", "punches"},
	14: {"stab", "stabs"},
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	LVL_IMMORT  = 31 // C: LVL_IMMORT=31 — duplicated here to avoid import cycle with pkg/game
	NUM_OF_DIRS = 6
	maxExpGain  = 1000000
)

const (
	SPELL_INVISIBLE    = 1
	SKILL_BACKSTAB     = 100
	SKILL_CIRCLE       = 101
	SKILL_KICK         = 102
	SKILL_BASH         = 103
	SKILL_PUNCH        = 104
	SKILL_DRAGON_KICK  = 105
	SKILL_TIGER_PUNCH  = 106
	SKILL_HEADBUTT     = 107
	SKILL_SMACKHEADS   = 108
	SKILL_SLUG         = 109
	SKILL_SERPENT_KICK = 110
	SKILL_BITE         = 111
	SKILL_DISEMBOWEL   = 112
	SKILL_NECKBREAK    = 113
	SKILL_RETREAT      = 114
	SKILL_ESCAPE       = 115
	SKILL_PARRY        = 116
	SKILL_DODGE        = 117
)

const (
	AFF_INVISIBLE    = 1
	AFF_HIDE         = 2
	AFF_SLEEP        = 3
	AFF_CHARM        = 4
	AFF_SANCTUARY    = 5
	AFF_PROTECT_EVIL = 6
	AFF_PROTECT_GOOD = 7
	AFF_GROUP        = 8
	AFF_HASTE        = 9
	AFF_SLOW         = 10
)

const (
	AFF_STR_GROUP     = "AFF_GROUP"
	AFF_STR_WEREWOLF  = "AFF_WEREWOLF"
	AFF_STR_VAMPIRE   = "AFF_VAMPIRE"
	AFF_STR_FLESH_ALT = "AFF_FLESH_ALTER"
	AFF_STR_HASTE     = "AFF_HASTE"
	AFF_STR_SLOW      = "AFF_SLOW"
)

const (
	TYPE_UNDEFINED = 0
	TYPE_HIT       = 300
	TYPE_BLUDGEON  = TYPE_HIT + 5
	TYPE_POUND     = TYPE_HIT + 7
	TYPE_PUNCH     = TYPE_HIT + 13
	TYPE_BITE      = TYPE_HIT + 4
	TYPE_CLAW      = TYPE_HIT + 8
	TYPE_SLASH     = TYPE_HIT + 3
	TYPE_CRUSH     = TYPE_HIT + 6
	TYPE_MAUL      = TYPE_HIT + 9
	TYPE_THRASH    = TYPE_HIT + 10
	TYPE_PIERCE    = TYPE_HIT + 11
	TYPE_STAB      = TYPE_HIT + 14
	TYPE_WHIP      = TYPE_HIT + 2
	TYPE_BLAST     = TYPE_HIT + 12
	TYPE_SUFFERING = 399
)

const (
	RACE_UNDEAD  = 3
	RACE_VAMPIRE = 8
)

// fleshAlteredType returns the attack_hit_text index for unarmed NPCs with
// AFF_FLESH_ALTER based on mob level.
// Ported from flesh_altered_type() in new_cmds.c:1783.
// Maps level ranges to attack type indices matching the C implementation.
// Note: if attack_hit_text is changed, this function must be updated too.
func fleshAlteredType(level int) int {
	switch {
	case level <= 3:
		return 7 // pound
	case level <= 6:
		return 11 // pierce
	case level <= 9:
		return 3 // slash
	case level <= 15:
		return 7 // pound
	case level <= 21:
		return 3 // slash
	case level <= 24:
		return 7 // pound
	default:
		return 3 // slash (level >= 25)
	}
}

// **********************************
// 1. appear()
// **********************************

func Appear(ch Combatant) {
	msg := fmt.Sprintf("%s slowly fades into existence.", ch.GetName())
	if ch.GetLevel() >= LVL_IMMORT {
		msg = fmt.Sprintf("You feel a strange presence as %s appears, seemingly from nowhere.", ch.GetName())
	}
	if BroadcastMessage != nil {
		BroadcastMessage(ch.GetRoom(), msg, ch.GetName())
	}
}

// **********************************
// 2. updatePos()
// **********************************

func GetPositionFromHP(hp, currentPos int) int {
	if hp > 0 {
		if currentPos > PosStunned {
			return currentPos
		}
		return PosStanding
	}
	if hp <= -11 {
		return PosDead
	}
	if hp <= -6 {
		return PosMortally
	}
	if hp <= -3 {
		return PosIncap
	}
	return PosStunned
}

// **********************************
// 4. deathCry()
// **********************************

func DeathCry(ch Combatant) string {
	roomVNum := ch.GetRoom()
	msg := fmt.Sprintf("Your blood freezes as you hear %s's death cry.", ch.GetName())
	if BroadcastMessage != nil {
		BroadcastMessage(roomVNum, msg, "")
	}
	return fmt.Sprintf("%d", roomVNum)
}

// **********************************
// 5. takeDamage()
// **********************************

func TakeDamage(ch, victim Combatant, dam int, attackType int) bool {
	if victim.GetPosition() <= PosDead {
		return false
	}
	chName := ch.GetName()
	victimName := victim.GetName()
	roomVNum := ch.GetRoom()

	if ch.GetRoom() != victim.GetRoom() {
		return false
	}

	if victimName != chName && !ch.IsNPC() && !victim.IsNPC() {
		if ch.GetLevel() <= 10 {
			return false
		}
		if victim.GetLevel() <= 10 {
			return false
		}
	}

	if victimName != chName && ch.GetPosition() > PosStunned {
		if ch.GetFighting() == "" {
			ch.SetFighting(victimName)
		}
		if victim.GetPosition() > PosStunned && victim.GetFighting() == "" {
			victim.SetFighting(chName)
		}
	}

	if !victim.IsNPC() && victim.GetLevel() >= LVL_IMMORT {
		dam = 0
	}

	if dam > 3000 {
		dam = 3000
	}
	if dam < 0 {
		dam = 0
	}

	victim.TakeDamage(dam)

	newPos := GetPositionFromHP(victim.GetHP(), victim.GetPosition())

	isWeapon := attackType >= TYPE_HIT && attackType < TYPE_SUFFERING
	if !isWeapon {
		if SkillMessageFunc != nil {
			SkillMessageFunc(dam, chName, victimName, attackType, ch.GetRoom())
		}
	} else {
		if newPos == PosDead || dam == 0 {
			sent := false
			if SkillMessageFunc != nil {
				sent = SkillMessageFunc(dam, chName, victimName, attackType, ch.GetRoom())
			}
			if !sent {
				DamMessage(dam, ch, victim, attackType-TYPE_HIT)
			}
		} else {
			DamMessage(dam, ch, victim, attackType-TYPE_HIT)
		}
	}

	switch newPos {
	case PosMortally:
		victim.SendMessage("You are mortally wounded, and will die soon, if not aided.\r\n")
		if BroadcastMessage != nil {
			BroadcastMessage(ch.GetRoom(),
				fmt.Sprintf("%s is mortally wounded, and will die soon, if not aided.", victimName), "")
		}
	case PosIncap:
		victim.SendMessage("You are incapacitated and will slowly die, if not aided.\r\n")
		if BroadcastMessage != nil {
			BroadcastMessage(ch.GetRoom(),
				fmt.Sprintf("%s is incapacitated and will slowly die, if not aided.", victimName), "")
		}
	case PosStunned:
		victim.SendMessage("You're stunned, but will probably regain consciousness again.\r\n")
		if BroadcastMessage != nil {
			BroadcastMessage(ch.GetRoom(),
				fmt.Sprintf("%s is stunned, but will probably regain consciousness again.", victimName), "")
		}
	case PosDead:
		victim.SendMessage("You are dead!  Sorry...\r\n")
		if BroadcastMessage != nil {
			BroadcastMessage(roomVNum, fmt.Sprintf("%s is dead!  R.I.P.", victimName), "")
		}
	default:
		if dam > victim.GetMaxHP()/4 {
			victim.SendMessage("That really did HURT!\r\n")
		}
		if victim.GetHP() < victim.GetMaxHP()/4 {
			victim.SendMessage("You wish that your wounds would stop BLEEDING so much!\r\n")
			if DoFlee != nil {
				DoFlee(victimName)
			}
		}
	}

	if newPos < PosSleeping && victim.GetFighting() != "" {
		victim.StopFighting()
	}

	if newPos == PosDead {
		if victim.IsNPC() {
			if IsInGroup(ch) {
				GroupGain(ch, victim)
			} else {
				exp := 0
				if GetExp != nil {
					exp = GetExp(victimName)
				}
				if exp > maxExpGain {
					exp = maxExpGain
				}
				exp = CalcLevelDiff(ch, victim, exp)

				if exp > 1 {
					ch.SendMessage(fmt.Sprintf("You receive %d experience points.\r\n", exp))
				} else {
					ch.SendMessage("You receive one lousy experience point.\r\n")
				}
			}
		}

		CounterProcs(ch)
		DieWithKiller(victim, ch, attackType)
	}

	if dam > 0 {
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Skill message functions for DamMessage
// ---------------------------------------------------------------------------

// damMessageTier describes one entry of the weapon damage message table.
// CRIT-010: In the original C code, messages were loaded from data files via
// Each tier has multiple variants; one is randomly selected per hit.
// This restores the C CircleMUD behavior from load_messages() + misc/messages,
// where 3-4 variants per tier kept combat feeling fresh.
type damMessageTier struct {
	MinDamage int
	Room      []string
	Char      []string
	Victim    []string
}

// randPick selects a random element from a slice.
func randPick[T any](s []T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	return s[GetRoller().IntN(len(s))]
}

var damMessageTiers = []damMessageTier{
	// Tier 0: miss
	{
		0,
		[]string{
			"$n tries to #w $N, but misses.",
			"$n's swing goes wide, missing $N entirely.",
			"$n lunges at $N and connects with nothing but air.",
		},
		[]string{
			"You try to #w $N, but miss.",
			"You swing at $N and hit nothing.",
			"Your clumsy strike goes wide of $N.",
		},
		[]string{
			"$n tries to #w you, but misses.",
			"$n's attack sails past you harmlessly.",
			"$n lunges at you but can't find the range.",
		},
	},
	// Tier 1: scratch (1-2)
	{
		1,
		[]string{
			"$n scratches $N as $e #W $M.",
			"$n grazes $N with a glancing blow.",
		},
		[]string{
			"You scratch $N as you #w $M.",
			"You clip $N — barely worth the effort.",
		},
		[]string{
			"$n scratches you as $e #W you.",
			"$n's strike just barely catches you.",
		},
	},
	// Tier 2: barely (3-4)
	{
		3,
		[]string{
			"$n barely #W $N.",
			"$n's feeble blow barely lands on $N.",
		},
		[]string{
			"You barely #w $N.",
			"You land a pathetic blow on $N.",
		},
		[]string{
			"$n barely #W you.",
			"$n's weak strike barely registers.",
		},
	},
	// Tier 3: light (5-6)
	{
		5,
		[]string{
			"$n #W $N.",
			"$n lands a light blow on $N.",
		},
		[]string{
			"You #w $N.",
			"You land a light hit on $N.",
		},
		[]string{
			"$n #W you.",
			"$n hits you with a light blow.",
		},
	},
	// Tier 4: hard (7-10)
	{
		7,
		[]string{
			"$n #W $N hard.",
			"$n's solid blow catches $N flush.",
			"$n connects firmly with $N.",
		},
		[]string{
			"You #w $N hard.",
			"Your solid strike catches $N flush.",
			"You connect firmly with $N.",
		},
		[]string{
			"$n #W you hard.",
			"$n's solid blow catches you flush.",
			"$n connects firmly with you.",
		},
	},
	// Tier 5: very hard (11-17)
	{
		11,
		[]string{
			"$n #W $N very hard.",
			"$n's heavy strike staggers $N.",
		},
		[]string{
			"You #w $N very hard.",
			"Your heavy blow staggers $N.",
		},
		[]string{
			"$n #W you very hard.",
			"$n's heavy strike staggers you.",
		},
	},
	// Tier 6: extremely hard (18-25)
	{
		18,
		[]string{
			"$n #W $N extremely hard.",
			"$n wallops $N with bone-rattling force!",
		},
		[]string{
			"You #w $N extremely hard.",
			"You wallop $N with bone-rattling force!",
		},
		[]string{
			"$n #W you extremely hard.",
			"$n wallops you with bone-rattling force!",
		},
	},
	// Tier 7: violently (26-35) — C version: "massacres $N to small fragments"
	{
		26,
		[]string{
			"$n #W $N violently.",
			"$n massacres $N to small fragments with $s #w!",
		},
		[]string{
			"You #w $N violently.",
			"You massacre $N to small fragments with your #w!",
		},
		[]string{
			"$n #W you violently.",
			"$n massacres you to small fragments with $s #w!",
		},
	},
	// Tier 8: savagely (36-47) — C version: "OBLITERATES $N with deadly #w"
	{
		36,
		[]string{
			"$n #W $N savagely.",
			"$n OBLITERATES $N with $s deadly #w!!",
		},
		[]string{
			"You #w $N savagely.",
			"You OBLITERATE $N with your deadly #w!!",
		},
		[]string{
			"$n #W you savagely.",
			"$n OBLITERATES you with $s deadly #w!!",
		},
	},
	// Tier 9: MUTILATES (48-59) — C version: "EVISCERATES $N with incredible #w"
	{
		48,
		[]string{
			"$n MUTILATES $N!",
			"$n EVISCERATES $N with $s incredible #w!!",
		},
		[]string{
			"You MUTILATE $N!",
			"You EVISCERATE $N with your incredible #w!!",
		},
		[]string{
			"$n MUTILATES you!",
			"$n EVISCERATES you with $s incredible #w!!",
		},
	},
	// Tier 10: DISEMBOWELS (60-79)
	{
		60,
		[]string{
			"$n DISEMBOWELS $N!!",
			"$n opens $N up like a gutted fish!!",
		},
		[]string{
			"You DISEMBOWEL $N!!",
			"You open $N up like a gutted fish!!",
		},
		[]string{
			"$n DISEMBOWELS you!!",
			"$n opens you up like a gutted fish!!",
		},
	},
	// Tier 11: DESTROYS (80-100)
	{
		80,
		[]string{
			"$n DESTROYS $N!!!",
			"$n annihilates $N with a devastating blow!!!",
		},
		[]string{
			"You DESTROY $N!!!",
			"You annihilate $N with a devastating blow!!!",
		},
		[]string{
			"$n DESTROYS you!!!",
			"$n annihilates you with a devastating blow!!!",
		},
	},
	// Tier 12: OBLITERATES (101-9999)
	{
		101,
		[]string{
			"$n OBLITERATES $N!!!!",
			"$n reduces $N to a bloody smear on the ground!!!!",
			"$n delivers a blow of legend against $N!!!!",
		},
		[]string{
			"You OBLITERATE $N!!!!",
			"You reduce $N to a bloody smear on the ground!!!!",
			"You deliver a blow of legend against $N!!!!",
		},
		[]string{
			"$n OBLITERATES you!!!!",
			"$n reduces you to a bloody smear on the ground!!!!",
			"$n delivers a blow of legend against you!!!!",
		},
	},
	// Tier 13: ROCK (10000+)
	{
		10000,
		[]string{
			"$n R O C K S the Hell Out Of $N!!!!!!!!!!!!!!!!!!!!!!!!",
			"$n delivers a blow so catastrophic that reality itself flinches!!!!!!!!!!",
		},
		[]string{
			"You R O C K the Hell Out Of $N!!!!!!!!!!!!!!!!!!!!!!!!",
			"You deliver a blow so catastrophic that reality itself flinches!!!!!!!!!!",
		},
		[]string{
			"$n R O C K S the Hell Out Of You!!!!!!!!!!!!!!!!!!!!!!!!",
			"$n delivers a blow so catastrophic that reality itself flinches!!!!!!!!!!",
		},
	},
}

// DamMessage sends the appropriate damage message for weapon attacks.
func DamMessage(dam int, ch, victim Combatant, attackType int) {
	var tier *damMessageTier
	for i := len(damMessageTiers) - 1; i >= 0; i-- {
		if dam >= damMessageTiers[i].MinDamage {
			tier = &damMessageTiers[i]
			break
		}
	}
	if tier == nil {
		return
	}

	singular := AttackHitTexts[attackType].Singular
	plural := AttackHitTexts[attackType].Plural

	sex := ch.GetSex()
	roomMsg := replaceMessageTokens(randPick(tier.Room), ch.GetName(), victim.GetName(), singular, plural, sex)
	charMsg := replaceMessageTokens(randPick(tier.Char), ch.GetName(), victim.GetName(), singular, plural, sex)
	victimMsg := replaceMessageTokens(randPick(tier.Victim), ch.GetName(), victim.GetName(), singular, plural, sex)

	if BroadcastMessage != nil {
		BroadcastMessage(ch.GetRoom(), roomMsg, ch.GetName()+" "+victim.GetName())
	}

	// CRIT-010: Send attacker and victim their own messages.
	// In C, act() sent to TO_CHAR and TO_VICT separately.
	if SendToCharFunc != nil {
		SendToCharFunc(ch.GetName(), charMsg)
		SendToCharFunc(victim.GetName(), victimMsg)
	}
}

// replaceMessageTokens substitutes $n, $N, $e, #w, #W in a message template.
// sex follows C SEX_* constants: 0=male, 1=female, 2=neutral/other.
func replaceMessageTokens(msg, chName, victimName, singular, plural string, sex int) string {
	var subject, object, possessive string
	switch sex {
	case 1:
		subject, object, possessive = "she", "her", "her"
	case 0:
		subject, object, possessive = "he", "him", "his"
	default:
		subject, object, possessive = "it", "it", "its"
	}
	result := msg
	result = strings.ReplaceAll(result, "$n", chName)
	result = strings.ReplaceAll(result, "$N", victimName)
	result = strings.ReplaceAll(result, "$e", subject)
	result = strings.ReplaceAll(result, "$E", object)
	result = strings.ReplaceAll(result, "$s", possessive)
	result = strings.ReplaceAll(result, "$m", chName)
	result = strings.ReplaceAll(result, "$M", victimName)
	result = strings.ReplaceAll(result, "#w", singular)
	result = strings.ReplaceAll(result, "#W", plural)
	return result
}

// BackstabMult mirrors backstab_mult() from src/class.c lines 720-729.
// Shared by combat/fight_core.go (circle/disembowel) and game/skill_combat.go (DoBackstab).
func BackstabMult(level int) float64 {
	if level <= 0 {
		return 1.0
	}
	if level >= LVL_IMMORT {
		return 20.0
	}
	return float64(level)*0.2 + 1.0
}

// **********************************
// 7. groupGain / calcLevelDiff / performGroupGain
// **********************************

func IsInGroup(ch Combatant) bool {
	return false
}

func CalcLevelDiff(ch, victim Combatant, base int) int {
	levelDiff := ch.GetLevel() - victim.GetLevel()
	share := base
	if share > maxExpGain {
		share = maxExpGain
	}
	if share < 1 {
		share = 1
	}
	if levelDiff > 0 {
		if !IsInGroup(ch) {
			levelDiff -= 2
		}
		switch {
		case levelDiff > 15:
			share -= int(float64(share) * 0.7)
		case levelDiff > 10:
			share -= int(float64(share) * 0.5)
		case levelDiff > 5:
			share -= int(float64(share) * 0.3)
		}
	}
	if ch.GetLevel() > 20 {
		share -= int(float64(share) * 0.2)
	}
	if share < 1 {
		share = 1
	}
	return share
}

func PerformGroupGain(ch, victim Combatant, base int) {
	share := CalcLevelDiff(ch, victim, base)
	if share > 1 {
		ch.SendMessage(fmt.Sprintf("You receive your share of experience -- %d points.\r\n", share))
	} else {
		ch.SendMessage("You receive your share of experience -- one measly little point!\r\n")
	}
}

func GroupGain(ch, victim Combatant) {
	victimExp := 0
	if GetExp != nil {
		victimExp = GetExp(victim.GetName())
	}
	base := victimExp
	if base > 100 {
		base -= int(float64(base) * 0.01)
	}
	if base < 1 {
		base = 1
	}
	PerformGroupGain(ch, victim, base)
}

// **********************************
// 8. rawKill()
// **********************************

func RawKill(ch Combatant, attackType int) {
	if ch.GetRoom() < 0 {
		return
	}
	if ch.GetFighting() != "" {
		ch.StopFighting()
	}
	DeathCry(ch)
}

// **********************************
// 9. dieWithKiller()
// **********************************

func DieWithKiller(ch, killer Combatant, attackType int) {
	RawKill(ch, attackType)
}

// **********************************
// 10. die()
// **********************************

func Die(ch Combatant) {
	RawKill(ch, TYPE_UNDEFINED)
}

// **********************************
// 11. makeCorpse()
// **********************************

func MakeCorpse(victim Combatant, attackType int) {}

// **********************************
// 12. makeDust()
// **********************************

func MakeDust(victim Combatant, attackType int) {}

// **********************************
// 13. counterProcs()
// **********************************

func CounterProcs(ch Combatant) {}

// **********************************
// 14. attitudeLoot()
//
// Port of fight.c:attitude_loot. The legacy hook-driven loot/brag logic is
// removed; the engine path handles corpse creation and looting via struct
// callbacks and the game layer.
// **********************************

func AttitudeLoot(ch, victim Combatant) {}

// **********************************
// 15. damMessage()
// **********************************

// **********************************
// 16. skillMessage()
// **********************************

func SkillMessage(dam int, ch, victim Combatant, attackType int) {
	if attackType < TYPE_HIT {
		if SkillMessageFunc != nil {
			SkillMessageFunc(dam, ch.GetName(), victim.GetName(), attackType, ch.GetRoom())
		}
	}
}

// **********************************
// 21. stopFighting / setFighting helpers
// **********************************

func NewNamedCombatant(name string, roomVNum int) Combatant {
	return &namedCombatant{name: name, room: roomVNum, isNPC: false}
}

type namedCombatant struct {
	name  string
	room  int
	isNPC bool
}

func (n *namedCombatant) GetName() string           { return n.name }
func (n *namedCombatant) IsNPC() bool               { return n.isNPC }
func (n *namedCombatant) GetRoom() int              { return n.room }
func (n *namedCombatant) GetLevel() int             { return 0 }
func (n *namedCombatant) GetHP() int                { return 0 }
func (n *namedCombatant) GetMaxHP() int             { return 0 }
func (n *namedCombatant) GetAC() int                { return 0 }
func (n *namedCombatant) GetTHAC0() int             { return 0 }
func (n *namedCombatant) GetDamageRoll() DiceRoll   { return DiceRoll{} }
func (n *namedCombatant) GetPosition() int          { return PosStanding }
func (n *namedCombatant) SetPosition(pos int)       {}
func (n *namedCombatant) GetClass() int             { return 0 }
func (n *namedCombatant) GetStr() int               { return 0 }
func (n *namedCombatant) GetStrAdd() int            { return 0 }
func (n *namedCombatant) GetDex() int               { return 0 }
func (n *namedCombatant) GetInt() int               { return 0 }
func (n *namedCombatant) GetWis() int               { return 0 }
func (n *namedCombatant) GetHitroll() int           { return 0 }
func (n *namedCombatant) GetDamroll() int           { return 0 }
func (n *namedCombatant) GetSex() int               { return 1 }
func (n *namedCombatant) GetMaster() string         { return "" }
func (n *namedCombatant) TakeDamage(amount int)     {}
func (n *namedCombatant) Heal(amount int)           {}
func (n *namedCombatant) SetFighting(target string) {}
func (n *namedCombatant) StopFighting()             {}
func (n *namedCombatant) GetFighting() string       { return "" }
func (n *namedCombatant) SendMessage(msg string)    {}
func (n *namedCombatant) GetSendMessage(msg string) {}
