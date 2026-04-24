// act_comm.go — Ported from src/act.comm.c (Dark Pawns MUD)
//
// Communication commands: say, race-say, group-say, tell, reply, shout,
// whisper, ask, write, page, gossip, chat, auction, gratz, newbie, think,
// clan-tell, and language translation functions.
package game

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Subcommand indices (matching act.comm.c subcmd enum, roughly)
// ---------------------------------------------------------------------------
const (
	subcmdHoller  = 0
	subcmdShout   = 1
	subcmdGossip  = 2
	subcmdAuction = 3
	subcmdGratz   = 4
	subcmdNewbie  = 5
	subcmdWhisper = 6
	subcmdAsk     = 7
	subcmdQSay    = 8
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
// Position constants
// ---------------------------------------------------------------------------
const (
	posDead            = 0
	posMortallyWounded = 1
	posIncapacitated   = 2
	posStunned         = 3
	posSleeping        = 4
	posResting         = 5
	posSitting         = 6
	posFighting        = 7
	posStanding        = 8
)

// ---------------------------------------------------------------------------
// Condition indices
// ---------------------------------------------------------------------------
const (
	condFull  = 0
	condThirst = 1
	condDrunk  = 2
)

// ---------------------------------------------------------------------------
// PLR flags (Player Flags) bit positions — same bits as structs.h PLR_*
// These go into p.Flags (uint64).
// ---------------------------------------------------------------------------
const (
	plrNoShout uint64 = 1 << 0
	_                 = 1 << 1
	_                 = 1 << 2
	_                 = 1 << 3
	plrWriting uint64 = 1 << 4
	plrOutlaw  uint64 = 1 << 5
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
	lvlGod         = 34
	lvlGreatGod    = 38
	lvlImplementor = 40
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

var undercommonSyllables = []syllable{
	{" ", " "}, {"the ", "de "}, {"with", "wit"}, {"this", "dis"},
	{"that", "dat"}, {"are", "be"}, {"is", "be"}, {"you", "du"},
	{"i", "me"}, {"my", "me"}, {"or", "nor"}, {"your", "de"},
	{"they", "dey"}, {"them", "dem"}, {"what", "wot"}, {"and", "an"},
}

var drunkSyllables = []syllable{
	{" ", " "}, {"are", "arsh"}, {"and", "andsh"}, {"how", "howsh"},
	{"what", "wha'"}, {"is", "ish"}, {"where", "whersh"},
	{"kill", "murderize"}, {"ck", "shkin"}, {"the ", "th' "},
}

// ---------------------------------------------------------------------------
// Language translation functions
// ---------------------------------------------------------------------------

func speakRakshasan(said string) string   { return applySyllableSubstitution(said, rakSyllables) }
func speakDwarven(said string) string     { return applySyllableSubstitution(said, dwarfSyllables) }
func speakElven(said string) string       { return applySyllableSubstitution(said, elfSyllables) }
func speakGnoll(said string) string       { return applySyllableSubstitution(said, gnollSyllables) }
func speakDraconian(said string) string   { return applySyllableSubstitution(said, draconianSyllables) }
func speakGiantish(said string) string    { return applySyllableSubstitution(said, giantishSyllables) }
func speakDeadspeak(said string) string   { return applySyllableSubstitution(said, deadspeakSyllables) }
func speakUndercommon(said string) string { return applySyllableSubstitution(said, undercommonSyllables) }
func speakDrunk(said string) string       { return applySyllableSubstitution(said, drunkSyllables) }

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

// twoArguments splits two words and the remainder.
func twoArguments(input string) (string, string, string) {
	a, r := oneArgument(input)
	b, rest := oneArgument(r)
	return a, b, rest
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
func (w *World) getCharVis(ch *Player, name string) *Player {
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
func deleteAnsiControls(s string) string {
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
type lastTellersData struct {
	store map[int]int
}

func (w *World) getLastTellers() *lastTellersData {
	// World must have this accessible. We store it as a field on World
	// via initLastTellers.
	return w.lastTellers
}

// initLastTellers ensures the map is initialized. Called from acmd registration.
func (w *World) initLastTellers() {
	if w.lastTellers == nil {
		w.lastTellers = &lastTellersData{store: make(map[int]int)}
	}
}

func (w *World) setLastTeller(chID, victID int) {
	w.initLastTellers()
	w.lastTellers.store[chID] = victID
}

func (w *World) getLastTeller(chID int) int {
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
func (w *World) doRaceSay(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)

	if checkStupid(ch) {
		sendToChar(ch, "You are too stupid to communicate with language!\r\n")
		return true
	}
	if ch.Flags&plrNoShout != 0 {
		sendToChar(ch, "You cannot race-say!\r\n")
		return true
	}
	if arg == "" {
		sendToChar(ch, "Yes, but WHAT do you want to say?\r\n")
		return true
	}

	var translate func(string) string
	var raceName string

	switch ch.Race {
	case raceDwarf, raceDeepDwarf:
		translate = speakDwarven
		raceName = "Dwarven"
	case raceElf, raceSurfaceElf:
		translate = speakElven
		raceName = "Elven"
	case raceGnoll:
		translate = speakGnoll
		raceName = "Gnoll"
	case raceDraconian:
		translate = speakDraconian
		raceName = "Draconian"
	case raceGiantish:
		translate = speakGiantish
		raceName = "Giantish"
	case raceUndead:
		translate = speakDeadspeak
		raceName = "Deadspeak"
	case raceDrow, raceRakshasa:
		translate = speakRakshasan
		raceName = "Rakshasan"
	default:
		return true
	}

	translated := translate(arg)
	verb := determineVerb(arg)

	// Send to others in the room.
	verbMsg := fmt.Sprintf(" %s, ", verb)
	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
		if p.Name == ch.Name {
			continue
		}

		// Same race / immortals hear the original with race tag.
		// Other races hear the translated version.
		if p.Race == ch.Race || p.GetLevel() >= lvlImmort || p.IsNPC() {
			p.SendMessage(fmt.Sprintf("%s%s'(In %s) %s'\r\n", ch.Name, verbMsg, raceName, arg))
		} else {
			p.SendMessage(fmt.Sprintf("%s%s'%s'\r\n", ch.Name, verbMsg, translated))
		}
	}

	// Self-message.
	if ch.Flags&prfNoRepeat == 0 {
		ch.SendMessage(fmt.Sprintf("You%s'(In %s) %s'\r\n", verbMsg, raceName, arg))
	} else {
		sendToChar(ch, "Ok.\r\n")
	}

	return true
}

// doSay — port of do_say().
func (w *World) doSay(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)

	if checkStupid(ch) {
		sendToChar(ch, "You are too stupid to communicate with language!\r\n")
		return true
	}
	if ch.Flags&plrNoShout != 0 {
		sendToChar(ch, "You cannot speak!\r\n")
		return true
	}
	if arg == "" {
		sendToChar(ch, "Yes, but WHAT do you want to say?\r\n")
		return true
	}

	// Drunk substitution.
	speech := arg
	if ch.Conditions[condDrunk] > 10 {
		speech = speakDrunk(arg)
	}

	verb := determineVerb(arg)

	msg := fmt.Sprintf("$n %s, '%s'", verb, speech)
	msg = deleteAnsiControls(msg)
	w.roomMessage(ch.RoomVNum, msg)

	if ch.Flags&prfNoRepeat == 0 {
		selfMsg := fmt.Sprintf("You %s, '%s'\r\n", verb, arg)
		selfMsg = deleteAnsiControls(selfMsg)
		sendToChar(ch, selfMsg)
	} else {
		sendToChar(ch, "Ok.\r\n")
	}

	return true
}

// doGSay — port of do_gsay().
func (w *World) doGSay(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)

	if !ch.InGroup {
		sendToChar(ch, "But you are not a member of any group!\r\n")
		return true
	}
	if arg == "" {
		sendToChar(ch, "Yes, but WHAT do you want to group-say?\r\n")
		return true
	}

	msg := fmt.Sprintf("$n tells the group, '%s'", arg)
	msg = deleteAnsiControls(msg)

	// Find group leader.
	var leader *Player
	if ch.Following != "" {
		if l, ok := w.GetPlayer(ch.Following); ok {
			leader = l
		}
	}
	if leader == nil {
		leader = ch
	}

	// Broadcast to group members in the room.
	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
		if p.Name == ch.Name || !p.InGroup {
			continue
		}
		if p.Following == leader.Name || p.Name == leader.Name || ch.Following == p.Name {
			p.SendMessage(fmt.Sprintf("\x1B[1;37m%s\033[0m\r\n", msg))
		}
	}

	if ch.Flags&prfNoRepeat == 0 {
		selfMsg := fmt.Sprintf("You tell the group, '%s'\r\n", arg)
		selfMsg = deleteAnsiControls(selfMsg)
		ch.SendMessage(fmt.Sprintf("\x1B[1;37m%s\033[0m\r\n", selfMsg))
	} else {
		sendToChar(ch, "Ok.\r\n")
	}

	return true
}

// performTell — port of perform_tell().
func (w *World) performTell(ch *Player, vict *Player, arg string) {
	msg := fmt.Sprintf("$n tells you, '%s'", arg)
	msg = deleteAnsiControls(msg)
	vict.SendMessage(fmt.Sprintf("\033[0;31m%s\033[0m\r\n", msg))

	// AFK notice.
	if vict.Flags&prfAfk != 0 {
		ch.SendMessage(fmt.Sprintf("%s is AFK right now, %s may not hear you.\r\n",
			vict.Name, hisHer(vict.Sex)))
	}

	// Echo to sender.
	if ch.Flags&prfNoRepeat == 0 {
		echo := fmt.Sprintf("You tell $N, '%s'", arg)
		echo = deleteAnsiControls(echo)
		ch.SendMessage(fmt.Sprintf("\033[0;31m%s\033[0m\r\n", echo))
	} else {
		sendToChar(ch, "Ok.\r\n")
	}

	// Track for reply.
	w.setLastTeller(vict.ID, ch.ID)
}

// doTell — port of do_tell().
func (w *World) doTell(ch *Player, me *MobInstance, cmd string, arg string) bool {
	target, msg := oneArgument(arg)
	if target == "" || msg == "" {
		sendToChar(ch, "Who do you want to tell what??\r\n")
		return true
	}

	vict := w.getCharVis(ch, target)
	if vict == nil {
		sendToChar(ch, "No one by that name is playing.\r\n")
		return true
	}
	if vict.Name == ch.Name {
		sendToChar(ch, "You try to tell yourself something.\r\n")
		return true
	}
	if ch.Flags&prfNoTell != 0 && ch.GetLevel() < lvlImmort {
		sendToChar(ch, "You can't tell other people while you have notell on.\r\n")
		return true
	}
	if ch.Flags&plrNoShout != 0 {
		sendToChar(ch, "You cannot tell anyone anything!\r\n")
		return true
	}
	if w.roomHasFlag(ch.RoomVNum, "soundproof") {
		sendToChar(ch, "The walls seem to absorb your words.\r\n")
		return true
	}
	if !vict.IsNPC() && vict.Flags&plrWriting != 0 {
		ch.SendMessage(fmt.Sprintf("%s's writing a message right now; try again later.\r\n", vict.Name))
		return true
	}

	victNotellOrSP := (vict.Flags&prfNoTell != 0 || w.roomHasFlag(vict.RoomVNum, "soundproof"))
	if victNotellOrSP && ch.GetLevel() < lvlImmort {
		ch.SendMessage(fmt.Sprintf("%s can't hear you.\r\n", vict.Name))
		return true
	}

	w.performTell(ch, vict, msg)
	return true
}

// doReply — port of do_reply().
func (w *World) doReply(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)

	lastID := w.getLastTeller(ch.ID)
	if lastID == noBody {
		sendToChar(ch, "You have no one to reply to!\r\n")
		return true
	}
	if arg == "" {
		sendToChar(ch, "What is your reply?\r\n")
		return true
	}

	// Find the last teller by ID.
	var tch *Player
	for _, p := range w.allPlayers() {
		if p.ID == lastID {
			tch = p
			break
		}
	}
	if tch == nil {
		sendToChar(ch, "They are no longer playing.\r\n")
		return true
	}

	if ch.Flags&plrNoShout != 0 {
		sendToChar(ch, "You cannot tell anyone anything!\r\n")
		return true
	}
	if !tch.IsNPC() && tch.Flags&plrWriting != 0 {
		sendToChar(ch, "They are writing now, try later.\r\n")
		return true
	}

	tchNotellOrSP := (tch.Flags&prfNoTell != 0 || w.roomHasFlag(tch.RoomVNum, "soundproof"))
	if tchNotellOrSP {
		sendToChar(ch, "They can't hear you.\r\n")
		return true
	}
	if w.roomHasFlag(ch.RoomVNum, "soundproof") {
		sendToChar(ch, "The walls seem to absorb your words.\r\n")
		return true
	}

	w.performTell(ch, tch, arg)
	return true
}

// doSpecComm — port of do_spec_comm() (shout, whisper, ask).
func (w *World) doSpecComm(ch *Player, me *MobInstance, cmd string, arg string) bool {
	switch strings.ToLower(cmd) {
	case "shout":
		return w.doShout(ch, me, arg)
	case "whisper":
		return w.doWhisper(ch, me, arg)
	case "ask":
		return w.doAsk(ch, me, arg)
	}
	return true
}

// doShout — shout implementation.
func (w *World) doShout(ch *Player, me *MobInstance, arg string) bool {
	arg = skipSpaces(arg)

	if arg == "" {
		sendToChar(ch, "Shout what?\r\n")
		return true
	}
	if ch.GetLevel() < levelCanShout {
		sendToChar(ch, "You must be at least level 5 to shout.\r\n")
		return true
	}
	if ch.Flags&prfNoShout != 0 {
		sendToChar(ch, "You can't shout.\r\n")
		return true
	}
	if w.roomHasFlag(ch.RoomVNum, "soundproof") {
		sendToChar(ch, "The walls seem to absorb your words.\r\n")
		return true
	}

	msg := fmt.Sprintf("%s shouts, '%s'\r\n", ch.Name, arg)
	for _, p := range w.allPlayers() {
		if p.IsNPC() || p.Name == ch.Name {
			continue
		}
		if p.Flags&prfDeaf != 0 {
			continue
		}
		if p.Flags&prfNoShout != 0 {
			continue
		}
		if w.roomHasFlag(p.RoomVNum, "soundproof") {
			continue
		}
		p.SendMessage(msg)
	}

	sendToChar(ch, fmt.Sprintf("You shout, '%s'\r\n", arg))
	return true
}

// doWhisper — whisper implementation.
func (w *World) doWhisper(ch *Player, me *MobInstance, arg string) bool {
	target, msg := oneArgument(arg)
	if target == "" || msg == "" {
		sendToChar(ch, "Whisper whom what?\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, target)
	if vict == nil {
		sendToChar(ch, "No one by that name is here.\r\n")
		return true
	}

	vict.SendMessage(fmt.Sprintf("\x1B[1;33m%s whispers, '%s'\033[0m\r\n", ch.Name, msg))
	ch.SendMessage(fmt.Sprintf("You whisper to %s, '%s'\r\n", vict.Name, msg))

		// Broadcast to rest of room that whisper occurred.
	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
		if p.Name != ch.Name && p.Name != vict.Name {
			p.SendMessage(fmt.Sprintf("%s whispers something to %s.\r\n", ch.Name, vict.Name))
		}
	}

	return true
}

// doAsk — ask implementation (identical to whisper but broadcasts as ask).
func (w *World) doAsk(ch *Player, me *MobInstance, arg string) bool {
	target, msg := oneArgument(arg)
	if target == "" || msg == "" {
		sendToChar(ch, "Ask whom what?\r\n")
		return true
	}

	vict := w.getCharRoomVis(ch, target)
	if vict == nil {
		sendToChar(ch, "No one by that name is here.\r\n")
		return true
	}

	vict.SendMessage(fmt.Sprintf("\x1B[1;36m%s asks, '%s'\033[0m\r\n", ch.Name, msg))
	ch.SendMessage(fmt.Sprintf("You ask %s, '%s'\r\n", vict.Name, msg))

	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
		if p.Name != ch.Name && p.Name != vict.Name {
			p.SendMessage(fmt.Sprintf("%s asks %s something.\r\n", ch.Name, vict.Name))
		}
	}

	return true
}

// doWrite — port of do_write().
func (w *World) doWrite(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)

	if arg == "" {
		sendToChar(ch, "Write on what?\r\n")
		return true
	}

	// Find a writing surface (tablet, paper, etc.) in inventory or room.
	// Simplified: NPCs check obj list, players check inventory.
	// For now, just say they start writing.
	args := strings.Fields(arg)
	if len(args) == 0 {
		sendToChar(ch, "Write what?\r\n")
		return true
	}
	objName := args[0]
	_ = objName
	sendToChar(ch, "You start writing.\r\n")
	return true
}

// doPage -- port of do_page().
func (w *World) doPage(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		sendToChar(ch, "Page whom?\r\n")
		return true
	}

	// Format: target msg or multiple targets "target1 target2 msg"
	// Simplified: single target
	target, msg := halfChop(arg)
	if target == "" {
		sendToChar(ch, "Page whom?\r\n")
		return true
	}

	tch := w.getCharVis(ch, target)
	if tch == nil {
		sendToChar(ch, "No one by that name is playing.\r\n")
		return true
	}

	if msg == "" {
		msg = fmt.Sprintf("%s pages you!\r\n", ch.Name)
	} else {
		tch.SendMessage(fmt.Sprintf("\r\n%s pages: '%s'\r\n", ch.Name, msg))
	}

	sendToChar(ch, fmt.Sprintf("You page %s with '%s'\r\n", tch.Name, msg))
	return true
}

// doGenComm -- port of do_gen_comm() (gossip, chat, auction, gratz, newbie).
func (w *World) doGenComm(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		// Determine channel name from cmd / subcmd
		switch strings.ToLower(cmd) {
		case "gossip":
			sendToChar(ch, "Gossip what?\r\n")
		case "auction":
			sendToChar(ch, "Auction what?\r\n")
		case "gratz":
			sendToChar(ch, "Gratz whom?\r\n")
		case "newbie":
			sendToChar(ch, "Newbie what?\r\n")
		default:
			sendToChar(ch, "Say what?\r\n")
		}
		return true
	}

	// Build channel header
	var header string
	var flag uint64
	var minLevel int
	var channelName string

	switch strings.ToLower(cmd) {
	case "gossip":
		header = fmt.Sprintf("%s gossips, '%s'\r\n", ch.Name, arg)
		flag = prfNoGossip
		minLevel = levelCanGossip
		channelName = "gossip"
	case "auction":
		header = fmt.Sprintf("%s auctions, '%s'\r\n", ch.Name, arg)
		flag = prfNoAuct
		channelName = "auction"
	case "gratz":
		header = fmt.Sprintf("%s congratulates, '%s'\r\n", ch.Name, arg)
		flag = prfNoGratz
		channelName = "gratz"
	case "newbie":
		header = fmt.Sprintf("%s says, '%s'\r\n", ch.Name, arg)
		flag = prfNoNewbie
		channelName = "newbie"
	default:
		sendToChar(ch, "Unknown channel.\r\n")
		return true
	}

	if ch.GetLevel() < minLevel {
		sendToChar(ch, fmt.Sprintf("You need to be level %d to use that channel.\r\n", minLevel))
		return true
	}

	for _, p := range w.allPlayers() {
		if p.IsNPC() || p.Name == ch.Name {
			continue
		}
		if p.Flags&prfDeaf != 0 {
			continue
		}
		if p.Flags&flag != 0 {
			continue
		}
		p.SendMessage(header)
	}

	sendToChar(ch, fmt.Sprintf("You %s, '%s'\r\n", channelName, arg))
	return true
}

// doQcomm -- port of do_qcomm() (team/quiz communication).
func (w *World) doQcomm(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		sendToChar(ch, "What do you want to say?\r\n")
		return true
	}

	msg := fmt.Sprintf("%s says, '%s'\r\n", ch.Name, arg)
	for _, p := range w.GetPlayersInRoom(ch.RoomVNum) {
		if p.Name != ch.Name {
			p.SendMessage(msg)
		}
	}
	sendToChar(ch, fmt.Sprintf("You say, '%s'\r\n", arg))
	return true
}

// doThink -- port of do_think().
func (w *World) doThink(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		sendToChar(ch, "What do you want to think?\r\n")
		return true
	}

	sendToChar(ch, fmt.Sprintf("You think: '%s'\r\n", arg))
	return true
}

// doCTell -- port of do_ctell() (clan tell).
func (w *World) doCTell(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = skipSpaces(arg)
	if arg == "" {
		sendToChar(ch, "What do you want to tell your clan?\r\n")
		return true
	}

	// Clan system not yet implemented -- broadcast to all players as a fallback.
	msg := fmt.Sprintf("[Clan] %s tells the clan, '%s'\r\n", ch.Name, arg)
	for _, p := range w.allPlayers() {
		if p.Name == ch.Name {
			continue
		}
		if p.Flags&prfDeaf != 0 || p.Flags&prfNoCtell != 0 {
			continue
		}
		p.SendMessage(msg)
	}

	sendToChar(ch, fmt.Sprintf("You tell your clan, '%s'\r\n", arg))
	return true
}
