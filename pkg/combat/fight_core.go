// Package combat — fight_core.go
// Port of src/fight.c from the Dark Pawns C codebase.
package combat

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// ---------------------------------------------------------------------------
// Game-layer hooks
//
// All legacy package-level function hooks have been migrated to the
// GameCallbacks struct owned by CombatEngine. fight_core functions access them
// through the cb* helpers in callbacks.go, which read from the active
// callbacks instance set during engine initialization.
// ---------------------------------------------------------------------------

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
	SKILL_RETREAT      = 149
	SKILL_ESCAPE       = 157
	SKILL_PARRY        = 172
)

// AFF_* are C affect *bit positions* from src/structs.h. They are passed
// straight through cbHasAffect → (*Player|*MobInstance).IsAffected, which tests
// bit (1<<pos) and maps via AffBitToEngineFlag — so these MUST match the real
// structs.h positions, not an arbitrary 1..N sequence. (Previously fabricated as
// 1..10, which made cbHasAffect check the wrong bits: AFF_SANCTUARY=5 hit
// SENSE_LIFE, AFF_HASTE=9 hit CURSE, etc. — DP-1025.)
const (
	AFF_INVISIBLE    = 1  // structs.h AFF_INVISIBLE
	AFF_SANCTUARY    = 7  // structs.h AFF_SANCTUARY
	AFF_GROUP        = 8  // structs.h AFF_GROUP
	AFF_PROTECT_EVIL = 12 // structs.h AFF_PROTECT_EVIL
	AFF_PROTECT_GOOD = 13 // structs.h AFF_PROTECT_GOOD
	AFF_SLEEP        = 14 // structs.h AFF_SLEEP
	AFF_DODGE        = 17 // structs.h AFF_DODGE
	AFF_HIDE         = 19 // structs.h AFF_HIDE
	AFF_CHARM        = 21 // structs.h AFF_CHARM
	AFF_HASTE        = 33 // structs.h AFF_HASTE
	AFF_SLOW         = 34 // structs.h AFF_SLOW
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

// **********************************
// 1. updatePos()
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

// UpdatePositionAfterDamage transitions a victim into the wounded band after
// damage has already been applied to its HP. It derives the new position from
// current HP (POS_STUNNED / POS_INCAP / POS_MORTALLYW / POS_DEAD), sets it,
// emits the matching wounded-state message to the victim and its room, and
// drops the victim's FIGHTING reference once it can no longer fight
// (pos < POS_SLEEPING). It returns the new position; callers must invoke the
// death pipeline when the return value is PosDead (HP <= -11). Death messaging
// is intentionally left to the death handler, so PosDead emits no message here.
//
// Mirrors the update_pos + wound-message block of fight.c:1484-1512. broadcast
// may be nil to suppress the third-person room message.
func UpdatePositionAfterDamage(victim Combatant, broadcast func(roomVNum int, message, exclude string)) int {
	newPos := GetPositionFromHP(victim.GetHP(), victim.GetPosition())
	victim.SetPosition(newPos)

	name := victim.GetName()
	room := victim.GetRoom()
	switch newPos {
	case PosMortally:
		victim.SendMessage("You are mortally wounded, and will die soon, if not aided.\r\n")
		if broadcast != nil {
			broadcast(room, fmt.Sprintf("%s is mortally wounded, and will die soon, if not aided.", name), name)
		}
	case PosIncap:
		victim.SendMessage("You are incapacitated and will slowly die, if not aided.\r\n")
		if broadcast != nil {
			broadcast(room, fmt.Sprintf("%s is incapacitated and will slowly die, if not aided.", name), name)
		}
	case PosStunned:
		victim.SendMessage("You're stunned, but will probably regain consciousness again.\r\n")
		if broadcast != nil {
			broadcast(room, fmt.Sprintf("%s is stunned, but will probably regain consciousness again.", name), name)
		}
	}

	// fight.c:1500 — a downed victim can no longer fight back. The attacker
	// keeps its FIGHTING reference and finishes the victim off next round.
	if newPos < PosSleeping && victim.GetFighting() != "" {
		victim.StopFighting()
	}
	return newPos
}

// ApplyDamageModifiers applies the fight.c damage() modifier block
// (src/fight.c:1466-1483) to a raw damage figure and returns the adjusted value:
//
//	race-hate weapons  dam += GET_LEVEL(ch)   per matching race-hate slot
//	sanctuary          dam /= 2
//	protect evil/good  dam -= GET_LEVEL(victim)/4 when attacker is evil/good
//	immortal victim    dam  = 0
//	clamp              dam  = MAX(MIN(dam, 3000), 0)
//
// It is the single funnel every damage path must pass computed damage through —
// melee (engine.processCombatPair), skills (game.doDamage), spells
// (game.DoSpellDamage), and the full damage() port (TakeDamage) — so sanctuary,
// protection auras, race-hate, the 3000 cap, and immortal invulnerability apply
// uniformly instead of only on the spell path (DP-1025). The C order is
// preserved exactly; do not reorder — sanctuary halving before vs after the
// protection subtractions yields different results.
//
// ch may be nil for source-less damage (environment/DoT with no attacker); the
// attacker-dependent modifiers (race-hate, protection) are skipped in that case
// while victim-only modifiers (sanctuary, immortal, cap) still apply.
func ApplyDamageModifiers(ch, victim Combatant, dam int) int {
	if victim == nil {
		return dam
	}
	victimName := victim.GetName()

	if ch != nil {
		chName := ch.GetName()
		// race-hate weapons: +attacker level per matching slot, no break —
		// C applies the bonus once for every matching race_hate entry.
		if callbacks != nil && callbacks.GetRaceHate != nil && callbacks.GetRace != nil {
			victimRace := cbGetRace(victimName)
			for i := 0; i < 5; i++ {
				if cbGetRaceHate(chName, i) == victimRace {
					dam += ch.GetLevel()
				}
			}
		}
		if cbHasAffect(victimName, AFF_SANCTUARY) {
			dam /= 2
		}
		if cbHasAffect(victimName, AFF_PROTECT_EVIL) && cbGetAlignment(chName) <= -350 {
			dam -= victim.GetLevel() / 4
		}
		if cbHasAffect(victimName, AFF_PROTECT_GOOD) && cbGetAlignment(chName) >= 350 {
			dam -= victim.GetLevel() / 4
		}
	} else if cbHasAffect(victimName, AFF_SANCTUARY) {
		// Source-less damage still honors sanctuary; race-hate and the
		// alignment-gated protection auras have no attacker to test against.
		dam /= 2
	}

	// You can't damage an immortal (fight.c:1480).
	if !victim.IsNPC() && victim.GetLevel() >= LVL_IMMORT {
		dam = 0
	}

	return max(min(dam, 3000), 0)
}

// **********************************
// 3. changeAlignment()
// **********************************

func ChangeAlignment(killer, victim Combatant) {
	if killer.IsNPC() {
		return
	}
	if callbacks == nil || callbacks.GetAlignment == nil {
		return
	}
	victimAlign := cbGetAlignment(victim.GetName())
	killerAlign := cbGetAlignment(killer.GetName())
	if victimAlign > -350 && victimAlign < 350 {
		return
	}
	newAlign := killerAlign + (-victimAlign-killerAlign)>>4
	if newAlign > 1000 {
		newAlign = 1000
	}
	if newAlign < -1000 {
		newAlign = -1000
	}
	cbSetAlignment(killer.GetName(), newAlign)
}

// **********************************
// 4. deathCry()
// **********************************

func DeathCry(ch Combatant) string {
	var rooms []string
	roomVNum := ch.GetRoom()
	msg := fmt.Sprintf("Your blood freezes as you hear %s's death cry.", ch.GetName())
	// death_cry() uses TO_ROOM: the dead character does not hear the room
	// broadcast (fight.c:558-577).
	cbBroadcast(roomVNum, msg, ch.GetName())
	rooms = append(rooms, fmt.Sprintf("%d", roomVNum))
	for door := 0; door < NUM_OF_DIRS; door++ {
		adjRoom := cbGetAdjacentRoom(roomVNum, door)
		if adjRoom >= 0 {
			cbBroadcast(adjRoom, "Your blood freezes as you hear someone's death cry.", "")
			rooms = append(rooms, fmt.Sprintf("%d", adjRoom))
		}
	}
	return strings.Join(rooms, ";")
}

// EmitDeathPositionMessage emits the POS_DEAD branch of damage(). It is kept
// separate from UpdatePositionAfterDamage because the latter is also used by
// callers that only need the wounded-band transition; the actual C death
// pipeline owns this message immediately before death_cry/raw_kill.
func EmitDeathPositionMessage(victim Combatant, broadcast func(roomVNum int, message, exclude string)) {
	victim.SendMessage("You are dead!  Sorry...\r\n")
	if broadcast != nil {
		broadcast(victim.GetRoom(),
			capitalizeFightMessage(fmt.Sprintf("%s is dead!  R.I.P.", victim.GetName())),
			victim.GetName())
	}
}

// **********************************
// 5. takeDamage()
// **********************************

// TakeDamage applies the complete C damage() path and uses the package's
// normal death callback. Callers that own a game-layer death pipeline should
// use TakeDamageWithDeath so the damage/message ordering stays shared.
func TakeDamage(ch, victim Combatant, dam int, attackType int) bool {
	return takeDamage(ch, victim, dam, attackType, nil)
}

// TakeDamageWithDeath applies the complete C damage() path, replacing the
// default DieWithKiller/RawKill tail with onDeath. This is needed by delayed
// game-layer actions whose death bookkeeping is owned by World.HandleDeath;
// the callback runs only after C's skill/wound/death messages and state
// transitions have been emitted.
func TakeDamageWithDeath(ch, victim Combatant, dam int, attackType int, onDeath func()) bool {
	return takeDamage(ch, victim, dam, attackType, onDeath)
}

func takeDamage(ch, victim Combatant, dam int, attackType int, onDeath func()) bool {
	if victim.GetPosition() <= PosDead {
		return false
	}
	chName := ch.GetName()
	victimName := victim.GetName()
	roomVNum := ch.GetRoom()

	if ch.GetRoom() != victim.GetRoom() {
		if !ch.IsNPC() || ch.GetLevel() >= LVL_IMMORT {
			cbLog("Attempt to assign damage when ch and vict are in different rooms.",
				"NRM", LVL_IMMORT, false)
		}
		return false
	}

	isOutlaw := cbHasPlrFlag(victimName, "PLR_OUTLAW")
	if !isOutlaw && victim.GetFighting() != chName && chName != victimName {
		if cbHasRoomFlag(roomVNum, "ROOM_PEACEFUL") {
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

	if cbIsShopkeeper(victimName) {
		if ch.GetFighting() != "" {
			ch.StopFighting()
		}
		if victim.GetFighting() != "" {
			victim.StopFighting()
		}
		return false
	}

	// jail guard logic (fight.c:1370): guards respond to PK in cities
	if ch.IsNPC() && !victim.IsNPC() &&
		cbMobHasJailGuardSpec(chName) {
		if dam > 0 && ch.GetHP() > ch.GetMaxHP()/2 {
			hasVampire := cbHasAffectStr(victimName, AFF_STR_VAMPIRE)
			hasWerewolf := cbHasAffectStr(victimName, AFF_STR_WEREWOLF)
			if !hasVampire && !hasWerewolf {
				cbBroadcast(ch.GetRoom(),
					fmt.Sprintf("%s grabs %s by the collar, and quickly beats %s into submission.",
						chName, victimName, victimName), "")
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
		// Aggro list tracking is done via game-layer hooks.
		// The C code iterates room people to find FIGHTING(vict)==ch, remembers via memory.
		// This is inherently game-layer; we signal intent via PerformCommand if possible.

		if victim.GetPosition() > PosStunned && victim.GetFighting() == "" {
			victim.SetFighting(chName)
			// MOB_MEMORY: NPC remembers PC attacker (fight.c:1445)
			if cbHasMobFlag(victimName, "MOB_MEMORY") && !ch.IsNPC() && ch.GetLevel() < LVL_IMMORT {
				cbPerformCommand(victimName, fmt.Sprintf("remember %s", chName))
			}
			// MOB_HUNTER: NPC starts hunting PC attacker (fight.c:1449)
			if cbHasMobFlag(victimName, "MOB_HUNTER") && !ch.IsNPC() && ch.GetLevel() < LVL_IMMORT {
				cbPerformCommand(victimName, fmt.Sprintf("hunt %s", chName))
			}
		}
		// MOB_HUNTER: attacker also hunts victim (fight.c:1453)
		if cbHasMobFlag(chName, "MOB_HUNTER") && !victim.IsNPC() && victim.GetLevel() < LVL_IMMORT {
			cbPerformCommand(chName, fmt.Sprintf("hunt %s", victimName))
		}
	}

	// stop_follower: if victim follows ch, break following (fight.c:1457) —
	// the charm branch makes the attacking master's pet denounce him.
	if cbGetFollowing(victimName) == chName {
		cbStopFollowerOfMaster(victimName, chName)
	}

	// AFF_HIDE: attacker becomes visible on offensive action (fight.c:1459)
	if cbHasAffect(chName, AFF_HIDE) {
		cbRemoveAffect(chName, AFF_HIDE)
		cbBroadcast(ch.GetRoom(),
			fmt.Sprintf("%s slowly fades into existence.", chName), chName)
	}

	dam = ApplyDamageModifiers(ch, victim, dam)

	victim.TakeDamage(dam)

	if chName != victimName && !ch.IsNPC() && ch.GetLevel() < 2 {
		cbGainExp(chName, victim.GetLevel()*dam)
	}

	newPos := GetPositionFromHP(victim.GetHP(), victim.GetPosition())
	victim.SetPosition(newPos)

	if newPos <= PosStunned {
		if ch.IsNPC() && !victim.IsNPC() && victim.GetLevel() <= 5 {
			ch.StopFighting()
		}
		if !victim.IsNPC() && cbHasRoomFlag(victim.GetRoom(), "ROOM_NEUTRAL") {
			if victim.GetFighting() != "" {
				victim.StopFighting()
			}
			victim.TakeDamage(-(victim.GetHP() - 1))
			cbBroadcast(victim.GetRoom(),
				fmt.Sprintf("%s is saved by the powers of the gods!", victimName), "")
			return false
		}
	}

	isWeapon := attackType >= TYPE_HIT && attackType < TYPE_SUFFERING
	if !isWeapon {
		cbSkillMessage(dam, chName, victimName, attackType, ch.GetRoom())
	} else {
		if newPos == PosDead || dam == 0 {
			sent := cbSkillMessage(dam, chName, victimName, attackType, ch.GetRoom())
			if !sent {
				DamMessage(dam, ch, victim, attackType-TYPE_HIT)
			}
		} else {
			DamMessage(dam, ch, victim, attackType-TYPE_HIT)
		}
	}

	if !victim.IsNPC() && cbIsMounted(victimName) && dam > 0 && GetRoller().Number(0, 99) < 10 {
		cbDismount(victimName)
	}

	switch newPos {
	case PosMortally:
		victim.SendMessage("You are mortally wounded, and will die soon, if not aided.\r\n")
		cbBroadcast(ch.GetRoom(),
			fmt.Sprintf("%s is mortally wounded, and will die soon, if not aided.", victimName), "")
	case PosIncap:
		victim.SendMessage("You are incapacitated and will slowly die, if not aided.\r\n")
		cbBroadcast(ch.GetRoom(),
			fmt.Sprintf("%s is incapacitated and will slowly die, if not aided.", victimName), "")
	case PosStunned:
		victim.SendMessage("You're stunned, but will probably regain consciousness again.\r\n")
		cbBroadcast(ch.GetRoom(),
			fmt.Sprintf("%s is stunned, but will probably regain consciousness again.", victimName), "")
	case PosDead:
		victim.SendMessage("You are dead!  Sorry...\r\n")
		cbBroadcast(roomVNum, capitalizeFightMessage(fmt.Sprintf("%s is dead!  R.I.P.", victimName)), "")
	default:
		if dam > victim.GetMaxHP()/4 {
			victim.SendMessage("That really did HURT!\r\n")
			// C burns number(0,2) for the optional room scream even when the
			// victim is an NPC; TO_NOTVICT excludes both combatants
			// (fight.c:1580-1585). Keep the draw before the caller's
			// post-damage improve_skill (R3/R5e).
			if GetRoller().Number(0, 2) == 0 {
				cbBroadcast(roomVNum,
					capitalizeFightMessage(fmt.Sprintf("%s screams in pain!", victimName)),
					chName+" "+victimName)
			}
		}
		if victim.GetHP() < victim.GetMaxHP()/4 {
			victim.SendMessage("You wish that your wounds would stop BLEEDING so much!\r\n")
			if cbHasMobFlag(victimName, "MOB_WIMPY") && chName != victimName {
				cbDoFlee(victimName)
			}
		}
		if !victim.IsNPC() && cbGetWimpyLev(victimName) > 0 &&
			victimName != chName && newPos >= PosFighting &&
			victim.GetHP() < cbGetWimpyLev(victimName) {
			victim.SendMessage("You wimp out, and attempt to flee!\r\n")
			hasRetreat := cbGetSkill(victimName, SKILL_RETREAT) > 0
			hasEscape := cbGetSkill(victimName, SKILL_ESCAPE) > 0
			if hasRetreat || hasEscape {
				cbDoRetreat(victimName)
			} else {
				cbDoFlee(victimName)
			}
		}
	}

	if newPos < PosSleeping && victim.GetFighting() != "" {
		victim.StopFighting()
	}

	if newPos == PosDead {
		if onDeath == nil && victim.IsNPC() {
			if IsInGroup(ch) {
				GroupGain(ch, victim)
			} else {
				exp := cbGetExp(victimName)
				if exp > maxExpGain {
					exp = maxExpGain
				}
				exp = CalcLevelDiff(ch, victim, exp)

				if exp > 1 {
					ch.SendMessage(fmt.Sprintf("You receive %d experience points.\r\n", exp))
				} else {
					ch.SendMessage("You receive one lousy experience point.\r\n")
				}
				if !ch.IsNPC() {
					cbGainExp(chName, exp)
				}

				// autogold on kill (fight.c:1654)
				if cbHasPrfFlag(chName, "PRF_AUTOGOLD") {
					cbPerformCommand(chName, "get all gold corpse")
				}

				// autosplit — fight.c:756-830
				if cbHasPrfFlag(chName, "PRF_AUTOSPLIT") {
					gold := cbGetGold(chName)
					if gold > 0 {
						numMembers := cbCountGroupMembers(chName, ch.GetRoom())
						if numMembers > 1 {
							perMember := gold / numMembers
							if perMember > 0 {
								cbApplyToGroupMembers(chName, ch.GetRoom(), func(memberName string) {
									if memberName != chName {
										cbSetGold(memberName, cbGetGold(memberName)+perMember)
									}
								})
								ch.SendMessage(fmt.Sprintf("You split the gold and keep %d for yourself.\r\n", perMember))
								cbSetGold(chName, cbGetGold(chName)-gold+perMember+(gold%numMembers))
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
		if onDeath == nil && !victim.IsNPC() {
			if !ch.IsNPC() && chName != victimName {
				// Pkill (fight.c:1672)
				cbLog(fmt.Sprintf("(PK) %s killed by %s at room %d", victimName, chName, roomVNum),
					"BRF", LVL_IMMORT, true)
				// flag killer as outlaw if victim wasn't one (fight.c:1675)
				if !cbHasPlrFlag(victimName, "PLR_OUTLAW") {
					cbSetPlrFlag(chName)
				}
			} else {
				cbLog(fmt.Sprintf("%s killed by %s at room %d", victimName, chName, roomVNum),
					"BRF", LVL_IMMORT, true)
			}
			if chName != victimName {
				cbSetPks(chName, cbGetPks(chName)+1)
			}
			cbSetDeaths(victimName, cbGetDeaths(victimName)+1)
			cbSetLastDeath(victimName, time.Now().Unix())
		}

		if onDeath != nil {
			onDeath()
		} else {
			cbSetKills(chName, cbGetKills(chName)+1)

			CounterProcs(ch)
			DieWithKiller(victim, ch, attackType)
		}

		if onDeath == nil && chName != victimName &&
			(cbHasMobFlag(chName, "MOB_AGGR24") || cbHasMobFlag(chName, "MOB_LOOTS")) {
			AttitudeLoot(ch, victim)
		}

		// autoloot on kill (fight.c:1708)
		if onDeath == nil && !ch.IsNPC() && victim.IsNPC() && chName != victimName {
			if cbHasPrfFlag(chName, "PRF_AUTOLOOT") {
				cbPerformCommand(chName, "get all corpse")
			}
		}
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
// C has exactly one fixed message for each audience at each severity.
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
	return s[GetRoller().Number(0, len(s)-1)]
}

// damMessageTiers — weapon damage message tiers.
//
// Boundaries and texts follow src/fight.c:895-992 exactly (DP-1043). The C
// if/else chain is:
//
//	dam==0 → 0, <=2 → 1, <=4 → 2, <=6 → 3, <=10 → 4, <=14 → 5,
//	<=19 → 6, <=23 → 7, <=33 → 8, <=43 → 9, <=53 → 10, else → 11
//
// MinDamage below is the lowest damage value that selects each tier. C's
// dam_weapons[] has exactly one message per tier, so each tier below keeps
// that single verbatim string.
var damMessageTiers = []damMessageTier{
	// Tier 0: miss (dam == 0)
	{
		0,
		[]string{"$n tries to #w $N, but misses."},
		[]string{"You try to #w $N, but miss."},
		[]string{"$n tries to #w you, but misses."},
	},
	// Tier 1: scratch (1-2)
	{
		1,
		[]string{"$n scratches $N as $e #W $M."},
		[]string{"You scratch $N as you #w $M."},
		[]string{"$n scratches you as $e #W you."},
	},
	// Tier 2: barely (3-4)
	{
		3,
		[]string{"$n barely #W $N."},
		[]string{"You barely #w $N."},
		[]string{"$n barely #W you."},
	},
	// Tier 3: light (5-6)
	{
		5,
		[]string{"$n #W $N."},
		[]string{"You #w $N."},
		[]string{"$n #W you."},
	},
	// Tier 4: hard (7-10)
	{
		7,
		[]string{"$n #W $N hard."},
		[]string{"You #w $N hard."},
		[]string{"$n #W you hard."},
	},
	// Tier 5: very hard (11-14)
	{
		11,
		[]string{"$n #W $N very hard."},
		[]string{"You #w $N very hard."},
		[]string{"$n #W you very hard."},
	},
	// Tier 6: extremely hard (15-19)
	{
		15,
		[]string{"$n #W $N extremely hard."},
		[]string{"You #w $N extremely hard."},
		[]string{"$n #W you extremely hard."},
	},
	// Tier 7: massacres (20-23)
	{
		20,
		[]string{"$n massacres $N to small fragments with $s #w."},
		[]string{"You massacre $N to small fragments with your #w."},
		[]string{"$n massacres you to small fragments with $s #w."},
	},
	// Tier 8: OBLITERATES (24-33)
	{
		24,
		[]string{"$n OBLITERATES $N with $s deadly #w!!"},
		[]string{"You OBLITERATE $N with your deadly #w!!"},
		[]string{"$n OBLITERATES you with $s deadly #w!!"},
	},
	// Tier 9: EVISCERATES (34-43)
	{
		34,
		[]string{"$n EVISCERATES $N with $s incredible #w!!"},
		[]string{"You EVISCERATE $N with your incredible #w!!"},
		[]string{"$n EVISCERATES you with $s incredible #w!!"},
	},
	// Tier 10: DESTROYS (44-53)
	{
		44,
		[]string{"$n DESTROYS $N with $s ungodly #w!!"},
		[]string{"You DESTROY $N with your ungodly #w!!"},
		[]string{"$n DESTROYS you with $s ungodly #w!!"},
	},
	// Tier 11: ROCKS THE HELL OUT OF (54+)
	{
		54,
		[]string{"$n ROCKS THE HELL OUT OF $N with $s ultimate #w!!"},
		[]string{"You ROCK THE HELL OUT OF $N with your ultimate #w!!"},
		[]string{"$n ROCKS THE HELL OUT OF you with $s ultimate #w!!"},
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
	victimSex := victim.GetSex()
	roomMsg := replaceMessageTokens(tier.Room[0], ch.GetName(), victim.GetName(), singular, plural, sex, victimSex)
	charMsg := replaceMessageTokens(tier.Char[0], ch.GetName(), victim.GetName(), singular, plural, sex, victimSex)
	victimMsg := replaceMessageTokens(tier.Victim[0], ch.GetName(), victim.GetName(), singular, plural, sex, victimSex)

	cbBroadcast(ch.GetRoom(), roomMsg, ch.GetName()+" "+victim.GetName())

	// CRIT-010: Send attacker and victim their own messages.
	// In C, act() sent to TO_CHAR and TO_VICT separately.
	cbSendToChar(ch.GetName(), charMsg)
	cbSendToChar(victim.GetName(), victimMsg)
}

// replaceMessageTokens applies the character and weapon substitutions used by
// act() fight messages.
// sex uses the Go Actor encoding: 0=male, 1=female, 2=neutral/other.
func replaceMessageTokens(msg, chName, victimName, singular, plural string, sex, victimSex int) string {
	chPronouns := sexPronouns(sex)
	victimPronouns := sexPronouns(victimSex)
	result := msg
	result = strings.ReplaceAll(result, "$n", chName)
	result = strings.ReplaceAll(result, "$N", victimName)
	result = strings.ReplaceAll(result, "$e", chPronouns.subjective)
	result = strings.ReplaceAll(result, "$E", victimPronouns.subjective)
	result = strings.ReplaceAll(result, "$s", chPronouns.possessive)
	result = strings.ReplaceAll(result, "$S", victimPronouns.possessive)
	result = strings.ReplaceAll(result, "$m", chPronouns.objective)
	result = strings.ReplaceAll(result, "$M", victimPronouns.objective)
	result = strings.ReplaceAll(result, "#w", singular)
	result = strings.ReplaceAll(result, "#W", plural)
	return capitalizeFightMessage(result)
}

func capitalizeFightMessage(message string) string {
	if message == "" {
		return message
	}
	runes := []rune(message)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// SendWeaponMessage mirrors damage()'s weapon-message routing: misses and
// deaths try the file-backed skill_message table first; ordinary hits use the
// deterministic severity ladder. attackType is the zero-based weapon index.
func SendWeaponMessage(dam int, ch, victim Combatant, attackType int) {
	if dam == 0 || victim.GetHP() <= -11 {
		if cbSkillMessage(dam, ch.GetName(), victim.GetName(), TYPE_HIT+attackType, ch.GetRoom()) {
			return
		}
	}
	DamMessage(dam, ch, victim, attackType)
}

// BackstabMult mirrors backstab_mult() from src/class.c lines 720-729.
// Shared by combat/fight_core.go (circle/disembowel) and game/skill_combat.go (DoBackstab).
//
// C's backstab_mult() returns int, so the expression (level*.2)+1 is computed
// in float and then truncated to int on return. We reproduce that truncation
// (DP-1033): level 14 → 3.8 → 3, level 19 → 4.8 → 4.
func BackstabMult(level int) float64 {
	if level <= 0 {
		return 1.0
	}
	if level >= LVL_IMMORT {
		return 20.0
	}
	// C: return ((level*.2)+1); — float arithmetic truncated to int on return.
	return float64(int(float64(level)*0.2 + 1.0))
}

// **********************************
// 7. groupGain / calcLevelDiff / performGroupGain
// **********************************

func IsInGroup(ch Combatant) bool {
	chName := ch.GetName()
	chRoom := ch.GetRoom()
	if cbHasAffectStr(chName, AFF_STR_GROUP) {
		if ch.GetName() == "" {
			return cbGetFollowersInRoom(chName, chRoom) > 0
		}
		if cbGetMasterInRoom(chName, chRoom) {
			return true
		}
		if cbGetFellowFollowersInRoom(chName, chRoom) {
			return true
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
			share = int(float64(share) - float64(share)*0.7)
		case levelDiff > 10:
			share = int(float64(share) - float64(share)*0.5)
		case levelDiff > 5:
			share = int(float64(share) - float64(share)*0.3)
		}
	}
	if ch.GetLevel() > 20 {
		share = int(float64(share) - float64(share)*0.2)
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
	if !ch.IsNPC() {
		cbGainExp(ch.GetName(), share)
	}
	ChangeAlignment(ch, victim)
}

func GroupGain(ch, victim Combatant) {
	leaderName := ch.GetName()
	if leaderName == "" {
		leaderName = ch.GetName()
	}
	numMembers := cbCountGroupMembers(leaderName, ch.GetRoom())
	if numMembers < 1 {
		numMembers = 1
	}

	victimExp := cbGetExp(victim.GetName())
	base := victimExp / numMembers
	if base > 100 {
		base -= int(float64(base) * 0.01)
	}
	if base < 1 {
		base = 1
	}

	cbApplyToGroupMembers(leaderName, ch.GetRoom(), func(memberName string) {
		m := NewNamedCombatant(memberName, ch.GetRoom())
		PerformGroupGain(m, victim, base)
	})
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
	cbRemoveAllAffects(chName)
	cbUnmount(chName)
	DeathCry(ch)

	// Default to corpse unless GetRace tells us the victim is undead/vampire.
	victimRace := cbGetRace(chName)
	makeDust := victimRace == RACE_UNDEAD || victimRace == RACE_VAMPIRE
	if makeDust {
		cbMakeDust(chName, attackType)
	} else {
		cbMakeCorpse(chName, attackType)
	}
	cbExtractChar(chName)
}

// **********************************
// 9. dieWithKiller()
// **********************************

func DieWithKiller(ch, killer Combatant, attackType int) {
	chName := ch.GetName()

	cbGainExp(chName, -cbGetExp(chName)/37)

	if !ch.IsNPC() && callbacks != nil && callbacks.GetConstitution != nil && callbacks.SetConstitution != nil {
		level := ch.GetLevel()
		if level > 5 && GetRoller().Number(0, 3) == 0 { // 25% chance (C: !number(0,3))
			conVal := cbGetConstitution(chName) - 1
			if level > 20 && GetRoller().Number(0, 5) == 0 { // ~17% chance (C: !number(0,5))
				conVal--
			}
			if conVal < 0 {
				conVal = 0
			}
			cbSetConstitution(chName, conVal)
		}
	}

	roomVNum := ch.GetRoom()
	if cbHasScriptFlag(chName, "MS_DEATH") {
		cbRunDeathScript(killer.GetName(), chName, roomVNum)
	}

	RawKill(ch, attackType)
}

// **********************************
// 10. counterProcs()
// **********************************

func CounterProcs(ch Combatant) {
	if ch.IsNPC() {
		return
	}
	kills := cbGetKills(ch.GetName())

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
		cbIncreaseMaxStat(ch.GetName(), "hp")
		cbIncreaseMaxStat(ch.GetName(), "mana")
		cbIncreaseMaxStat(ch.GetName(), "move")
		ch.Heal(ch.GetMaxHP() - ch.GetHP())
	default:
		return
	}

	if reward {
		// Global blessing — heal all connected players
		cbHealAllPlayers()
		// Log milestone
		cbLog(fmt.Sprintf("%s hit %d kills.", ch.GetName(), kills), "NRM", LVL_IMMORT, false)
	}
}

// **********************************
// 14. attitudeLoot()
//
// Port of fight.c:attitude_loot.
// Loots corpse, junks cheap items in two passes (get → junk → wear → get → junk → wear),
// and broadcasts one of 12 randomized brag messages for MOB_AGGR24/MOB_LOOTS attackers.
// **********************************

func AttitudeLoot(ch, victim Combatant) {
	chName := ch.GetName()

	// Phase 1: loot corpse
	cbPerformCommand(chName, fmt.Sprintf("get all corpse of %s", victim.GetName()))

	// C source (fight.c:1128): first junk pass — discard items with cost <= 150
	cbJunkInventoryItems(chName)

	// C source (fight.c:1138): auto-wear anything wearable
	cbPerformCommand(chName, "wear all")

	// C source (fight.c:1139): second get — picks up items from worn containers
	cbPerformCommand(chName, fmt.Sprintf("get all corpse of %s", victim.GetName()))

	// C source (fight.c:1141): second junk pass
	cbJunkInventoryItems(chName)

	// C source (fight.c:1156): auto-wear anything newly acquired
	cbPerformCommand(chName, "wear all")

	// C source (fight.c:1173-1248): brag messages — only for MOB_AGGR24 mobs (already filtered at call site)
	// Additional early outs from C source: ch == victim (already handled at call site — chName != victimName)
	// and !can_speak(ch) — we skip that here; if the mob can't speak, cbBroadChat does nothing.
	BragMessage(ch, victim)
}

// BragMessage sends one of 12 randomized brag messages matching the C source (fight.c:1173-1248).
// The killer brags via cbBroadChat if the victim is a player, or randomly (1-in-21) for mobs.
func BragMessage(ch, victim Combatant) {
	chName := ch.GetName()
	victimName := victim.GetName()
	victimIsNPC := victim.IsNPC()

	// C source: if !IS_MOB(victim) || !number(0,20) — always brag on player kills,
	// 1-in-21 chance on mob kills.
	if victimIsNPC && GetRoller().Number(0, 20) != 0 {
		return
	}

	msg := pickBragMessage(chName, victimName, victimIsNPC, victim.GetSex())
	if msg == "" {
		return
	}

	cbBroadChat(chName, msg)
}

// pickBragMessage returns one of 12 brag messages matching the C source (fight.c:1173-1248).
// Returns empty string if the killer shouldn't speak (certain messages skip mob kills).
func pickBragMessage(chName, victimName string, victimIsNPC bool, victimSex int) string {
	// Get alignment for case 5
	alignment := cbGetAlignment(chName)
	isEvil := alignment <= -350

	// Get kill count for case 6
	kills := cbGetKills(chName)

	// Possessive pronoun matching C HSHR(victim) — sex of the victim
	// Go encoding: 0=male, 1=female, 2=neutral (differs from C SEX_* in structs.h)
	var possessive string
	switch victimSex {
	case 0: // SEX_MALE
		possessive = "his"
	case 1: // SEX_FEMALE
		possessive = "her"
	default:
		possessive = "its"
	}

	// Uniform random pick [1,12] matching C switch(number(1,12))
	switch GetRoller().Number(1, 12) {
	case 1:
		return fmt.Sprintf("I killed %s and looted %s stinkin' corpse!", victimName, possessive)
	case 2:
		return fmt.Sprintf("%s was tough, but had good eq...", victimName)
	case 3:
		return fmt.Sprintf("%s was easy xp.", victimName)
	case 4:
		return fmt.Sprintf("Muhahahaha... %s is dead!", victimName)
	case 5:
		if isEvil {
			return "Now you will see that evil will always triumph, because good is dumb."
		}
		return fmt.Sprintf("%s is dead! R.I.P.", victimName)
	case 6:
		return fmt.Sprintf("Kill number %d: %s.", kills, victimName)
	case 7:
		if victimIsNPC {
			return ""
		}
		return fmt.Sprintf("Oh, did that hurt, %s? *innocent stare*", victimName)
	case 8:
		if victimIsNPC {
			return ""
		}
		return fmt.Sprintf("What the hell was %s doing out of newbie training?", victimName)
	case 9:
		if victimIsNPC {
			return ""
		}
		return fmt.Sprintf("I think I finally found a use for that punk %s: fertilizer!", victimName)
	case 10:
		if victimIsNPC {
			return ""
		}
		return fmt.Sprintf("Hrmm.. Is this your head, %s? *cackle*", victimName)
	case 11:
		if victimIsNPC {
			return ""
		}
		return fmt.Sprintf("Hey %s, was that suicide or did you try to fight back?", victimName)
	default:
		return ""
	}
}

// **********************************
// 15. damMessage()
// **********************************

// **********************************
// 16. stopFighting / setFighting helpers
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
