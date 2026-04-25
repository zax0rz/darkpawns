package spells

import (
	"strings"
)

// SaySpell constructs the verbal component of a spell and sends it to the room.
// Ported from src/spell_parser.c say_spell().
//
// Parameters use interface{} since this package cannot import game/combat directly
// (would create circular deps). Expected types:
//   - ch: implements GetClass() int, IsNPC() bool, SendMessage(string), GetName() string
//   - tch/targetChar: same as ch, or nil
//   - tobj/targetObj: implements GetName() string, or nil
func SaySpell(ch interface{}, spellNum int, tch, tobj interface{}, world interface{}) {
	if ch == nil {
		return
	}

	spellName := GetSpellName(spellNum)
	if spellName == "" {
		return
	}

	obfuscated := ObfuscateSpellName(spellName)

	// Determine the format string based on target presence and class
	var format string
	chClass := getClass(ch)
	chPsionic := isClassPsionicOrMystic(chClass)

	// Build the description verb
	var actionDesc string
	if chPsionic {
		actionDesc = "focuses $s will..."
	} else {
		actionDesc = "utters the words, '%s'."
	}

	if tch != nil && sameRoom(ch, tch) {
		if chPsionic {
			if selfTarget(ch, tch) {
				format = "$n focuses $s will..."
			} else {
				format = "$n stares at $N and focuses $s will..."
			}
		} else {
			if selfTarget(ch, tch) {
				format = "$n closes $s eyes and " + actionDesc
			} else {
				format = "$n stares at $N and " + actionDesc
			}
		}
	} else if tobj != nil && sameRoomWithObj(ch, tobj) {
		if chPsionic {
			format = "$n stares at $p and focuses $s will..."
		} else {
			format = "$n stares at $p and " + actionDesc
		}
	} else {
		if chPsionic {
			format = "$n focuses $s will..."
		} else {
			format = "$n " + actionDesc
		}
	}

	// Send to room — each person sees either real or obfuscated name
	sendToRoom(format, ch, tobj, tch, spellName, obfuscated, world)

	// If target char is in room and not self, send target-specific message
	if tch != nil && !selfTarget(ch, tch) && sameRoom(ch, tch) {
		var targetMsg string
		tchClass := getClass(tch)
		if chPsionic {
			targetMsg = "$n focuses $s will on you..."
		} else {
			spellForTarget := spellName
			if chClass != tchClass {
				spellForTarget = obfuscated
			}
			targetMsg = "$n stares at you and utters the words, '" + spellForTarget + "'."
		}
		sendAct(targetMsg, ch, nil, tch, world)
	}
}

// ObfuscateSpellName applies syllable substitution to generate gibberish.
// Ported from src/spell_parser.c.
func ObfuscateSpellName(name string) string {
	if name == "" {
		return ""
	}

	// Syllable substitution table from spells.h
	type syllable struct {
		org, new string
	}
	syls := []syllable{
		{" ", " "},
		{"ar", "abra"},
		{"ate", "i"},
		{"cau", "kada"},
		{"blind", "nose"},
		{"bur", "mosa"},
		{"cu", "judi"},
		{"de", "oculo"},
		{"dis", "mar"},
		{"ect", "kamina"},
		{"en", "uns"},
		{"gro", "cra"},
		{"light", "dies"},
		{"lo", "hi"},
		{"magi", "kari"},
		{"mon", "bar"},
		{"mor", "zak"},
		{"move", "sido"},
		{"ness", "lacri"},
		{"ning", "illa"},
		{"per", "duda"},
		{"ra", "gru"},
		{"re", "candus"},
		{"son", "sabru"},
		{"tect", "infra"},
		{"tri", "cula"},
		{"ven", "nofo"},
		{"word of", "inset"},
		{"a", "i"}, {"b", "v"}, {"c", "q"}, {"d", "m"}, {"e", "o"}, {"f", "y"},
		{"g", "t"},
		{"h", "p"}, {"i", "u"}, {"j", "y"}, {"k", "t"}, {"l", "r"}, {"m", "w"},
		{"n", "b"},
		{"o", "a"}, {"p", "s"}, {"q", "d"}, {"r", "f"}, {"s", "g"}, {"t", "h"},
		{"u", "e"},
		{"v", "z"}, {"w", "x"}, {"x", "n"}, {"y", "l"}, {"z", "k"}, {"", ""},
	}

	var result strings.Builder
	ofs := 0
	lower := strings.ToLower(name)

	for ofs < len(lower) {
		matched := false
		for _, syl := range syls {
			if syl.org == "" {
				break
			}
			if ofs+len(syl.org) <= len(lower) && lower[ofs:ofs+len(syl.org)] == syl.org {
				result.WriteString(syl.new)
				ofs += len(syl.org)
				matched = true
				break
			}
		}
		if !matched {
			// Keep original character
			result.WriteByte(lower[ofs])
			ofs++
		}
	}

	return result.String()
}

// GetSpellName returns the name for a spell number, or "" if unknown.
func GetSpellName(spellNum int) string {
	if spellNum >= 0 && spellNum < len(spellNamesTable) {
		return spellNamesTable[spellNum]
	}
	return ""
}

// spellNamesTable maps spell number -> name. Indexed by spell number.
// Extended list matching the C spells[] array from spell_parser.c.
var spellNamesTable = []string{
	"", // 0 — undefined
	"armor",
	"teleport",
	"bless",
	"blindness",
	"burning hands",
	"call lightning",
	"charm",
	"chill touch",
	"clone",
	"color spray",
	"control weather",
	"create food",
	"create water",
	"cure blind",
	"cure critical",
	"cure light",
	"curse",
	"detect alignment",
	"detect invis",
	"detect magic",
	"detect poison",
	"dispel evil",
	"earthquake",
	"enchant weapon",
	"energy drain",
	"fireball",
	"harm",
	"heal",
	"invisible",
	"lightning bolt",
	"locate object",
	"magic missile",
	"poison",
	"protect evil",
	"remove curse",
	"sanctuary",
	"shocking grasp",
	"sleep",
	"strength",
	"summon",
	"meteor swarm",
	"recall",
	"remove poison",
	"sense life",
	"protect good",
	"dispel good",
	"holy shield",
	"group heal",
	"group recall",
	"infravision",
	"waterwalk",
	"mass heal",
	"fly",
	"lycanthropy",
	"vampirism",
	"sobriety",
	"group invis",
	"hellfire",
	"enchant armor",
	"identify",
	"mind poke",
	"mind blast",
	"chameleon",
	"levitate",
	"metalskin",
	"globe of invulnerability",
	"vitality",
	"invigorate",
	"lesser perception",
	"greater perception",
	"mind attack",
	"adrenaline boost",
	"psychic shield",
	"change density",
	"acid blast",
	"dominate",
	"cell adjustment",
	"zen",
	"mirror image",
	"mass dominate",
	"divine intervention",
	"mind bar",
	"soul leech",
	"mindsight",
	"transparency",
	"know align",
	"gate",
	"word of intellect",
	"lay hands",
	"mental lapse",
	"smokescreen",
	"ray of disruption",
	"disintegration",
	"calliope",
	"protection from good",
	"flame strike",
	"haste",
	"slow",
	"dream travel",
	"psiblast",
	"glyph of summoning",
	"waterbreathe",
	"drowning",
	"petrify",
	"conjure elemental",
}

// isClassPsionicOrMystic returns true for psionic (5) or mystic (7) classes.
func isClassPsionicOrMystic(class int) bool {
	return class == 5 || class == 7
}

// getClass extracts the class from an interface{} that implements GetClass() int.
func getClass(ch interface{}) int {
	if ch == nil {
		return 0
	}
	type classer interface{ GetClass() int }
	if c, ok := ch.(classer); ok {
		return c.GetClass()
	}
	return 0
}

// selfTarget checks if ch and tch are the same entity.
func selfTarget(ch, tch interface{}) bool {
	if ch == nil || tch == nil {
		return false
	}
	type namer interface{ GetName() string }
	cn, cok := ch.(namer)
	tn, tok := tch.(namer)
	if cok && tok {
		return cn.GetName() == tn.GetName()
	}
	return ch == tch
}

// sameRoom is a stub — returns true (assumes same room) until world integration.
func sameRoom(ch, tch interface{}) bool {
	return true
}

// sameRoomWithObj is a stub — returns true.
func sameRoomWithObj(ch, tobj interface{}) bool {
	return true
}

// sendToRoom sends act-formatted messages to the room.
// This is a stub that delegates to world for real room iteration.
func sendToRoom(format string, ch, tobj, tch interface{}, realName, obfuscated string, world interface{}) {
	// TODO: Real room iteration — iterate world[ch.in_room].people
	// For each person: if same class as caster, use realName; else use obfuscated
	// For now, just send to ch
	type sender interface{ SendMessage(string) }
	if s, ok := ch.(sender); ok {
		msg := strings.Replace(format, "%s", realName, 1)
		msg = strings.ReplaceAll(msg, "$n", "Someone")
		msg = strings.ReplaceAll(msg, "$N", "someone")
		msg = strings.ReplaceAll(msg, "$s", "their")
		msg = strings.ReplaceAll(msg, "$p", "something")
		s.SendMessage(msg)
	}
}

// sendAct is a minimal act() replacement.
func sendAct(format string, ch, obj, victim interface{}, world interface{}) {
	type sender interface{ SendMessage(string) }
	if s, ok := ch.(sender); ok {
		s.SendMessage(format)
	}
}
