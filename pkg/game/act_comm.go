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
// Condition indices — canonical source: pkg/game/limits.go
// ---------------------------------------------------------------------------
const (
	condDrunk = CondDrunk //nolint:unused // used in comm_say.go
)

// ---------------------------------------------------------------------------
// PLR flags (Player Flags) bit positions — same bits as structs.h PLR_*
// These go into p.Flags (uint64).
// ---------------------------------------------------------------------------
const (
	plrNoShout   uint64 = 1 << 0
	PLR_INVISIBLE uint64 = 1 << 1
	_                    = 1 << 2
	_                    = 1 << 3
	plrWriting    uint64 = 1 << 4
	plrOutlaw     uint64 = 1 << 5
)

// ---------------------------------------------------------------------------
// PRF flags (Preference flags) — use high bits of p.Flags since low bits
// are taken by PLR_*. The C code has these as separate bits in PRF_FLAGS.
// ---------------------------------------------------------------------------
const (
	prfNoTell   uint64 = 1 << 16
	prfNoShout  uint64 = 1 << 17
	prfNoGossip uint64 = 1 << 18
	prfNoAuct   uint64 = 1 << 19
	prfNoGratz  uint64 = 1 << 20
	prfNoNewbie uint64 = 1 << 21
	prfNoRepeat uint64 = 1 << 22
	prfDeaf     uint64 = 1 << 23
	prfAfk      uint64 = 1 << 24
	prfNoCtell  uint64 = 1 << 25
)

// ---------------------------------------------------------------------------
// Level constants. lvlImmort is declared in spec_procs4.go (31).
// ---------------------------------------------------------------------------
const (
	lvlGod = 34 //nolint:unused // used in item_consumable.go, item_transfer.go
)

// ---------------------------------------------------------------------------
// Misc constants
// ---------------------------------------------------------------------------
const (
	noBody           = -1
	levelCanShout    = 5
	levelCanGossip   = 5
	hollerMoveCost   = 10
	maxNoteLength    = 1000
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

func applySyllableSubstitution(input string, syls []syllable) string {
	if input == "" {
		return ""
	}
	runes := []rune(input)
	var out strings.Builder
	out.Grow(len(runes) * 2)
	pos := 0
	for pos < len(runes) {
		matched := false
		bestLen := 0
		bestNew := ""
		for _, s := range syls {
			sr := []rune(s.org)
			if len(sr) == 0 || pos+len(sr) > len(runes) {
				continue
			}
			match := true
			for i := 0; i < len(sr); i++ {
				if runes[pos+i] != sr[i] {
					match = false
					break
				}
			}
			if match && len(sr) > bestLen {
				bestLen = len(sr)
				bestNew = s.new
				matched = true
			}
		}
		if matched {
			out.WriteString(bestNew)
			pos += bestLen
		} else {
			out.WriteRune(runes[pos])
			pos++
		}
	}
	return out.String()
}

// ---------------------------------------------------------------------------
// Syllable tables — ported verbatim from act.comm.c
// ---------------------------------------------------------------------------

var rakSyllables = []syllable{
	{" ", " "}, {"are", "nec"}, {"and", "arrl"}, {"be", "fess"},
	{"how", "ciss"}, {"what", "rriit"}, {"is", "garr"}, {"ou", "owwl"},
	{"where", "kaal"}, {"me", "phis"}, {"dwarf", "dwarf"},
	{"elf", "elf"}, {"fucking", "fucking"},
	{"serapis", "Serapis"}, {"Serapis", "Serapis"},
	{"kill", "llirr"}, {"kender", "kenderkin"}, {"centaur", "centaur"},
	{"rakshasa", "rakshasa"}, {"Rakshasa", "Rakshasa"},
	{"human", "human"}, {"elven", "elven"}, {"dwarven", "dwarven"},
	{"god", "kashka"}, {"God", "Kashka"}, {"who", "rukkaturl"},
	{"ck", "k"}, {"cks", "th"}, {"the ", "(growl) "},
}

var dwarfSyllables = []syllable{
	{" ", " "}, {"are", "icht"}, {"and", "ent"}, {"be", "ki"},
	{"how", "var"}, {"what", "war"}, {"is", "ict"}, {"ou", "agen"},
	{"where", "hung"}, {"me", "mein"}, {"dwarf", "dwarf"}, {"Dwarf", "Dwarf"},
	{"elf", "eli"}, {"Elf", "Eli"}, {"fucking", "fucking"},
	{"serapis", "Serapis"}, {"Serapis", "Serapis"},
	{"kill", "k'ne"}, {"kender", "kenderkin"}, {"centaur", "centaur"},
	{"rakshasa", "rakshasa"}, {"Rakshasa", "Rakshasa"},
	{"human", "human"}, {"elven", "eli"}, {"Elven", "Eli"},
	{"dwarven", "dwarven"}, {"god", "g'du"}, {"God", "G'du"},
	{"who", "b'ir"}, {"ck", "k"}, {"cks", "ks"},
	{"the ", "t'el "},
}

var elfSyllables = []syllable{
	{" ", " "}, {"are", "est"}, {"and", "et"}, {"be", "deleste"},
	{"how", "quad"}, {"what", "quod"}, {"is", "est"}, {"ou", "estra"},
	{"where", "este"}, {"me", "ego"}, {"dwarf", "dwarf"},
	{"elf", "elvinisti"}, {"Elf", "Elvinisti"}, {"fucking", "fucking"},
	{"serapis", "Serapis"}, {"Serapis", "Serapis"},
	{"kill", "beligant"}, {"kender", "kenderkin"}, {"centaur", "centaur"},
	{"rakshasa", "rakshasa"}, {"Rakshasa", "Rakshasa"},
	{"human", "human"}, {"elven", "elvenesti"}, {"Elven", "Elvenesti"},
	{"dwarven", "dwarven"}, {"god", "deus"}, {"God", "Deorum"},
	{"who", "quelsteno"}, {"ck", "llin"}, {"cks", "llins"},
	{"the ", "a "},
}

var gnollSyllables = []syllable{
	{" ", " "}, {"are", "is"}, {"and", "n"}, {"be", "be"},
	{"how", "ow"}, {"what", "wot"}, {"is", "be"}, {"ou", "a"},
	{"where", "wherr"}, {"me", "me"}, {"dwarf", "dwarf"},
	{"elf", "elf"}, {"fucking", "fucking"},
	{"serapis", "Serapis"}, {"Serapis", "Serapis"},
	{"kill", "k'll"}, {"kender", "kender"}, {"centaur", "centaur"},
	{"rakshasa", "rakshasa"}, {"Rakshasa", "Rakshasa"},
	{"human", "human"}, {"elven", "elven"}, {"dwarven", "dwarven"},
	{"god", "gud"}, {"God", "Gud"}, {"who", "oo"},
	{"ck", "k"}, {"cks", "ks"}, {"the ", "da "},
	{"a", "a"}, {"an", "an"}, {"you", "yous"}, {"they", "dem"},
	{"them", "dem"}, {"i", "me"}, {"my", "me"}, {"your", "yer"},
	{"have", "as"}, {"for", "fer"}, {"of", "o"}, {"to", "ta"},
	{"will", "wo"}, {"can", "ken"}, {"orc", "orc"}, {"good", "gud"},
}

var draconianSyllables = []syllable{
	{" ", " "}, {"are", "or"}, {"and", "sz"}, {"be", "be"},
	{"how", "ha"}, {"what", "wat"}, {"is", "xith"}, {"ou", "x"},
	{"where", "wher"}, {"me", "xi"}, {"dwarf", "zex"},
	{"elf", "zel"}, {"fucking", "fucking"},
	{"serapis", "Xith'xis"}, {"Serapis", "Xith'xis"},
	{"kill", "k'xith"}, {"kender", "kix'zel"}, {"centaur", "zen'tor"},
	{"rakshasa", "xak'sa"}, {"Rakshasa", "Xak'sa"},
	{"human", "xuman"}, {"elven", "zelven"}, {"dwarven", "zexen"},
	{"god", "zexon"}, {"God", "Zexon"}, {"who", "xi"},
	{"ck", "x"}, {"cks", "xis"}, {"the ", "zo "},
	{"a", "ha"}, {"th", "zz"}, {"you", "xiu"}, {"e", "zek"},
	{"to", "kix"}, {"or", "vyz"}, {"dragon", "vur"}, {"orc", "rex"},
	{"gnoll", "zev"},
}

var giantishSyllables = []syllable{
	{" ", " "}, {"are", "arr"}, {"and", "n"}, {"be", "be"},
	{"how", "hoo"}, {"what", "wot"}, {"is", "iz"}, {"ou", "oo"},
	{"where", "wur"}, {"me", "me"}, {"dwarf", "dwar"},
	{"elf", "elf"}, {"fucking", "fookin"},
	{"serapis", "Serap"}, {"Serapis", "Serap"},
	{"kill", "k'rush"}, {"kender", "kender"}, {"centaur", "sentaur"},
	{"rakshasa", "raksha"}, {"Rakshasa", "Raksha"},
	{"human", "human"}, {"elven", "elven"}, {"dwarven", "dwarven"},
	{"god", "gor"}, {"God", "Gor"}, {"who", "oo"},
	{"ck", "k"}, {"cks", "ks"}, {"the ", "da "},
	{"a", "a"}, {"to", "tuh"}, {"you", "yoo"}, {"with", "wiv"},
	{"giant", "gigant"}, {"your", "yer"},
}

var deadspeakSyllables = []syllable{
	{" ", " "}, {"a", "au"}, {"e", "eu"}, {"i", "ei"}, {"o", "ou"},
	{"u", "uu"}, {"the ", "theu "}, {"is", "eis"}, {"of", "eof"},
}

var drunkSyllables = []syllable{ //nolint:unused // used by speakDrunk in comm_say.go
	{" ", " "}, {"are", "arsh"}, {"and", "andsh"}, {"how", "howsh"},
	{"what", "wha'"}, {"is", "ish"}, {"where", "whersh"},
	{"kill", "murderize"}, {"ck", "shkin"}, {"the ", "th' "},
}

// ---------------------------------------------------------------------------
// Language translation functions
// ---------------------------------------------------------------------------

func speakRakshasan(said string) string { return applySyllableSubstitution(said, rakSyllables) }
func speakDwarven(said string) string   { return applySyllableSubstitution(said, dwarfSyllables) }
func speakElven(said string) string     { return applySyllableSubstitution(said, elfSyllables) }
func speakGnoll(said string) string     { return applySyllableSubstitution(said, gnollSyllables) }
func speakDraconian(said string) string { return applySyllableSubstitution(said, draconianSyllables) }
func speakGiantish(said string) string  { return applySyllableSubstitution(said, giantishSyllables) }
func speakDeadspeak(said string) string { return applySyllableSubstitution(said, deadspeakSyllables) }
func speakDrunk(said string) string     { return applySyllableSubstitution(said, drunkSyllables) } //nolint:unused // used in comm_say.go

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

// oneArgument splits input into (firstWord, restAfterFirst).
func oneArgument(input string) (string, string) {
	input = skipSpaces(input)
	if input == "" {
		return "", ""
	}
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return "", ""
	}
	word := fields[0]
	rest := strings.TrimPrefix(input, word)
	rest = skipSpaces(rest)
	return word, rest
}

// halfChop splits first word from rest (mirrors C half_chop).
func halfChop(input string) (string, string) {
	return oneArgument(input)
}

// allPlayers returns a snapshot of all connected players.
func (w *World) allPlayers() []*Player {
	w.mu.RLock()
	defer w.mu.RUnlock()
	players := make([]*Player, 0, len(w.players))
	for _, p := range w.players {
		players = append(players, p)
	}
	return players
}

// getCharVis finds a player by name anywhere in the world (case-insensitive, prefix).
func (w *World) getCharVis(ch *Player, name string) *Player { //nolint:unused // used in comm_channel.go, comm_tell.go, graph.go
	for _, p := range w.allPlayers() {
		if !p.IsNPC() && (strings.EqualFold(p.Name, name) ||
			strings.HasPrefix(strings.ToLower(p.Name), strings.ToLower(name))) {
			return p
		}
	}
	return nil
}

// getCharRoomVis finds a player by name in the same room as ch.
func (w *World) getCharRoomVis(ch *Player, name string) *Player {
	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
		if p.Name == ch.Name {
			continue
		}
		if strings.EqualFold(p.Name, name) ||
			strings.HasPrefix(strings.ToLower(p.Name), strings.ToLower(name)) {
			return p
		}
	}
	return nil
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

// deleteAnsiControls strips ANSI escape sequences from a string.
func deleteAnsiControls(s string) string { //nolint:unused // used in comm_say.go, comm_tell.go
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1B' && i+1 < len(s) && s[i+1] == '[' {
			// Skip the escape sequence.
			for j := i; j < len(s); j++ {
				if s[j] >= 'A' && s[j] <= 'Z' || s[j] >= 'a' && s[j] <= 'z' {
					i = j
					break
				}
			}
			continue
		}
		buf.WriteByte(s[i])
	}
	return buf.String()
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
// Global last-teller tracking (replacing C GET_LAST_TELL, NOBODY sentinel)
// ---------------------------------------------------------------------------

// lastTellers is a map of tellerID -> lastTellRecipientID.
// Initialized lazily in getLastTellers/setLastTeller.
type lastTellersData struct { //nolint:unused // used in world.go
	store map[int]int
}

func (w *World) initLastTellers() { //nolint:unused // used in setLastTeller/getLastTeller
	if w.lastTellers == nil {
		w.lastTellers = &lastTellersData{store: make(map[int]int)}
	}
}
func (w *World) setLastTeller(chID, victID int) { //nolint:unused // used in comm_tell.go
	w.initLastTellers()
	w.lastTellers.store[chID] = victID
}

func (w *World) getLastTeller(chID int) int { //nolint:unused // used in comm_tell.go
	w.initLastTellers()
	if id, ok := w.lastTellers.store[chID]; ok {
		return id
	}
	return noBody
}

// ---------------------------------------------------------------------------
// 1–9. Language translation functions (defined above)
// 10.   speakDrunk (defined above)
//
// Now the ACMD functions:
// ---------------------------------------------------------------------------

// doRaceSay — port of do_race_say().
