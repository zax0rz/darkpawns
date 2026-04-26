// Package combat — fight_core.go
// Port of src/fight.c from the Dark Pawns C codebase.
package combat

import (
	"fmt"
	"math/rand"
	"strings"
)

// ---------------------------------------------------------------------------
// Game-layer hooks
// ---------------------------------------------------------------------------

var (
	BroadcastMessage            func(roomVNum int, msg string, exclude string)
	SkillMessageFunc            func(dam int, ch, vict string, attackType int) bool
	GainExp                     func(name string, amount int)
	ExtractChar                 func(name string)
	MakeCorpseFunc              func(victim string, attackType int)
	MakeDustFunc                func(victim string, attackType int)
	LogMessage                  func(msg string, level string, minLevel int, toLog bool)
	IsShopkeeper                func(name string) bool
	GetRace                     func(name string) int
	GetRaceHate                 func(name string, index int) int
	HasAffect                   func(name string, aff int) bool
	HasAffectStr                func(name string, aff string) bool
	RemoveAffect                func(name string, skillNum int)
	RemoveAllAffects            func(name string)
	RunDeathScript              func(killer, victim string, roomVNum int)
	RunFightScript              func(mob, target string, roomVNum int)
	HasScriptFlag               func(name string, flag string) bool
	HasMobFlag                  func(name string, flag string) bool
	HasMobVNum                  func(name string, vnum int) bool
	HasRoomFlag                 func(roomVNum int, flag string) bool
	HasPrfFlag                  func(name string, flag string) bool
	HasPlrFlag                  func(name string, flag string) bool
	SetPlrFlag                  func(name string) bool
	IsMounted                   func(name string) bool
	Dismount                    func(name string)
	GetWimpyLev                 func(name string) int
	GetSkill                    func(name string, skillNum int) int
	DoFlee                      func(name string)
	DoRetreat                   func(name string)
	GetKills                    func(name string) int64
	SetKills                    func(name string, kills int64)
	GetDeaths                   func(name string) int64
	SetDeaths                   func(name string, deaths int64)
	SetLastDeath                func(name string, t int64)
	GetConstitution             func(name string) int
	SetConstitution             func(name string, val int)
	GetPks                      func(name string) int64
	SetPks                      func(name string, pks int64)
	Unmount                     func(name string)
	GetAlignment                func(name string) int
	SetAlignment                func(name string, val int)
	GetExp                      func(name string) int
	BuildTHAC0                  func(class, level int) int
	GetNPCData                  func(name string) (attackType int, damDice, damSize int)
	GetWeaponInfo               func(chName string) (wType, damDice, damSize int, isBlessed bool)
	GetMobAC                    func(name string) int
	GetAdjacentRoom             func(roomVNum, door int) int
	GetFollowersInRoom          func(name string, roomVNum int) int
	GetMasterInRoom             func(name string, roomVNum int) bool
	GetFellowFollowersInRoom    func(name string, roomVNum int) bool
	CountGroupMembers           func(leaderName string, roomVNum int) int
	ApplyToGroupMembers         func(leaderName string, roomVNum int, fn func(name string))
	PerformCommand              func(chName, cmd string)
	BroadChatFunc               func(chName string, msg string)
	IsInRoom                    func(name string, roomVNum int) bool
	IncreaseMaxStat              func(name string, stat string) // "hp", "mana", or "move"
	HealAllPlayers               func()                     // Heal all connected players to full
	GetGold                      func(name string) int
	SetGold                      func(name string, gold int)
)

// ---------------------------------------------------------------------------
// attack_hit_text — weapon attack names (fight.c:63-81)
// ---------------------------------------------------------------------------

type AttackHitText struct {
	Singular string
	Plural   string
}

var AttackHitTexts = []AttackHitText{
	0: {"hit", "hits"},
	1: {"sting", "stings"},
	2: {"whip", "whips"},
	3: {"slash", "slashes"},
	4: {"bite", "bites"},
	5: {"bludgeon", "bludgeons"},
	6: {"crush", "crushes"},
	7: {"pound", "pounds"},
	8: {"claw", "claws"},
	9: {"maul", "mauls"},
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
	LVL_IMMORT  = 31
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
)

const (
	AFF_STR_GROUP     = "AFF_GROUP"
	AFF_STR_WEREWOLF  = "AFF_WEREWOLF"
	AFF_STR_VAMPIRE   = "AFF_VAMPIRE"
	AFF_STR_FLESH_ALT = "AFF_FLESH_ALTER"
)

const (
	TYPE_UNDEFINED = 0
	TYPE_HIT       = 2000
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
	TYPE_SUFFERING = 3000
)

const (
	RACE_UNDEAD  = 3
	RACE_VAMPIRE = 8
)

// **********************************
// 1. appear()
// **********************************

func Appear(ch Combatant) {
	if HasAffect != nil && HasAffect(ch.GetName(), SPELL_INVISIBLE) {
		if RemoveAffect != nil {
			RemoveAffect(ch.GetName(), SPELL_INVISIBLE)
		}
	}
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

func GetPositionFromHP(hp int) int {
	if hp > 0 {
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
// 3. changeAlignment()
// **********************************

func ChangeAlignment(killer, victim Combatant) {
	if killer.IsNPC() {
		return
	}
	victimAlign := GetAlignment(victim.GetName())
	killerAlign := GetAlignment(killer.GetName())
	if victimAlign >= -350 && victimAlign <= 350 {
		return
	}
	newAlign := killerAlign + (-victimAlign-killerAlign)>>4
	if newAlign > 1000 {
		newAlign = 1000
	}
	if newAlign < -1000 {
		newAlign = -1000
	}
	if SetAlignment != nil {
		SetAlignment(killer.GetName(), newAlign)
	}
}

// **********************************
// 4. deathCry()
// **********************************

func DeathCry(ch Combatant) string {
	var rooms []string
	roomVNum := ch.GetRoom()
	msg := fmt.Sprintf("Your blood freezes as you hear %s's death cry.", ch.GetName())
	if BroadcastMessage != nil {
		BroadcastMessage(roomVNum, msg, "")
	}
	rooms = append(rooms, fmt.Sprintf("%d", roomVNum))
	for door := 0; door < NUM_OF_DIRS; door++ {
		if GetAdjacentRoom != nil {
			adjRoom := GetAdjacentRoom(roomVNum, door)
			if adjRoom >= 0 {
				if BroadcastMessage != nil {
					BroadcastMessage(adjRoom, "Your blood freezes as you hear someone's death cry.", "")
				}
				rooms = append(rooms, fmt.Sprintf("%d", adjRoom))
			}
		}
	}
	return strings.Join(rooms, ";")
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
		if !ch.IsNPC() || ch.GetLevel() >= LVL_IMMORT {
			if LogMessage != nil {
				LogMessage("Attempt to assign damage when ch and vict are in different rooms.",
					"NRM", LVL_IMMORT, false)
			}
		}
		return false
	}

	isOutlaw := HasPlrFlag != nil && HasPlrFlag(victimName, "PLR_OUTLAW")
	if !isOutlaw && victim.GetFighting() != chName && chName != victimName {
		if HasRoomFlag != nil && HasRoomFlag(roomVNum, "ROOM_PEACEFUL") {

			return false
		}
	}

	if victimName != chName && !ch.IsNPC() && !victim.IsNPC() {
		if ch.GetLevel() <= 10 {

			return false
		}
		if victim.GetLevel() <= 10 && !isOutlaw {

			return false
		}
	}

	if IsShopkeeper != nil && IsShopkeeper(victimName) {

		if ch.GetFighting() != "" {
			ch.StopFighting()
		}
		if victim.GetFighting() != "" {
			victim.StopFighting()
		}
		return false
	}

	// jail guard logic (fight.c:1370): guards respond to PK in cities
	if ch.IsNPC() && !victim.IsNPC() && HasMobVNum != nil &&
		(HasMobVNum(chName, 8102) || HasMobVNum(chName, 8103)) {
		if dam > 0 && ch.GetHP() > ch.GetMaxHP()/2 {
			hasVampire := HasAffectStr != nil && HasAffectStr(victimName, AFF_STR_VAMPIRE)
			hasWerewolf := HasAffectStr != nil && HasAffectStr(victimName, AFF_STR_WEREWOLF)
			if !hasVampire && !hasWerewolf {
				if BroadcastMessage != nil {
					BroadcastMessage(roomVNum,
						fmt.Sprintf("%s grabs %s by the collar, and quickly beats %s into submission.",
							chName, victimName, victimName), "")
				}
				victim.StopFighting()
				return false
			}
		}
	}

	if victimName != chName && ch.GetPosition() > PosStunned {
		if ch.GetFighting() == "" {
			ch.SetFighting(victimName)
		}

		// charm retarget (fight.c:1410): charmed NPC attacking their master's friend
		// In C: if victim is charmed and master in room, hit(ch, victim->master, TYPE_UNDEFINED)
		// Can't construct a Combatant for master here — game layer handles via hooks.
		// We leave a comment; the PerformCommand/Flee/etc hooks should cover it.

		// NPC target switching (fight.c:1420): high-level NPCs switch to highest-damage attacker
		if ch.IsNPC() && ch.GetLevel() > 20 {
			if HasMobFlag != nil && HasMobFlag(victimName, "HAS_AGGR_LIST") {
				// Aggro list tracking would be done via game-layer hooks.
				// The C code iterates room people to find FIGHTING(vict)==ch, remembers via memory.
				// This is inherently game-layer; we signal intent via PerformCommand if possible.
			}
		}

		if victim.GetPosition() > PosStunned && victim.GetFighting() == "" {
			victim.SetFighting(chName)
			// MOB_MEMORY: NPC remembers PC attacker (fight.c:1445)
			if HasMobFlag != nil && HasMobFlag(victimName, "MOB_MEMORY") && !ch.IsNPC() && ch.GetLevel() < LVL_IMMORT {
				if PerformCommand != nil {
					PerformCommand(victimName, fmt.Sprintf("remember %s", chName))
				}
			}
			// MOB_HUNTER: NPC starts hunting PC attacker (fight.c:1449)
			if HasMobFlag != nil && HasMobFlag(victimName, "MOB_HUNTER") && !ch.IsNPC() && ch.GetLevel() < LVL_IMMORT {
				if PerformCommand != nil {
					PerformCommand(victimName, fmt.Sprintf("hunt %s", chName))
				}
			}
		}
		// MOB_HUNTER: attacker also hunts victim (fight.c:1453)
		if HasMobFlag != nil && HasMobFlag(chName, "MOB_HUNTER") && !victim.IsNPC() && victim.GetLevel() < LVL_IMMORT {
			if PerformCommand != nil {
				PerformCommand(chName, fmt.Sprintf("hunt %s", victimName))
			}
		}
	}

	// stop_follower: if victim follows ch, break following (fight.c:1457)
	// Handled via game-layer hooks.

	// AFF_HIDE: attacker becomes visible on offensive action (fight.c:1459)
	if HasAffect != nil && HasAffect(chName, AFF_HIDE) {
		if RemoveAffect != nil {
			RemoveAffect(chName, AFF_HIDE)
		}
		if BroadcastMessage != nil {
			BroadcastMessage(roomVNum,
				fmt.Sprintf("%s slowly fades into existence.", chName), chName)
		}
	}

	if GetRaceHate != nil {
		victimRace := GetRace(victimName)
		for i := 0; i < 5; i++ {
			if GetRaceHate(chName, i) == victimRace {
				dam += ch.GetLevel() // no break — C applies for every matching slot
			}
		}
	}

	if HasAffect != nil && HasAffect(victimName, AFF_SANCTUARY) {
		dam /= 2
	}
	if HasAffect != nil && HasAffect(victimName, AFF_PROTECT_EVIL) && GetAlignment(chName) < -350 {
		dam -= victim.GetLevel() / 4
	}
	if HasAffect != nil && HasAffect(victimName, AFF_PROTECT_GOOD) && GetAlignment(chName) > 350 {
		dam -= victim.GetLevel() / 4
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

	if chName != victimName && !ch.IsNPC() && ch.GetLevel() < 2 {
		if GainExp != nil {
			GainExp(chName, victim.GetLevel()*dam)
		}
	}

	newPos := GetPositionFromHP(victim.GetHP())

	if newPos <= PosStunned {
		if ch.IsNPC() && !victim.IsNPC() && victim.GetLevel() <= 5 {
			ch.StopFighting()
		}
		if !victim.IsNPC() && HasRoomFlag != nil && HasRoomFlag(victim.GetRoom(), "ROOM_NEUTRAL") {
			if victim.GetFighting() != "" {
				victim.StopFighting()
			}
			victim.TakeDamage(-(victim.GetHP() - 1))
			if BroadcastMessage != nil {
				BroadcastMessage(victim.GetRoom(),
					fmt.Sprintf("%s is saved by the powers of the gods!", victimName), "")
			}
			return false
		}
	}

	isWeapon := attackType >= TYPE_HIT && attackType < TYPE_SUFFERING
	if !isWeapon {
		if SkillMessageFunc != nil {
			SkillMessageFunc(dam, chName, victimName, attackType)
		}
	} else {
		if newPos == PosDead || dam == 0 {
			sent := false
			if SkillMessageFunc != nil {
				sent = SkillMessageFunc(dam, chName, victimName, attackType)
			}
			if !sent {
				DamMessage(dam, ch, victim, attackType-TYPE_HIT)
			}
		} else {
			DamMessage(dam, ch, victim, attackType-TYPE_HIT)
		}
	}

	if !victim.IsNPC() && IsMounted != nil && IsMounted(victimName) && dam > 0 && rand.Intn(100) < 10 {
		if Dismount != nil {
			Dismount(victimName)
		}
	}

	switch newPos {
	case PosMortally:
			victim.SendMessage("You are mortally wounded, and will die soon, if not aided.\r\n")
			if BroadcastMessage != nil {
				BroadcastMessage(roomVNum,
					fmt.Sprintf("%s is mortally wounded, and will die soon, if not aided.", victimName), "")
			}
		case PosIncap:
			victim.SendMessage("You are incapacitated and will slowly die, if not aided.\r\n")
			if BroadcastMessage != nil {
				BroadcastMessage(roomVNum,
					fmt.Sprintf("%s is incapacitated and will slowly die, if not aided.", victimName), "")
			}
		case PosStunned:
			victim.SendMessage("You're stunned, but will probably regain consciousness again.\r\n")
			if BroadcastMessage != nil {
				BroadcastMessage(roomVNum,
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
				if HasMobFlag != nil && HasMobFlag(victimName, "MOB_WIMPY") && chName != victimName {
					if DoFlee != nil {
						DoFlee(victimName)
					}
				}
				if !victim.IsNPC() && GetWimpyLev != nil && GetWimpyLev(victimName) > 0 &&
					victimName != chName && newPos >= PosFighting &&
					victim.GetHP() < GetWimpyLev(victimName) {
					hasRetreat := GetSkill != nil && GetSkill(victimName, SKILL_RETREAT) > 0
					hasEscape := GetSkill != nil && GetSkill(victimName, SKILL_ESCAPE) > 0
					if hasRetreat || hasEscape {
						if DoRetreat != nil {
							DoRetreat(victimName)
						}
					} else if DoFlee != nil {
						DoFlee(victimName)
					}
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
				exp := GetExp(victimName)
				if exp > maxExpGain {
					exp = maxExpGain
				}
				exp = CalcLevelDiff(ch, victim, exp)
				
					if exp > 1 {
						ch.SendMessage(fmt.Sprintf("You receive %d experience points.\r\n", exp))
					} else {
						ch.SendMessage("You receive one lousy experience point.\r\n")
					}
				if !ch.IsNPC() && GainExp != nil {
					GainExp(chName, exp)
				}

				// autogold on kill (fight.c:1654)
				if HasPrfFlag != nil && HasPrfFlag(chName, "PRF_AUTOGOLD") {
					if PerformCommand != nil {
						PerformCommand(chName, "get all gold corpse")
					}
				}

				// autosplit — fight.c:756-830
				if HasPrfFlag != nil && HasPrfFlag(chName, "PRF_AUTOSPLIT") && GetGold != nil && SetGold != nil && ApplyToGroupMembers != nil {
					gold := GetGold(chName)
					if gold > 0 {
						numMembers := CountGroupMembers(chName, ch.GetRoom())
						if numMembers > 1 {
							perMember := gold / numMembers
							if perMember > 0 {
								ApplyToGroupMembers(chName, ch.GetRoom(), func(memberName string) {
									if memberName != chName {
										if SetGold != nil {
												SetGold(memberName, GetGold(memberName)+perMember)
											}
										}
									})
								ch.SendMessage(fmt.Sprintf("You split the gold and keep %d for yourself.\r\n", perMember))
								SetGold(chName, GetGold(chName)-gold+perMember+(gold%numMembers))
							} else {
								ch.SendMessage("You split no gold, you got none.\r\n")
							}
						}
					}
				}

				ChangeAlignment(ch, victim)
			}
		}

		// player death section (fight.c:1665)
		if !victim.IsNPC() {
			if !ch.IsNPC() && chName != victimName {
				// Pkill (fight.c:1672)
				if LogMessage != nil {
					LogMessage(fmt.Sprintf("(PK) %s killed by %s at room %d", victimName, chName, roomVNum),
						"BRF", LVL_IMMORT, true)
				}
				// flag killer as outlaw if victim wasn't one (fight.c:1675)
				if HasPlrFlag != nil && !HasPlrFlag(victimName, "PLR_OUTLAW") {
					if SetPlrFlag != nil {
						SetPlrFlag(chName)
					}
				}
			} else {
				if LogMessage != nil {
					LogMessage(fmt.Sprintf("%s killed by %s at room %d", victimName, chName, roomVNum),
						"BRF", LVL_IMMORT, true)
				}
			}
			if chName != victimName && GetPks != nil && SetPks != nil {
				SetPks(chName, GetPks(chName)+1)
			}
			if GetDeaths != nil && SetDeaths != nil {
				SetDeaths(victimName, GetDeaths(victimName)+1)
			}
			if SetLastDeath != nil {
				SetLastDeath(victimName, NowUnix())
			}
		}

		if GetKills != nil && SetKills != nil {
			SetKills(chName, GetKills(chName)+1)
		}

		CounterProcs(ch)
		DieWithKiller(victim, ch, attackType)

		if chName != victimName && HasMobFlag != nil &&
			(HasMobFlag(chName, "MOB_AGGR24") || HasMobFlag(chName, "MOB_LOOTS")) {
			AttitudeLoot(ch, victim)
		}

		// autoloot on kill (fight.c:1708)
		if !ch.IsNPC() && victim.IsNPC() && chName != victimName {
			if HasPrfFlag != nil && HasPrfFlag(chName, "PRF_AUTOLOOT") {
				if PerformCommand != nil {
					PerformCommand(chName, "get all corpse")
				}
			}
		}
	}

	if dam > 0 {
		return true
	}
	return false
}

var NowUnix = func() int64 { return 0 }

// ---------------------------------------------------------------------------
// Skill message functions for DamMessage
// ---------------------------------------------------------------------------

// damMessageTier describes one entry of the weapon damage message table.
type damMessageTier struct {
	MinDamage int
	Room      string
	Char      string
	Victim    string
}

var damMessageTiers = []damMessageTier{
	{0, "$n tries to #w $N, but misses.", "You try to #w $N, but miss.", "$n tries to #w you, but misses."},
	{1, "$n scratches $N as $e #W $M.", "You scratch $N as you #w $M.", "$n scratches you as $e #W you."},
	{3, "$n barely #W $N.", "You barely #w $N.", "$n barely #W you."},
	{5, "$n #W $N.", "You #w $N.", "$n #W you."},
	{7, "$n #W $N hard.", "You #w $N hard.", "$n #W you hard."},
	{11, "$n #W $N very hard.", "You #w $N very hard.", "$n #W you very hard."},
	{18, "$n #W $N extremely hard.", "You #w $N extremely hard.", "$n #W you extremely hard."},
	{26, "$n #W $N violently.", "You #w $N violently.", "$n #W you violently."},
	{36, "$n #W $N savagely.", "You #w $N savagely.", "$n #W you savagely."},
	{48, "$n MUTILATES $N!", "You MUTILATE $N!", "$n MUTILATES you!"},
	{60, "$n DISEMBOWELS $N!!", "You DISEMBOWEL $N!!", "$n DISEMBOWELS you!!"},
	{80, "$n DESTROYS $N!!!", "You DESTROY $N!!!", "$n DESTROYS you!!!"},
	{101, "$n OBLITERATES $N!!!!", "You OBLITERATE $N!!!!", "$n OBLITERATES you!!!!"},
	{10000, "$n R O C K S the Hell Out Of $N!!!!!!!!!!!!!!!!!!!!!!!!", "You R O C K the Hell Out Of $N!!!!!!!!!!!!!!!!!!!!!!!!", "$n R O C K S the Hell Out Of You!!!!!!!!!!!!!!!!!!!!!!!!"},
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

	roomMsg := replaceMessageTokens(tier.Room, ch.GetName(), victim.GetName(), singular, plural)
	_ = replaceMessageTokens(tier.Char, ch.GetName(), victim.GetName(), singular, plural)
	_ = replaceMessageTokens(tier.Victim, ch.GetName(), victim.GetName(), singular, plural)

	if BroadcastMessage != nil {
		BroadcastMessage(ch.GetRoom(), roomMsg, ch.GetName()+" "+victim.GetName())
	}


}

// replaceMessageTokens substitutes $n, $N, $e, #w, #W in a message template.
func replaceMessageTokens(msg, chName, victimName, singular, plural string) string {
	result := msg
	result = strings.ReplaceAll(result, "$n", chName)
	result = strings.ReplaceAll(result, "$N", victimName)
	result = strings.ReplaceAll(result, "$e", "he")
	result = strings.ReplaceAll(result, "$E", "him")
	result = strings.ReplaceAll(result, "$s", "his")
	result = strings.ReplaceAll(result, "$m", chName)
	result = strings.ReplaceAll(result, "$M", victimName)
	result = strings.ReplaceAll(result, "#w", singular)
	result = strings.ReplaceAll(result, "#W", plural)
	return result
}

// ---------------------------------------------------------------------------
// 6. makeHit()
// **********************************

func MakeHit(ch, victim Combatant) {
	chName := ch.GetName()
	victimName := victim.GetName()

	if ch.GetRoom() != victim.GetRoom() {
		if ch.GetFighting() == victimName {
			ch.StopFighting()
		}
		return
	}

	calcThaco := getTHAC0(ch)

	strIdx := strIndex(ch)
	if strIdx < len(strApp) {
		calcThaco -= strApp[strIdx].ToHit
	}

	wType := TYPE_HIT
	wieldDamNum, wieldDamSize := 0, 0
	isBlessed := false

	if GetWeaponInfo != nil {
		wType, wieldDamNum, wieldDamSize, isBlessed = GetWeaponInfo(chName)
	} else if !ch.IsNPC() {
		dr := ch.GetDamageRoll()
		wieldDamNum, wieldDamSize = dr.Num, dr.Sides
	} else if GetNPCData != nil {
		_, wieldDamNum, wieldDamSize = GetNPCData(chName)
	}

	if isBlessed {
		calcThaco--
	}

	calcThaco -= ch.GetHitroll()
	calcThaco -= int(float64(ch.GetInt()-13) / 1.5)
	calcThaco -= int(float64(ch.GetWis()-13) / 1.5)

	diceroll := rand.Intn(20) + 1

	victimAC := 0
	if GetMobAC != nil && victim.IsNPC() {
		victimAC = GetMobAC(victimName) / 10
	} else {
		victimAC = victim.GetAC() / 10
	}
	if victim.GetPosition() > PosSleeping {
		dexIdx := dexIndex(victim)
		if dexIdx >= 0 && dexIdx < len(dexApp) {
			victimAC += dexApp[dexIdx].Defensive
		}
	}
	if victimAC < -10 {
		victimAC = -10
	}

	isMiss := false
	if diceroll < 20 && victim.GetPosition() > PosSleeping {
		if diceroll == 1 || (calcThaco-diceroll) > victimAC {
			isMiss = true
		}
	}

	if !isMiss {
		dam := 0
		if strIdx < len(strApp) {
			dam = strApp[strIdx].ToDam
		}
		dam += ch.GetDamroll()
		if wieldDamNum > 0 && wieldDamSize > 0 {
			dam += RollDice(wieldDamNum, wieldDamSize)
		} else {
			dam += rand.Intn(ch.GetLevel()/3 + 1)
		}

		defPos := victim.GetPosition()
		if defPos < PosFighting {
			dam = int(float64(dam) * (1.0 + float64(PosFighting-defPos)/3.0))
		}
		if dam < 1 {
			dam = 1
		}

		dam = getMinusDam(dam, victim.GetAC())

		if ch.GetStr() == 0 {
			dam = 1
		}
		TakeDamage(ch, victim, dam, wType)
	} else {
		TakeDamage(ch, victim, 0, wType)
	}

	if ch.IsNPC() && ch.GetPosition() > PosStunned && HasScriptFlag != nil &&
		HasScriptFlag(chName, "MS_FIGHTING") && RunFightScript != nil {
		RunFightScript(chName, victimName, ch.GetRoom())
	}
}

// **********************************
// 7. groupGain / calcLevelDiff / performGroupGain
// **********************************

func IsInGroup(ch Combatant) bool {
	chName := ch.GetName()
	chRoom := ch.GetRoom()
	if HasAffectStr != nil && HasAffectStr(chName, AFF_STR_GROUP) {
		if ch.GetName() == "" {
			if GetFollowersInRoom != nil {
				return GetFollowersInRoom(chName, chRoom) > 0
			}
		} else {
			if GetMasterInRoom != nil && GetMasterInRoom(chName, chRoom) {
				return true
			}
			if GetFellowFollowersInRoom != nil && GetFellowFollowersInRoom(chName, chRoom) {
				return true
			}
		}
	}
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
	if !ch.IsNPC() && GainExp != nil {
		GainExp(ch.GetName(), share)
	}
	ChangeAlignment(ch, victim)
}

func GroupGain(ch, victim Combatant) {
	leaderName := ch.GetName()
	if leaderName == "" {
		leaderName = ch.GetName()
	}
	numMembers := 1
	if CountGroupMembers != nil {
		numMembers = CountGroupMembers(leaderName, ch.GetRoom())
	}
	if numMembers < 1 {
		numMembers = 1
	}

	victimExp := GetExp(victim.GetName())
	base := victimExp / numMembers
	if base > 100 {
		base -= int(float64(base) * 0.01)
	}
	if base < 1 {
		base = 1
	}

	if ApplyToGroupMembers != nil {
		ApplyToGroupMembers(leaderName, ch.GetRoom(), func(memberName string) {
			m := NewNamedCombatant(memberName, ch.GetRoom())
			PerformGroupGain(m, victim, base)
		})
	}
}

// **********************************
// 8. rawKill()
// **********************************

func RawKill(ch Combatant, attackType int) {
	chName := ch.GetName()
	if ch.GetRoom() < 0 {
		return
	}
	if ch.GetFighting() != "" {
		ch.StopFighting()
	}
	if RemoveAllAffects != nil {
		RemoveAllAffects(chName)
	}
	if Unmount != nil {
		Unmount(chName)
	}
	DeathCry(ch)

	victimRace := GetRace(chName)
	if victimRace == RACE_UNDEAD || victimRace == RACE_VAMPIRE {
		if MakeDustFunc != nil {
			MakeDustFunc(chName, attackType)
		}
	} else {
		if MakeCorpseFunc != nil {
			MakeCorpseFunc(chName, attackType)
		}
	}
	if ExtractChar != nil {
		ExtractChar(chName)
	}
}

// **********************************
// 9. dieWithKiller()
// **********************************

func DieWithKiller(ch, killer Combatant, attackType int) {
	chName := ch.GetName()

	if GainExp != nil {
		GainExp(chName, -(GetExp(chName) / 37))
	}

	if !ch.IsNPC() && GetConstitution != nil && SetConstitution != nil {
		conVal := GetConstitution(chName) - 1
		if conVal < 0 {
			conVal = 0
		}
		SetConstitution(chName, conVal)
	}

	roomVNum := ch.GetRoom()
	if HasScriptFlag != nil && HasScriptFlag(chName, "MS_DEATH") && RunDeathScript != nil {
		RunDeathScript(killer.GetName(), chName, roomVNum)
	}

	RawKill(ch, attackType)
}

// **********************************
// 10. die()
// **********************************

func Die(ch Combatant) {
	if GainExp != nil {
		GainExp(ch.GetName(), -(GetExp(ch.GetName()) / 3))
	}
	RawKill(ch, TYPE_UNDEFINED)
}

// **********************************
// 11. makeCorpse()
// **********************************

func MakeCorpse(victim Combatant, attackType int) {
	name := victim.GetName()
	if MakeCorpseFunc != nil {
		MakeCorpseFunc(name, attackType)
	}
}

// **********************************
// 12. makeDust()
// **********************************

func MakeDust(victim Combatant, attackType int) {
	name := victim.GetName()
	if MakeDustFunc != nil {
		MakeDustFunc(name, attackType)
	}
}

// **********************************
// 13. counterProcs()
// **********************************

func CounterProcs(ch Combatant) {
	if ch.IsNPC() {
		return
	}
	kills := int64(0)
	if GetKills != nil {
		kills = GetKills(ch.GetName())
	}

	reward := false
	switch kills {
	case 5000, 15000, 25000, 35000, 45000:
		// Minor milestones: full heal + global blessing
		ch.SendMessage("The gods reward your glory in battle!\r\n")
		ch.Heal(ch.GetMaxHP() - ch.GetHP())
		reward = true
	case 1000, 2000, 10000, 20000, 30000, 40000, 50000:
		// Major milestones: random +1 max stat, full heal, global blessing
		ch.SendMessage("The gods reward your many victories!\r\n")
		reward = true
		// C has a bug: missing break in switch cases means all 3 branches execute.
		// Reproducing the bug for fidelity.
		// In C: case 1: GET_MAX_HIT++; case 2: GET_MAX_MANA++; case 3: GET_MAX_MOVE++;
		//            default: GET_MAX_HIT++; break;
		// Since case 3 falls through to default and all lack breaks,
		// ALL THREE stats get +1 (case 1+3 hit, case 2 mana, case 3 move).
		if IncreaseMaxStat != nil {
			IncreaseMaxStat(ch.GetName(), "hp")
			IncreaseMaxStat(ch.GetName(), "mana")
			IncreaseMaxStat(ch.GetName(), "move")
		}
		ch.Heal(ch.GetMaxHP() - ch.GetHP())
	default:
		return
	}

	if reward {
		// Global blessing — heal all connected players
		if HealAllPlayers != nil {
			HealAllPlayers()
		}
		// Log milestone
		if LogMessage != nil {
			LogMessage(fmt.Sprintf("%s hit %d kills.", ch.GetName(), kills), "NRM", LVL_IMMORT, false)
		}
	}
}

// **********************************
// 14. attitudeLoot()
// **********************************

func AttitudeLoot(ch, victim Combatant) {
	chName := ch.GetName()
	if PerformCommand != nil {
		PerformCommand(chName, fmt.Sprintf("get all corpse of %s", victim.GetName()))
	}
	if BroadChatFunc != nil {
		BroadChatFunc(chName, "Grins wickedly as he picks your corpse.")
	}
}

// **********************************
// 15. damMessage()
// **********************************


// **********************************
// 16. skillMessage()
// **********************************

func SkillMessage(dam int, ch, victim Combatant, attackType int) {
	if attackType < TYPE_HIT {
		if SkillMessageFunc != nil {
			SkillMessageFunc(dam, ch.GetName(), victim.GetName(), attackType)
		}
	}
}

// **********************************
// 17. replaceString()
// **********************************

func replaceString(msg, chName, victimName string) string {
	result := strings.ReplaceAll(msg, "$n", chName)
	result = strings.ReplaceAll(result, "$N", victimName)
	return result
}

// **********************************
// 21. stopFighting / setFighting helpers
// **********************************

func NewNamedCombatant(name string, roomVNum int) Combatant {
	return &namedCombatant{name: name, room: roomVNum}
}

type namedCombatant struct {
	name string
	room int
}

func (n *namedCombatant) GetName() string            { return n.name }
func (n *namedCombatant) IsNPC() bool                 { return true }
func (n *namedCombatant) GetRoom() int                { return n.room }
func (n *namedCombatant) GetLevel() int               { return 0 }
func (n *namedCombatant) GetHP() int                  { return 0 }
func (n *namedCombatant) GetMaxHP() int               { return 0 }
func (n *namedCombatant) GetAC() int                  { return 0 }
func (n *namedCombatant) GetTHAC0() int               { return 0 }
func (n *namedCombatant) GetDamageRoll() DiceRoll     { return DiceRoll{} }
func (n *namedCombatant) GetPosition() int            { return PosStanding }
func (n *namedCombatant) GetClass() int               { return 0 }
func (n *namedCombatant) GetStr() int                 { return 0 }
func (n *namedCombatant) GetStrAdd() int              { return 0 }
func (n *namedCombatant) GetDex() int                 { return 0 }
func (n *namedCombatant) GetInt() int                 { return 0 }
func (n *namedCombatant) GetWis() int                 { return 0 }
func (n *namedCombatant) GetHitroll() int             { return 0 }
func (n *namedCombatant) GetDamroll() int             { return 0 }
func (n *namedCombatant) GetSex() int                 { return 1 }
func (n *namedCombatant) GetMaster() string           { return "" }
func (n *namedCombatant) TakeDamage(amount int)       {}
func (n *namedCombatant) Heal(amount int)             {}
func (n *namedCombatant) SetFighting(target string)   {}
func (n *namedCombatant) StopFighting()               {}
func (n *namedCombatant) GetFighting() string         { return "" }
func (n *namedCombatant) SendMessage(msg string)      {}
func (n *namedCombatant) GetSendMessage(msg string)   {}
