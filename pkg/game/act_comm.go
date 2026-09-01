// act_comm.go — Ported from src/act.comm.c (Dark Pawns MUD)
//
// Communication commands: say, race-say, group-say, tell, reply, shout,
// whisper, ask, write, page, gossip, chat, auction, gratz, newbie, think,
// clan-tell, and language translation functions.
package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// Race constants (matching structs.h RACE_*)
// ---------------------------------------------------------------------------
const (
	raceDwarf      = 1
	raceElf        = 2
	raceGnoll      = 3
	raceDraconian  = 4
	raceGiantish   = 5
	raceUndead     = 6
	raceDrow       = 7
	raceRakshasa   = 8
	raceDeepDwarf  = 9
	raceSurfaceElf = 10
)

// ---------------------------------------------------------------------------
// Position constants — canonical source: pkg/combat/formulas.go
// ---------------------------------------------------------------------------
const (
	posDead            = combat.PosDead
	posMortallyWounded = combat.PosMortally
	posIncapacitated   = combat.PosIncap
	posStunned         = combat.PosStunned
	posSleeping        = combat.PosSleeping
	posResting         = combat.PosResting
	posSitting         = combat.PosSitting
	posFighting        = combat.PosFighting
	posStanding        = combat.PosStanding
)

// ---------------------------------------------------------------------------
// PLR flags (Player Flags) bit positions — same bits as structs.h PLR_*
// These go into p.Flags (uint64).
// ---------------------------------------------------------------------------
const (
	plrNoShout    uint64 = 1 << 0
	PLR_INVISIBLE uint64 = 1 << 1
	_                    = 1 << 2
	_                    = 1 << 3
	plrWriting    uint64 = 1 << 4
	plrOutlaw     uint64 = 1 << 5
)

// ---------------------------------------------------------------------------
// Level constants. lvlImmort is declared in spec_procs4.go (31).
// ---------------------------------------------------------------------------
const (
	lvlGod = 34 // used in item_transfer.go
)

// ---------------------------------------------------------------------------
// Misc constants
// ---------------------------------------------------------------------------
const (
	noBody         = -1
	levelCanShout  = 2 // C: level_can_shout = 2 (src/config.c)
	levelCanGossip = 5
	hollerMoveCost = 20
	maxNoteLength  = 1000
)

// ---------------------------------------------------------------------------
// Syllable substitution — character-by-character left-to-right scan
// matching the C algorithm. At each position, tries all syllable entries
// and replaces the longest matching prefix.
// ---------------------------------------------------------------------------

type syllable struct {
	org string
	new string
}

// applySyllableSubstitution ports the C speak_* loop (act.comm.c speak_drunk /
// speak_elven / speak_dwarven, all the same shape). Two quirks are load-bearing:
// the inner table scan has NO break — a match appends and advances the cursor,
// then the scan CONTINUES from the next entry at the NEW position, so an
// earlier entry can never match text revealed by a later match in the same
// pass ("the killer" renders "th' killer": after "the " matches, "kill" is
// already passed and 'k' falls to the identity entry) — and each pass appends
// at most ONE unmatched byte. Longest-match-per-position substitution is NOT
// equivalent (it would emit "th' murderizeer ish here"). Byte-oriented like
// the C strcat/strncat calls.
func applySyllableSubstitution(input string, syls []syllable) string {
	if input == "" {
		return ""
	}
	out := make([]byte, 0, len(input)*2)
	pos := 0
	for pos < len(input) {
		for _, s := range syls {
			if s.org == "" {
				continue
			}
			if strings.HasPrefix(input[pos:], s.org) {
				out = append(out, s.new...)
				pos += len(s.org)
			}
		}
		if pos < len(input) {
			out = append(out, input[pos])
			pos++
		}
	}
	return string(out)
}

// ---------------------------------------------------------------------------
// Syllable tables — ported verbatim from act.comm.c
// ---------------------------------------------------------------------------

var rakSyllables = []syllable{
	{" ", " "},
	{"are", "nec"},
	{"and", "arrl"},
	{"be", "fess"},
	{"how", "ciss"},
	{"what", "rriit"},
	{"is", "garr"},
	{"ou", "owwl"},
	{"where", "kaal"},
	{"me", "phis"},
	{"dwarf", "dwarf"},
	{"elf", "elf"},
	{"fucking", "fucking"},
	{"serapis", "Serapis"},
	{"Serapis", "Serapis"},
	{"kill", "llirr"},
	{"kender", "kenderkin"},
	{"centaur", "centaur"},
	{"rakshasa", "rakshasa"},
	{"Rakshasa", "Rakshasa"},
	{"human", "human"},
	{"elven", "elven"},
	{"dwarven", "dwarven"},
	{"god", "kashka"},
	{"God", "Kashka"},
	{"who", "rukkaturl"},
	{"ck", "k"},
	{"cks", "th"},
	{"the ", "(growl) "},
}

var dwarfSyllables = []syllable{
	{" ", " "},
	{"are", "icht"},
	{"and", "ent"},
	{"be", "ki"},
	{"how", "var"},
	{"what", "war"},
	{"is", "ict"},
	{"ou", "agen"},
	{"where", "hung"},
	{"me", "mein"},
	{"dwarf", "dwarf"},
	{"Dwarf", "Dwarf"},
	{"elf", "eli"},
	{"Elf", "Eli"},
	{"fucking", "fucking"},
	{"serapis", "Serapis"},
	{"Serapis", "Serapis"},
	{"kill", "k'ne"},
	{"kender", "kenderkin"},
	{"centaur", "centaur"},
	{"rakshasa", "rakshasa"},
	{"Rakshasa", "Rakshasa"},
	{"human", "human"},
	{"elven", "eli"},
	{"Elven", "Eli"},
	{"dwarven", "dwarven"},
	{"god", "g'du"},
	{"God", "G'du"},
	{"who", "b'ir"},
	{"ck", "k"},
	{"cks", "ks"},
	{"the ", "t'el "},
}

var elfSyllables = []syllable{
	{" ", " "},
	{"are", "est"},
	{"and", "et"},
	{"be", "deleste"},
	{"how", "quad"},
	{"what", "quod"},
	{"is", "est"},
	{"ou", "estra"},
	{"where", "este"},
	{"me", "ego"},
	{"dwarf", "dwarf"},
	{"elf", "elvinisti"},
	{"Elf", "Elvinisti"},
	{"fucking", "fucking"},
	{"serapis", "Serapis"},
	{"Serapis", "Serapis"},
	{"kill", "beligant"},
	{"kender", "kenderkin"},
	{"centaur", "centaur"},
	{"rakshasa", "rakshasa"},
	{"Rakshasa", "Rakshasa"},
	{"human", "human"},
	{"elven", "elvenesti"},
	{"Elven", "Elvenesti"},
	{"dwarven", "dwarven"},
	{"god", "deus"},
	{"God", "Deorum"},
	{"who", "quelsteno"},
	{"ck", "llin"},
	{"cks", "llins"},
	{"the ", "a "},
}

var gnollSyllables = []syllable{
	{" ", " "},
	{"are", "is"},
	{"and", "n"},
	{"be", "be"},
	{"how", "ow"},
	{"what", "wot"},
	{"is", "be"},
	{"ou", "a"},
	{"where", "wherr"},
	{"me", "me"},
	{"dwarf", "dwarf"},
	{"elf", "elf"},
	{"fucking", "fucking"},
	{"serapis", "Serapis"},
	{"Serapis", "Serapis"},
	{"kill", "k'll"},
	{"kender", "kender"},
	{"centaur", "centaur"},
	{"rakshasa", "rakshasa"},
	{"Rakshasa", "Rakshasa"},
	{"human", "human"},
	{"elven", "elven"},
	{"dwarven", "dwarven"},
	{"god", "gud"},
	{"God", "Gud"},
	{"who", "oo"},
	{"ck", "k"},
	{"cks", "ks"},
	{"the ", "da "},
	{"a", "a"},
	{"an", "an"},
	{"you", "yous"},
	{"they", "dem"},
	{"them", "dem"},
	{"i", "me"},
	{"my", "me"},
	{"your", "yer"},
	{"have", "as"},
	{"for", "fer"},
	{"of", "o"},
	{"to", "ta"},
	{"will", "wo"},
	{"can", "ken"},
	{"orc", "orc"},
	{"good", "gud"},
}

var draconianSyllables = []syllable{
	{" ", " "},
	{"are", "or"},
	{"and", "sz"},
	{"be", "be"},
	{"how", "ha"},
	{"what", "wat"},
	{"is", "xith"},
	{"ou", "x"},
	{"where", "wher"},
	{"me", "xi"},
	{"dwarf", "zex"},
	{"elf", "zel"},
	{"fucking", "fucking"},
	{"serapis", "Xith'xis"},
	{"Serapis", "Xith'xis"},
	{"kill", "k'xith"},
	{"kender", "kix'zel"},
	{"centaur", "zen'tor"},
	{"rakshasa", "xak'sa"},
	{"Rakshasa", "Xak'sa"},
	{"human", "xuman"},
	{"elven", "zelven"},
	{"dwarven", "zexen"},
	{"god", "zexon"},
	{"God", "Zexon"},
	{"who", "xi"},
	{"ck", "x"},
	{"cks", "xis"},
	{"the ", "zo "},
	{"a", "ha"},
	{"th", "zz"},
	{"you", "xiu"},
	{"e", "zek"},
	{"to", "kix"},
	{"or", "vyz"},
	{"dragon", "vur"},
	{"orc", "rex"},
	{"gnoll", "zev"},
}

var giantishSyllables = []syllable{
	{" ", " "},
	{"are", "arr"},
	{"and", "n"},
	{"be", "be"},
	{"how", "hoo"},
	{"what", "wot"},
	{"is", "iz"},
	{"ou", "oo"},
	{"where", "wur"},
	{"me", "me"},
	{"dwarf", "dwar"},
	{"elf", "elf"},
	{"fucking", "fookin"},
	{"serapis", "Serap"},
	{"Serapis", "Serap"},
	{"kill", "k'rush"},
	{"kender", "kender"},
	{"centaur", "sentaur"},
	{"rakshasa", "raksha"},
	{"Rakshasa", "Raksha"},
	{"human", "human"},
	{"elven", "elven"},
	{"dwarven", "dwarven"},
	{"god", "gor"},
	{"God", "Gor"},
	{"who", "oo"},
	{"ck", "k"},
	{"cks", "ks"},
	{"the ", "da "},
	{"a", "a"},
	{"to", "tuh"},
	{"you", "yoo"},
	{"with", "wiv"},
	{"giant", "gigant"},
	{"your", "yer"},
}

var deadspeakSyllables = []syllable{
	{" ", " "},
	{"a", "au"},
	{"e", "eu"},
	{"i", "ei"},
	{"o", "ou"},
	{"u", "uu"},
	{"the ", "theu "},
	{"is", "eis"},
	{"of", "eof"},
}

// ---------------------------------------------------------------------------
// Language translation functions
// ---------------------------------------------------------------------------

func speakRakshasan(said string) string { return applySyllableSubstitution(said, rakSyllables) }
func speakDwarven(said string) string   { return applySyllableSubstitution(said, dwarfSyllables) }
func speakElven(said string) string     { return applySyllableSubstitution(said, elfSyllables) }
func speakGnoll(said string) string     { return applySyllableSubstitution(said, gnollSyllables) }
func speakDraconian(said string) string { return applySyllableSubstitution(said, draconianSyllables) }

func speakGiantish(said string) string { return applySyllableSubstitution(said, giantishSyllables) }

func speakDeadspeak(said string) string { return applySyllableSubstitution(said, deadspeakSyllables) }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// skipSpaces trims leading spaces (matching skip_spaces from merc.h).
func skipSpaces(s string) string {
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	return s
}

// fillWord reports whether a word is one of C's fill[] words (interpreter.c:853)
// that one_argument/two_arguments skip when parsing command arguments.
func fillWord(word string) bool {
	switch strings.ToLower(word) {
	case "in", "from", "with", "the", "on", "at", "to":
		return true
	}
	return false
}

// oneArgument copies the first non-fill-word argument (lowercased) into the
// first return value and returns the remainder, mirroring C one_argument
// (interpreter.c:1265): leading fill words are skipped and the argument is
// case-folded. The remainder preserves its original case.
func oneArgument(input string) (string, string) {
	for {
		input = skipSpaces(input)
		if input == "" {
			return "", ""
		}
		fields := strings.Fields(input)
		word := fields[0]
		rest := skipSpaces(strings.TrimPrefix(input, word))
		if lower := strings.ToLower(word); !fillWord(lower) {
			return lower, rest
		}
		input = rest
	}
}

// OneArgument exposes C one_argument parsing to command packages. It returns
// the first non-fill-word token lowercased, matching interpreter.c:1265.
func OneArgument(input string) (string, string) {
	return oneArgument(input)
}

// oneWordArg copies the first non-fill-word token, accepting a double-quoted
// span as one token, and returns the remainder. This mirrors C one_word
// (interpreter.c:1291), which do_mold uses for the new object's name.
func oneWordArg(input string) (string, string) {
	for {
		input = skipSpaces(input)
		if input == "" {
			return "", ""
		}

		var word, rest string
		if input[0] == '"' {
			quoted := input[1:]
			if close := strings.IndexByte(quoted, '"'); close >= 0 {
				word = quoted[:close]
				rest = quoted[close+1:]
			} else {
				word = quoted
				rest = ""
			}
		} else if split := strings.IndexByte(input, ' '); split >= 0 {
			word = input[:split]
			rest = input[split:]
		} else {
			word = input
		}

		word = strings.ToLower(word)
		rest = skipSpaces(rest)
		if !fillWord(word) {
			return word, rest
		}
		input = rest
	}
}

// OneWord exposes C one_word parsing to command packages. It lowercases the
// returned token and preserves the remainder's original case.
func OneWord(input string) (string, string) {
	return oneWordArg(input)
}

// halfChop splits the first whitespace-delimited word from the rest, mirroring
// C half_chop (interpreter.c:1372). It calls any_one_arg, which lowercases the
// first token but does NOT skip fill words; the remainder keeps its original
// case. e.g. "Bob Hello THERE" -> "bob", "Hello THERE".
func halfChop(input string) (string, string) {
	input = skipSpaces(input)
	if input == "" {
		return "", ""
	}
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return "", ""
	}
	word := fields[0]
	rest := skipSpaces(strings.TrimPrefix(input, word))
	return strings.ToLower(word), rest
}

// AllPlayers returns a snapshot of all connected players.
func (w *World) AllPlayers() []*Player {
	w.mu.RLock()
	defer w.mu.RUnlock()
	players := make([]*Player, 0, len(w.players))
	for _, p := range w.players {
		players = append(players, p)
	}
	return players
}

// isNumber checks if a string parses as an integer.
func isNumber(s string) bool {
	if s == "" {
		return false
	}
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return err == nil
}

// checkStupid returns true if the character is too stupid to speak.
// (C: GET_WIS == 0 || GET_INT == 0)
func checkStupid(ch *Player) bool {
	return ch.GetWis() == 0 || ch.GetInt() == 0
}

// determineVerb returns the speaking verb based on the last char of msg.
func determineVerb(msg string) string {
	if msg == "" {
		return "says"
	}
	switch msg[len(msg)-1] {
	case '!':
		return "exclaims"
	case '?':
		return "asks"
	case '.':
		return "states"
	default:
		return "says"
	}
}

// ---------------------------------------------------------------------------
// 1–9. Language translation functions (defined above)
// 10.   speakDrunk (defined above)
//
// Now the ACMD functions:
// ---------------------------------------------------------------------------

// doRaceSay — port of do_race_say().
