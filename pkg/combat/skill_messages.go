// Package combat — skill_messages.go
// CRIT-010: Multi-variant skill-specific combat messages.
//
// In CircleMUD, fight_messages[] was loaded from misc/messages via load_messages().
// Each skill had multiple hit/miss/die message variants, randomly selected per use.
// This restores that flavor for Dark Pawns.
package combat

import "strings"

// skillMsgTriplet holds room/char/victim perspectives for one message variant.
type skillMsgTriplet struct {
	Room   string
	Char   string
	Victim string
}

// skillMessageEntry holds all message variants for a skill's hit, miss, and die outcomes.
type skillMessageEntry struct {
	Hit  []skillMsgTriplet
	Miss []skillMsgTriplet
	Die  []skillMsgTriplet
}

// ---------------------------------------------------------------------------
// Skill message table — keyed by attack type (skill number constants from C).
// These match the Skill*Num constants in pkg/game/death.go.
// ---------------------------------------------------------------------------

var skillMessageTable = map[int]skillMessageEntry{
	// --- BACKSTAB (131) ---
	131: {
		Hit: []skillMsgTriplet{
			{
				"$n plunges $s blade deep into $N's back!",
				"You plunge your blade deep into $N's back!",
				"$n plunges a blade deep into your back!",
			},
			{
				"$n drives a knife between $N's shoulder blades!",
				"You drive a knife between $N's shoulder blades!",
				"$n drives a knife between your shoulder blades!",
			},
			{
				"$n slips behind $N and sinks steel into $s spine!",
				"You slip behind $N and sink steel into $s spine!",
				"$n slips behind you and sinks steel into your spine!",
			},
			{
				"$n catches $N off-guard with a vicious backstab!",
				"You catch $N off-guard with a vicious backstab!",
				"$n catches you off-guard with a vicious backstab!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n's backstab glances off $N's armor!",
				"Your backstab glances off $N's armor!",
				"$n's backstab glances off your armor!",
			},
			{
				"$n tries to slip behind $N, but $N turns just in time!",
				"You try to slip behind $N, but $N turns just in time!",
				"$n tries to slip behind you, but you turn just in time!",
			},
			{
				"$n fumbles $s backstab attempt on $N.",
				"You fumble your backstab attempt on $N.",
				"$n fumbles $s backstab attempt on you.",
			},
		},
		Die: []skillMsgTriplet{
			{
				"$n's blade finds $N's heart! $N crumples to the ground, DEAD!",
				"Your blade finds $N's heart! $N crumples to the ground, DEAD!",
				"$n's blade finds your heart! Darkness takes you...",
			},
			{
				"$n removes $N from this world with a single, perfect strike!",
				"You remove $N from this world with a single, perfect strike!",
				"$n removes you from this world with a single, perfect strike!",
			},
		},
	},

	// --- BASH (132) ---
	132: {
		Hit: []skillMsgTriplet{
			{
				"$n slams into $N, sending $M crashing to the ground!",
				"You slam into $N, sending $M crashing to the ground!",
				"$n slams into you! You crash to the ground!",
			},
			{
				"$n bashes $N with bone-rattling force!",
				"You bash $N with bone-rattling force!",
				"$n bashes you with bone-rattling force!",
			},
			{
				"$n charges into $N and sends $M sprawling!",
				"You charge into $N and send $M sprawling!",
				"$n charges into you and sends you sprawling!",
			},
			{
				"$n's brutal bash sends $N to the floor!",
				"Your brutal bash sends $N to the floor!",
				"$n's brutal bash sends you to the floor!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n's bash attempt falls short of $N.",
				"Your bash attempt falls short of $N.",
				"$n's bash attempt falls short of you.",
			},
			{
				"$n stumbles trying to bash $N and nearly falls!",
				"You stumble trying to bash $N and nearly fall!",
				"$n stumbles trying to bash you and nearly falls!",
			},
		},
		Die: []skillMsgTriplet{
			{
				"$n bashes $N's skull in! That's gonna leave a mark.",
				"You bash $N's skull in! That's gonna leave a mark.",
				"$n bashes your skull in! Everything goes dark.",
			},
		},
	},

	// --- KICK (134) ---
	134: {
		Hit: []skillMsgTriplet{
			{
				"$n connects with a solid kick to $N!",
				"You connect with a solid kick to $N!",
				"$n connects with a solid kick to you!",
			},
			{
				"$n's foot slams into $N with a sickening thud!",
				"Your foot slams into $N with a sickening thud!",
				"$n's foot slams into you with a sickening thud!",
			},
			{
				"$n delivers a sharp kick to $N's midsection!",
				"You deliver a sharp kick to $N's midsection!",
				"$n delivers a sharp kick to your midsection!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n's kick sails harmlessly past $N.",
				"Your kick sails harmlessly past $N.",
				"$n's kick sails harmlessly past you.",
			},
			{
				"$n tries to kick $N but loses $s balance!",
				"You try to kick $N but lose your balance!",
				"$n tries to kick you but loses $s balance!",
			},
		},
		Die: []skillMsgTriplet{
			{
				"$n's killing kick snaps $N's neck!",
				"Your killing kick snaps $N's neck!",
				"$n's killing kick snaps your neck!",
			},
		},
	},

	// --- PUNCH (136) ---
	136: {
		Hit: []skillMsgTriplet{
			{
				"$n clocks $N with a brutal punch!",
				"You clock $N with a brutal punch!",
				"$n clocks you with a brutal punch!",
			},
			{
				"$n's fist connects hard with $N's jaw!",
				"Your fist connects hard with $N's jaw!",
				"$n's fist connects hard with your jaw!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n's wild punch misses $N by a mile.",
				"Your wild punch misses $N by a mile.",
				"$n's wild punch misses you by a mile.",
			},
		},
		Die: nil,
	},

	// --- DRAGON KICK (188) ---
	188: {
		Hit: []skillMsgTriplet{
			{
				"$n unleashes a devastating dragon kick on $N!",
				"You unleash a devastating dragon kick on $N!",
				"$n unleashes a devastating dragon kick on you!",
			},
			{
				"$n's spinning dragon kick catches $N full in the chest!",
				"Your spinning dragon kick catches $N full in the chest!",
				"$n's spinning dragon kick catches you full in the chest!",
			},
			{
				"$n launches into the air and comes down on $N with a dragon kick!",
				"You launch into the air and come down on $N with a dragon kick!",
				"$n launches into the air and comes down on you with a dragon kick!",
			},
			{
				"$n's foot erupts into $N like a thunderbolt!",
				"Your foot erupts into $N like a thunderbolt!",
				"$n's foot erupts into you like a thunderbolt!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n's dragon kick misses $N completely!",
				"Your dragon kick misses $N completely!",
				"$n's dragon kick misses you completely!",
			},
			{
				"$n spins wildly but can't land the dragon kick on $N!",
				"You spin wildly but can't land the dragon kick on $N!",
				"$n spins wildly but can't land the dragon kick on you!",
			},
			{
				"$n stumbles out of $s dragon kick and nearly faceplants!",
				"You stumble out of your dragon kick and nearly faceplant!",
				"$n stumbles out of $s dragon kick and nearly faceplants!",
			},
		},
		Die: []skillMsgTriplet{
			{
				"$n's dragon kick caves in $N's ribcage! $N is DEAD!",
				"Your dragon kick caves in $N's ribcage! $N is DEAD!",
				"$n's dragon kick caves in your ribcage! Everything goes black.",
			},
			{
				"$n shatters $N with a legendary dragon kick!",
				"You shatter $N with a legendary dragon kick!",
				"$n shatters you with a legendary dragon kick!",
			},
		},
	},

	// --- TIGER PUNCH (223) ---
	223: {
		Hit: []skillMsgTriplet{
			{
				"$n's fist connects with $N like a freight train!",
				"Your fist connects with $N like a freight train!",
				"$n's fist connects with you like a freight train!",
			},
			{
				"$n delivers a devastating tiger punch to $N!",
				"You deliver a devastating tiger punch to $N!",
				"$n delivers a devastating tiger punch to you!",
			},
			{
				"$n's iron fist slams into $N with terrifying force!",
				"Your iron fist slams into $N with terrifying force!",
				"$n's iron fist slams into you with terrifying force!",
			},
			{
				"$n channels the fury of the tiger and strikes $N!",
				"You channel the fury of the tiger and strike $N!",
				"$n channels the fury of the tiger and strikes you!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n's tiger punch whiffs past $N!",
				"Your tiger punch whiffs past $N!",
				"$n's tiger punch whiffs past you!",
			},
			{
				"$n fails to connect with the tiger punch on $N.",
				"You fail to connect with the tiger punch on $N.",
				"$n fails to connect with the tiger punch on you.",
			},
		},
		Die: []skillMsgTriplet{
			{
				"$n's tiger punch ends $N — permanently!",
				"Your tiger punch ends $N — permanently!",
				"$n's tiger punch ends you — permanently!",
			},
		},
	},

	// --- DISEMBOWEL (184) ---
	184: {
		Hit: []skillMsgTriplet{
			{
				"$n disembowels $N with a vicious upward slash!",
				"You disembowel $N with a vicious upward slash!",
				"$n disembowels you with a vicious upward slash!",
			},
			{
				"$n's blade tears into $N's guts! Blood sprays everywhere!",
				"Your blade tears into $N's guts! Blood sprays everywhere!",
				"$n's blade tears into your guts! Blood sprays everywhere!",
			},
			{
				"$n opens $N up from hip to ribcage!",
				"You open $N up from hip to ribcage!",
				"$n opens you up from hip to ribcage!",
			},
			{
				"$n rips into $N with a gut-wrenching disembowelment!",
				"You rip into $N with a gut-wrenching disembowelment!",
				"$n rips into you with a gut-wrenching disembowelment!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n's disembowel attempt scrapes harmlessly across $N's armor!",
				"Your disembowel attempt scrapes harmlessly across $N's armor!",
				"$n's disembowel attempt scrapes harmlessly across your armor!",
			},
			{
				"$n tries to gut $N but $N twists away!",
				"You try to gut $N but $N twists away!",
				"$n tries to gut you but you twist away!",
			},
		},
		Die: []skillMsgTriplet{
			{
				"$n spills $N's guts across the floor! $N is DEAD!",
				"You spill $N's guts across the floor! $N is DEAD!",
				"$n spills your guts across the floor! The world fades to red...",
			},
			{
				"$n's blade finds $N's belly and doesn't stop until it hits spine!",
				"Your blade finds $N's belly and doesn't stop until it hits spine!",
				"$n's blade finds your belly and doesn't stop until it hits spine!",
			},
		},
	},

	// --- BITE (150) ---
	150: {
		Hit: []skillMsgTriplet{
			{
				"$n sinks $s teeth into $N!",
				"You sink your teeth into $N!",
				"$n sinks $s teeth into you!",
			},
			{
				"$n chomps down hard on $N!",
				"You chomp down hard on $N!",
				"$n chomps down hard on you!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n snaps at $N but gets only air!",
				"You snap at $N but get only air!",
				"$n snaps at you but gets only air!",
			},
		},
		Die: nil,
	},

	// --- HEADBUTT (141) ---
	141: {
		Hit: []skillMsgTriplet{
			{
				"$n headbutts $N with a sickening crack!",
				"You headbutt $N with a sickening crack!",
				"$n headbutts you with a sickening crack!",
			},
			{
				"$n's skull slams into $N's face!",
				"Your skull slams into $N's face!",
				"$n's skull slams into your face!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n's headbutt misses $N and $N grins.",
				"Your headbutt misses $N and $N grins.",
				"$n's headbutt misses you. You grin.",
			},
		},
		Die: nil,
	},

	// --- SERPENT KICK (156) ---
	156: {
		Hit: []skillMsgTriplet{
			{
				"$n strikes $N with a blindingly fast serpent kick!",
				"You strike $N with a blindingly fast serpent kick!",
				"$n strikes you with a blindingly fast serpent kick!",
			},
			{
				"$n's leg whips around into $N like a viper's strike!",
				"Your leg whips around into $N like a viper's strike!",
				"$n's leg whips around into you like a viper's strike!",
			},
			{
				"$n's serpent kick catches $N square in the temple!",
				"Your serpent kick catches $N square in the temple!",
				"$n's serpent kick catches you square in the temple!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n's serpent kick misses $N by inches!",
				"Your serpent kick misses $N by inches!",
				"$n's serpent kick misses you by inches!",
			},
		},
		Die: nil,
	},

	// --- CIRCLE (173) ---
	173: {
		Hit: []skillMsgTriplet{
			{
				"$n circles around and strikes $N from behind!",
				"You circle around and strike $N from behind!",
				"$n circles around and strikes you from behind!",
			},
			{
				"$n sidesteps and catches $N in $s blind spot!",
				"You sidestep and catch $N in $s blind spot!",
				"$n sidesteps and catches you in your blind spot!",
			},
			{
				"$n slips around $N's guard with a precise circle strike!",
				"You slip around $N's guard with a precise circle strike!",
				"$n slips around your guard with a precise circle strike!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n tries to circle $N but can't find an opening!",
				"You try to circle $N but can't find an opening!",
				"$n tries to circle you but can't find an opening!",
			},
		},
		Die: nil,
	},

	// --- SLEEPER (187) ---
	187: {
		Hit: []skillMsgTriplet{
			{
				"$n wraps $s arms around $N in a sleeper hold!",
				"You wrap your arms around $N in a sleeper hold!",
				"$n wraps $s arms around you in a sleeper hold!",
			},
			{
				"$n locks $N in a vice-like sleeper!",
				"You lock $N in a vice-like sleeper!",
				"$n locks you in a vice-like sleeper!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n's sleeper attempt fails — $N slips free!",
				"Your sleeper attempt fails — $N slips free!",
				"$n's sleeper attempt fails — you slip free!",
			},
		},
		Die: nil,
	},

	// --- NECKBREAK (190) ---
	190: {
		Hit: []skillMsgTriplet{
			{
				"$n snaps $N's neck with a brutal twist!",
				"You snap $N's neck with a brutal twist!",
				"$n snaps your neck with a brutal twist!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n grabs for $N's neck but $N breaks free!",
				"You grab for $N's neck but $N breaks free!",
				"$n grabs for your neck but you break free!",
			},
		},
		Die: []skillMsgTriplet{
			{
				"$n breaks $N's neck! A wet snap echoes through the room.",
				"You break $N's neck! A wet snap echoes through the room.",
				"$n breaks your neck. The world goes silent.",
			},
		},
	},

	// --- SLUG (146) ---
	146: {
		Hit: []skillMsgTriplet{
			{
				"$n slugs $N across the jaw!",
				"You slug $N across the jaw!",
				"$n slugs you across the jaw!",
			},
			{
				"$n connects with a massive haymaker on $N!",
				"You connect with a massive haymaker on $N!",
				"$n connects with a massive haymaker on you!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n's slug goes wide of $N!",
				"Your slug goes wide of $N!",
				"$n's slug goes wide of you!",
			},
		},
		Die: nil,
	},

	// --- SMACKHEADS (145) ---
	145: {
		Hit: []skillMsgTriplet{
			{
				"$n smacks $N's heads together!",
				"You smack $N's heads together!",
				"$n smacks your heads together!",
			},
		},
		Miss: []skillMsgTriplet{
			{
				"$n tries to smack $N's heads but misses!",
				"You try to smack $N's heads but miss!",
				"$n tries to smack your heads but misses!",
			},
		},
		Die: nil,
	},
}

// ---------------------------------------------------------------------------
// InitSkillMessages wires the skill-specific message lookup into the provided
// callbacks. If cb is nil, it falls back to the legacy SkillMessageFunc package
// variable so existing tests keep working until the migration is complete.
//
// Call during combat system initialization after Broadcast/SendToChar hooks are
// available. If called before those hooks are set, messages will still work —
// they just won't be delivered until the hooks are assigned.
func InitSkillMessages(cb *GameCallbacks) {
	impl := func(dam int, ch, victim Combatant, attackType int, roomVNum int) bool {
		entry, ok := skillMessageTable[attackType]
		if !ok {
			return false // no custom messages for this skill
		}

		// Select variant list based on outcome.
		// dam == 0 → miss. For die, callers should use DamMessage or death flow.
		var variants []skillMsgTriplet
		if dam == 0 {
			variants = entry.Miss
		} else {
			variants = entry.Hit
		}

		if len(variants) == 0 {
			return false // no messages for this outcome
		}

		msg := variants[GetRoller().Number(0, len(variants)-1)]

		// We don't have sex info here, so do basic token substitution only.
		roomMsg := basicTokenReplace(msg.Room, ch, victim)
		charMsg := basicTokenReplace(msg.Char, ch, victim)
		victimMsg := basicTokenReplace(msg.Victim, ch, victim)

		if cb != nil {
			if cb.Broadcast != nil {
				cb.Broadcast(roomVNum, roomMsg, []Combatant{ch, victim})
			}
			if cb.SendToChar != nil {
				if skillActSendOK(ch, false) {
					cb.SendToChar(ch, charMsg)
				}
				if skillActSendOK(victim, true) {
					cb.SendToChar(victim, victimMsg)
				}
			}
		}
		return true
	}

	if cb != nil {
		cb.SkillMessage = impl
	}
}

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

// basicTokenReplace handles $n/$N/$s/$e substitution with sex-aware pronouns.
// Falls back to male pronouns if the GetSex callback is not wired.
func basicTokenReplace(msg string, ch, victim Combatant) string {
	chName, victimName := ch.GetName(), victim.GetName()
	chSex := 0 // default male
	if s := cbGetSex(ch); s >= 0 {
		chSex = s
	}
	victimSex := 0
	if s := cbGetSex(victim); s >= 0 {
		victimSex = s
	}
	return replaceMessageTokens(msg, chName, victimName, "", "", chSex, victimSex)
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
