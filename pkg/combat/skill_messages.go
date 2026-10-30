// Package combat — skill_messages.go
//
// The skill-message path. C loads fight_messages[] from lib/misc/messages at
// boot (src/fight.c:126-137 load_messages) and damage() picks a variant with
// dice(1, number_of_attacks). The port embeds that same corpus
// (lib/misc/messages_embed.go) and parses it with ParseFightMessages; a load
// failure stops startup the way C's exit(1) does.
package combat

import "strings"

// InitEmbeddedFightMessages wires the canonical corpus embedded in the server
// binary. This is the production initializer; the file-based form remains
// available for tools and parser tests.
func InitEmbeddedFightMessages(cb *GameCallbacks) error {
	messages, err := LoadEmbeddedFightMessages()
	if err != nil {
		return err
	}
	InitFightMessages(cb, messages)
	return nil
}

// C's color levels (src/screen.h:43-46): PRF_COLOR_1 counts 1 (C_SPR) and
// PRF_COLOR_2 counts 2 (C_NRM), so both bits give C_CMP 3. skill_message and
// damage_message gate their escapes on C_CMP; damage()'s wounded-band line
// gates on C_SPR instead, so a partially colored player still sees that one
// (src/fight.c:1596).
const (
	colorLevelSPR = 1 // C_SPR (src/screen.h:44)
	colorLevelCMP = 3 // C_CMP (src/screen.h:46)

	skillColorYellow = "\x1b[33m" // KYEL (src/screen.h:35) as CCYEL emits it (src/screen.h:53)
	skillColorRed    = "\x1b[31m" // KRED (src/screen.h:33) as CCRED emits it (src/screen.h:51)
	skillColorReset  = "\x1b[0m"  // KNRM (src/screen.h:32) as CCNRM emits it (src/screen.h:50)
)

// bleedLineWoundedText is damage()'s wounded-band message (src/fight.c:1594).
const bleedLineWoundedText = "You wish that your wounds would stop BLEEDING so much!"

// sendColorBracketed reproduces C's direct-write color sandwich around an act()
// line:
//
//	send_to_char(CCxxx(name, lvl), name);
//	act(message, ..., TO_CHAR | TO_VICT);
//	send_to_char(CCNRM(name, lvl), name);
//
// Both escapes are direct writes, so they sit outside act()'s SENDOK gate: a
// sleeping, dead or PLR_WRITING recipient still receives them while the message
// itself is suppressed (src/comm.c:2481-2484 defines SENDOK). color is the
// caller's own gate — the recipient's color level, and for skill_message also
// the branch, since the god_msg arm is uncolored in C.
func sendColorBracketed(name Combatant, code, message string, toSleep, color bool) {
	if name == nil {
		return
	}
	if color {
		cbSendRaw(name, code)
	}
	if message != "" && skillActSendOK(name, toSleep) {
		cbSendToChar(name, message)
	}
	if color {
		cbSendRaw(name, skillColorReset)
	}
}

// BleedLine is damage()'s wounded-band line (src/fight.c:1593-1597):
//
//	sprintf(buf2, "%sYou wish that your wounds would stop BLEEDING so much!%s\r\n",
//	        CCRED(victim, C_SPR), CCNRM(victim, C_SPR));
//
// The escapes are part of that one string, so the reset lands before the CRLF.
// The gate is C_SPR, one level below the C_CMP damage-message gate, so a player
// with either color bit set sees the red. Exported for the game layer's own
// copy of the position switch (pkg/game/spec_fighter.go).
func BleedLine(victim Combatant) string {
	if cbGetColorLevel(victim) >= colorLevelSPR {
		return skillColorRed + bleedLineWoundedText + skillColorReset + "\r\n"
	}
	return bleedLineWoundedText + "\r\n"
}

// InitFightMessages wires a parsed C fight_messages table into cb. Selection
// uses dice(1, N), consuming exactly one shared RNG draw even when N is one.
func InitFightMessages(cb *GameCallbacks, messages FightMessages) {
	if cb == nil {
		return
	}

	cb.SkillMessage = func(dam int, ch, victim Combatant, attackType int, roomVNum int) bool {
		variants, ok := messages.Variants(attackType)
		if !ok || len(variants) == 0 {
			return false
		}

		selection := GetRoller().Dice(1, len(variants)) - 1
		if selection < 0 || selection >= len(variants) {
			return false
		}

		variant := variants[selection]

		// fight.c:1039-1090 branch order: god_msg is tested first, then a
		// non-zero dam splits into die (POS_DEAD) and hit, and a zero-damage
		// strike only reaches miss_msg when ch != vict. When dam is zero and
		// ch == vict C falls through with no message at all — after the dice
		// draw above has already consumed its RNG value.
		var action FightMessageAction
		colored := false
		victimToSleep := false
		switch {
		case !cbIsNPC(victim) && cbGetLevel(victim) >= LVL_IMMORT:
			action = variant.God
		case dam != 0:
			// update_pos sets POS_DEAD at GET_HIT <= -11 (fight.c:193).
			if cbGetHP(victim) <= -11 {
				action = variant.Die
			} else {
				action = variant.Hit
			}
			colored = true
			victimToSleep = true
		case ch != victim:
			action = variant.Miss
			colored = true
			victimToSleep = true
		default:
			return true
		}

		chSex := cbGetSex(ch)
		victimSex := cbGetSex(victim)
		render := func(message string) string {
			rendered := replaceMessageTokens(message, ch.GetName(), victim.GetName(), "", "", chSex, victimSex)
			return strings.ReplaceAll(rendered, "$p", cbWeaponDescription(ch))
		}
		sendColored := func(name Combatant, code, message string, toSleep bool) {
			// C's act() applies SENDOK after the direct color write. A
			// sleeping/dead TO_CHAR recipient therefore still receives the
			// surrounding color controls but not the attacker line.
			sendColorBracketed(name, code, message, toSleep,
				colored && cbGetColorLevel(name) >= colorLevelCMP)
		}

		if action.Room != "" && cb.Broadcast != nil {
			cb.Broadcast(roomVNum, render(action.Room), []Combatant{ch, victim})
		}
		sendColored(ch, skillColorYellow, render(action.Attacker), false)
		sendColored(victim, skillColorRed, render(action.Victim), victimToSleep)
		return true
	}
}

// skillActSendOK mirrors comm.c:2481-2482 SENDOK for the direct act() routes
// used by skill_message. TO_CHAR and the god victim act omit TO_SLEEP; the
// ordinary victim act includes it. Direct color writes are deliberately
// outside this gate, as in fight.c:1049-1054, 1064-1069, and 1080-1085.
func skillActSendOK(body Combatant, toSleep bool) bool {
	if body == nil || (!toSleep && body.GetPosition() <= PosSleeping) {
		return false
	}
	return !cbHasPlrFlag(body, "writing")
}

type pronounSet struct {
	subjective string // he/she/it
	objective  string // him/her/it
	possessive string // his/her/its
}

func sexPronouns(sex int) pronounSet {
	switch sex {
	case 1: // female
		return pronounSet{"she", "her", "her"}
	case 2: // neutral/it
		return pronounSet{"it", "it", "its"}
	default: // male
		return pronounSet{"he", "him", "his"}
	}
}
