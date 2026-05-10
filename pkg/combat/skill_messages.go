// Package combat — skill_messages.go
// CRIT-010: Multi-variant skill-specific combat messages.
//
// In CircleMUD, fight_messages[] was loaded from misc/messages via load_messages().
// Each skill had multiple hit/miss/die message variants, randomly selected per use.
// This restores that flavor for Dark Pawns.
package combat

import (
	"math/rand"
	"strings"
)

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

	// --- DRAGON KICK (222) ---
	222: {
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
// InitSkillMessages wires the global SkillMessageFunc to look up per-skill
// message variants from the skillMessageTable.
//
// Call during combat system initialization (when BroadcastMessage/SendToCharFunc
// are wired). If called before those hooks are set, messages will still work —
// they just won't be delivered until the hooks are assigned.
func InitSkillMessages() {
	SkillMessageFunc = func(dam int, chName, victimName string, attackType int) bool {
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

		msg := variants[rand.Intn(len(variants))]

		// We don't have sex info here, so do basic token substitution only.
		roomMsg := basicTokenReplace(msg.Room, chName, victimName)
		charMsg := basicTokenReplace(msg.Char, chName, victimName)
		victimMsg := basicTokenReplace(msg.Victim, chName, victimName)

		if BroadcastMessage != nil {
			BroadcastMessage(0, roomMsg, chName+" "+victimName) // room broadcast, vnum unknown in this path
		}
		if SendToCharFunc != nil {
			SendToCharFunc(chName, charMsg)
			SendToCharFunc(victimName, victimMsg)
		}
		return true
	}
}

// basicTokenReplace handles $n/$N substitution without sex-aware pronouns.
// Used by SkillMessageFunc which receives name strings, not full Combatant objects.
func basicTokenReplace(msg, chName, victimName string) string {
	result := msg
	result = strings.ReplaceAll(result, "$n", chName)
	result = strings.ReplaceAll(result, "$N", victimName)
	// $s/$e left as-is since we don't have sex — callers that need pronoun
	// resolution should use replaceMessageTokens from fight_core.go instead.
	return result
}
