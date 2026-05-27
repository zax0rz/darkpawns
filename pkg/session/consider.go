package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// cmdConsider compares the player against a target.
// Source: act.informative.c ACMD(do_consider) lines 2330-2450
func cmdConsider(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Consider killing who?")
		return nil
	}

	targetName := strings.TrimSpace(strings.Join(args, " "))
	roomVNum := s.player.GetRoom()

	// Check self
	if strings.EqualFold(targetName, s.player.Name) {
		s.Send("Easy!  Very easy indeed!")
		return nil
	}

	// Look for target in room — players first, then mobs
	var targetNameFound string
	var targetSex int
	var targetLevel int
	var targetHP int

	// Check players
	players := s.manager.world.GetPlayersInRoom(roomVNum)
	var targetPlayer *game.Player
	for _, p := range players {
		if strings.EqualFold(p.Name, targetName) && p.Name != s.player.Name {
			targetPlayer = p
			targetNameFound = p.Name
			targetSex = p.Sex
			targetLevel = p.Level
			targetHP = p.Health
			break
		}
	}

	// Check mobs if no player found
	var targetMob *game.MobInstance
	if targetPlayer == nil {
		mobs := s.manager.world.GetMobsInRoom(roomVNum)
		for _, m := range mobs {
			mobName := m.GetName()
			if mobName == "" {
				mobName = m.GetShortDesc()
			}
			if strings.EqualFold(mobName, targetName) || strings.EqualFold(m.GetShortDesc(), targetName) {
				targetMob = m
				targetNameFound = mobName
				targetSex = m.GetSex()
				targetLevel = m.GetLevel()
				targetHP = m.GetHP()
				break
			}
		}
	}

	if targetNameFound == "" {
		s.Send("They aren't here.")
		return nil
	}

	// Part 1: Damage comparison — from C lines 2390-2412
	// chardam = str_app[STRENGTH_APPLY_INDEX(ch)].todam + GET_DAMROLL(ch) + weapon_dice
	// victdam = same for target

	// todam values from src/constants.c str_app[]: indices 0-30
	// Index 0-30 correspond to STR 0-30, with indices 26-30 for 18/xx exceptional strengths
	todamTable := []int{
		-4, -4, -2, -1, -1, -1, 0, 0, 0, 0, // 0-9
		0, 0, 0, 0, 0, 0, 1, 1, 2, 7, // 10-19
		8, 9, 10, 11, 12, 14, // 20-25
		3, 3, 4, 5, 6, // 26-30: 18/01-50, 18/51-75, 18/76-90, 18/91-99, 18/100
	}

	// Player's base damage from strength
	pStrIdx := strengthApplyIndex(s.player.Stats.Str, s.player.Stats.StrAdd)
	chardam := 0
	if pStrIdx >= 0 && pStrIdx < len(todamTable) {
		chardam = todamTable[pStrIdx]
	}
	chardam += s.player.Damroll

	// Player's wielded weapon (SlotWield)
	charweapon, hasWeapon := s.player.Equipment.GetItemInSlot(game.SlotWield)
	if hasWeapon && charweapon != nil && charweapon.Prototype != nil {
		// Values[1] = numdice, Values[2] = sizedice — from C: dice(GET_OBJ_VAL(charwielded, 1), GET_OBJ_VAL(charwielded, 2))
		numDice := charweapon.Prototype.Values[1]
		sizeDice := charweapon.Prototype.Values[2]
		if numDice < 1 {
			numDice = 1
		}
		if sizeDice < 1 {
			sizeDice = 4
		}
		chardam += rollDice(numDice, sizeDice)
	} else {
		// Bare-handed: number(0, GET_LEVEL(ch)/3)
		if s.player.Level > 0 {
			// emulate number(0, N) = random 0..N
			chardam += rollDice(1, s.player.Level/3) - 1
			if chardam < 0 {
				chardam = 0
			}
		}
	}

	// Target's base damage from strength
	var victdam int
	if targetPlayer != nil {
		tStrIdx := strengthApplyIndex(targetPlayer.Stats.Str, targetPlayer.Stats.StrAdd)
		if tStrIdx >= 0 && tStrIdx < len(todamTable) {
			victdam = todamTable[tStrIdx]
		}
		victdam += targetPlayer.Damroll

		// Target's wielded weapon
		victWeapon, victHasWeapon := targetPlayer.Equipment.GetItemInSlot(game.SlotWield)
		if victHasWeapon && victWeapon != nil && victWeapon.Prototype != nil {
			numDice := victWeapon.Prototype.Values[1]
			sizeDice := victWeapon.Prototype.Values[2]
			if numDice < 1 {
				numDice = 1
			}
			if sizeDice < 1 {
				sizeDice = 4
			}
			victdam += rollDice(numDice, sizeDice)
		} else {
			if targetPlayer.Level > 0 {
				victdam += rollDice(1, targetPlayer.Level/3) - 1
				if victdam < 0 {
					victdam = 0
				}
			}
		}
	} else if targetMob != nil {
		// NPC: use mob damage dice
		damageRoll := targetMob.GetDamageRoll()
		if damageRoll.Num > 0 && damageRoll.Sides > 0 {
			victdam = rollDice(damageRoll.Num, damageRoll.Sides)
		} else {
			victdam = 0
		}
		victdam += targetMob.GetDamroll()
	}

	damdiff := victdam - chardam

	// Part 1 text: damage comparison — exact strings from C
	var firstLine string
	switch {
	case damdiff > 20:
		firstLine = "$N looks like $E could eat you for lunch, "
	case damdiff > 10:
		firstLine = "$N looks like $E could tear you up in a fight, "
	case damdiff > 5:
		firstLine = "$N looks like $E could hurt you in a fight, "
	case damdiff > -3:
		firstLine = "$N looks like a fair fight, "
	case damdiff > -5:
		firstLine = "$N looks like an easy kill, "
	case damdiff > -10:
		firstLine = "$N looks like a very easy kill, "
	default:
		firstLine = "$N might not even be worth the effort to kill, "
	}

	// Part 2 text: HP comparison — from C: hitdiff = GET_HIT(victim)-GET_HIT(ch)
	// Compared against multiples of player's HP
	hitdiff := targetHP - s.player.Health
	playerHP := s.player.Health

	var secondLine string
	switch {
	case hitdiff > 4*playerHP:
		secondLine = "looks to be\r\nin much better physical shape than you, "
	case hitdiff > 2*playerHP:
		secondLine = "looks to be\r\nin a lot better physical shape than you, "
	case hitdiff > playerHP:
		secondLine = "looks to be\r\nin better physical shape than you, "
	case hitdiff >= 0:
		secondLine = "looks to be\r\nin about the same physical shape as you, "
	case hitdiff > -25:
		secondLine = "looks to be\r\nin a little worse physical shape as you, "
	case hitdiff > -50:
		secondLine = "looks to be\r\nin a worse physical shape than you, "
	default:
		secondLine = "looks to be\r\nin a lot worse physical shape than you, "
	}

	// Part 3 text: level confidence — from C lines 2432-2448
	leveldiff := targetLevel - s.player.Level

	var thirdLine string
	switch {
	case leveldiff > s.player.Level:
		thirdLine = "and moves with an ease telling\r\nof many won battles."
	case leveldiff > s.player.Level/2:
		thirdLine = "and seems to know $S opponent."
	case leveldiff >= 0:
		thirdLine = "and seems about as confident in\r\nbattle as you do."
	case leveldiff > -3:
		thirdLine = "and seems less confident in\r\nbattle than you do."
	case leveldiff > -5:
		thirdLine = "and seems much less confident in\r\nbattle than you do."
	case leveldiff > -7:
		thirdLine = "and seems ready to run from a\r\nfight."
	default:
		thirdLine = "and seems like $E's never been\r\nin battle before."
	}

	// Build full message and resolve pronouns
	msg := resolvePronouns(firstLine+secondLine+thirdLine, targetNameFound, targetSex)

	s.Send(msg)

	// Broadcast consider action to the room
	broadcastConsider(s, targetNameFound, roomVNum)

	return nil
}

// strengthApplyIndex implements STRENGTH_APPLY_INDEX from utils.h line 440.
func strengthApplyIndex(str int, strAdd int) int {
	if strAdd == 0 || str != 18 {
		return str
	}
	switch {
	case strAdd <= 50:
		return 26 // 18/01-50
	case strAdd <= 75:
		return 27 // 18/51-75
	case strAdd <= 90:
		return 28 // 18/76-90
	case strAdd <= 99:
		return 29 // 18/91-99
	default:
		return 30 // 18/100
	}
}

// rollDice implements dice() from C class.c: dice(num, sides)
func rollDice(num, sides int) int {
	if num <= 0 || sides <= 0 {
		return 0
	}
	total := 0
	for i := 0; i < num; i++ {
		total += (1 + rand.IntN(sides))
	}
	return total
}

// resolvePronouns replaces $N, $E, $S, $M tokens in a consider message.
func resolvePronouns(msg string, targetName string, sex int) string {
	var subject, object, possessive string

	switch sex {
	case 0: // neutral
		subject = "it"
		object = "it"
		possessive = "its"
	case 1: // male
		subject = "he"
		object = "him"
		possessive = "his"
	case 2: // female
		subject = "she"
		object = "her"
		possessive = "her"
	default:
		subject = "it"
		object = "it"
		possessive = "its"
	}

	msg = strings.ReplaceAll(msg, "$N", targetName)
	msg = strings.ReplaceAll(msg, "$E", subject)
	msg = strings.ReplaceAll(msg, "$S", possessive)
	msg = strings.ReplaceAll(msg, "$M", object)

	return msg
}

// broadcastConsider sends the consider action message to everyone in the room except the considerer.
func broadcastConsider(s *Session, targetName string, roomVNum int) {
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "consider",
			From: s.player.Name,
			Text: fmt.Sprintf("%s considers %s.", s.player.Name, targetName),
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	s.manager.BroadcastToRoom(roomVNum, msg, s.player.Name)
}
