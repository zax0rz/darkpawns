package session

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// This file ports CircleMUD 3.0's medit.c (the OLC mobile editor) behind the
// CON_MEDIT-equivalent descriptor state. The architecture follows
// pkg/session/redit.go: the descriptor owns a working mob copy, later input
// lines route through its menu state, save/abort/disconnect mirror
// cleanup_olc, and the audience transitions ($n starts/stops using OLC) match
// C. All player-facing bytes are copied verbatim from src/medit.c (R1),
// including the numerical-input gate quirk and the script-menu's live
// shallow-script authority.

// These values mirror the MEDIT_* constants in src/olc.h. Keeping the state
// names local makes the descriptor transition table auditable without
// pretending that the other OLC families share this implementation.
type meditMode uint8

const (
	meditMainMenu meditMode = iota
	meditAlias
	meditSDesc
	meditLDesc
	meditDDesc
	meditNPCFlags
	meditAffFlags
	meditConfirmSaveString
	meditNoise
	meditScriptMenu
	meditNumericalResponse
	meditSex
	meditHitroll
	meditDamroll
	meditNDD
	meditSDD
	meditNumHPDice
	meditSizeHPDice
	meditAddHP
	meditAC
	meditExp
	meditGold
	meditPos
	meditDefaultPos
	meditAttack
	meditLevel
	meditAlignment
	meditRace
	meditScriptName
	meditScriptFlags
)

type meditState struct {
	mob        parser.Mob
	number     int
	zoneNumber int
	isNew      bool
	mode       meditMode
	olcVal     int // C OLC_VAL: the has-changed/quit-prompt flag.
}

// meditSaveMu serializes the zone mob-file save list. It mirrors C's
// olc_save_list for OLC_SAVE_MOB entries.
var (
	meditSaveMu   sync.Mutex
	meditSaveMobs = make(map[int]bool)
)

// The display tables below are byte-faithful copies of the C constant arrays
// the medit menus print (src/constants.c, src/fight.c). They intentionally
// differ from the parser's storage-name lists (pkg/parser/mob.go), which use
// normalized spellings ("AGGRESSIVE" vs C's "AGGR"); the bit positions are
// identical, only the menu spellings differ.

var meditGenderNames = []string{"Neutral", "Male", "Female"}

var meditPositionNames = []string{
	"Dead", "Mortally wounded", "Incapacitated", "Stunned", "Sleeping",
	"Resting", "Sitting", "Fighting", "Standing",
}

var meditAttackNames = []string{
	"hit", "sting", "whip", "slash", "bite", "bludgeon", "crush", "pound",
	"claw", "maul", "thrash", "pierce", "blast", "punch", "stab",
}

// C action_bits[] (src/constants.c); NUM_MOB_FLAGS is 25.
var meditMobFlagNames = []string{
	"SPEC", "SENTINEL", "SCAVENGER", "ISNPC", "AWARE", "AGGR", "STAY-ZONE",
	"WIMPY", "AGGR_EVIL", "AGGR_GOOD", "AGGR_NEUTRAL", "MEMORY", "HELPER",
	"!CHARM", "!SUMMN", "!SLEEP", "!BASH", "!BLIND", "HUNTER", "AGGR24",
	"RNDLD_ZONE", "MOUNTABLE", "RARE", "LOOTS", "OKGIVE",
}

// C affected_bits[] (src/constants.c); NUM_AFF_FLAGS is 37.
var meditAffFlagNames = []string{
	"BLIND", "INVIS", "DET-ALIGN", "DET-INVIS", "DET-MAGIC", "SENSE-LIFE",
	"WATERWALK", "SANCT", "GROUP", "CURSE", "INFRA", "POISON", "PROT-EVIL",
	"PROT-GOOD", "SLEEP", "!TRACK", "FLESH-ALTER", "DODGE", "SNEAK", "HIDE",
	"BERSERK", "CHARM", "FOLLOW", "WIMPY", "KUJI-KIRI", "CUTTHROAT", "FLY",
	"WEREWOLF", "VAMPIRE", "MOUNTED", "INVULN", "FLAMING", "NOTHING", "HASTE",
	"SLOW", "DREAM", "WATERBREATHE",
}

// C mscript_bits[] (src/constants.c); NUM_MSCRIPT_FLAGS is 10.
var meditScriptFlagNames = []string{
	"NONE", "BRIBE", "GREET", "ONGIVE", "SOUND", "DEATH", "ONPULSE (ALL)",
	"ONPULSE (PC)", "FIGHT", "ONCMD",
}

// C mob_races[] (src/constants.c); NUM_MOB_RACES is 31, RACE_OTHER is 16.
var meditRaceNames = []string{
	"Human", "Elf", "Dwarf", "Kender", "Centaur", "Rakshasa", "Troll",
	"Lycanthrope", "Vampire", "Undead", "Dragon", "Demon", "Horse", "Reptile",
	"Arachnid", "Rodent", "Other", "Vegetable", "Giant", "Demi-god", "Ogre",
	"Insect", "Mammal", "Fish", "Avian", "Magical Construct", "Amphibian",
	"Humanoid", "Faery", "Ssaur", "Minotaur",
}

// The storage-name lists mirror pkg/parser/mob.go's private bit tables so the
// editor can round-trip flag names through bit indices. Bit positions match
// src/structs.h; only the spellings differ from the C menu tables above.

var meditParserActionBits = []string{
	"SPEC", "SENTINEL", "SCAVENGER", "ISNPC", "AWARE", "AGGRESSIVE",
	"STAY_ZONE", "WIMPY", "AGGR_EVIL", "AGGR_GOOD", "AGGR_NEUTRAL", "MEMORY",
	"HELPER", "NOCHARM", "NOSUMMON", "NOSLEEP", "NOBASH", "NOBLIND", "HUNTER",
	"AGGR24", "RANDZON", "MOUNTABLE", "RARE", "LOOTS", "OKGIVE", "EXTRACT",
}

var meditParserAffectBits = []string{
	"BLIND", "INVISIBLE", "DETECT_ALIGN", "DETECT_INVIS", "DETECT_MAGIC",
	"SENSE_LIFE", "WATERWALK", "SANCTUARY", "GROUP", "CURSE", "INFRAVISION",
	"POISON", "PROTECT_EVIL", "PROTECT_GOOD", "SLEEP", "NOTRACK",
	"FLESH_ALTER", "DODGE", "SNEAK", "HIDE", "BERSERK", "CHARM", "FOLLOW",
	"WIMPY", "KUJI_KIRI", "CUTTHROAT", "FLY", "WEREWOLF", "VAMPIRE", "MOUNT",
	"INVULN", "FLAMING", "NOTHING", "HASTE", "SLOW", "DREAM", "WATERBREATHE",
	"METALSKIN", "ROBBED",
}

// meditExpLookup is the C EXP_LOOKUP table from medit.c, indexed by level.
// The C array has LEVEL_IMPL+1 (41) entries; levels above 40 read out of
// bounds in C (undefined). The Go port clamps the lookup instead of
// reproducing the overread; see parseMeditLevelLocked.
var meditExpLookup = []int{
	25,
	100, 200, 350, 600, 900,
	1500, 2500, 3500, 4500, 6000,
	7500, 9000, 10500, 12000, 14000,
	16000, 18000, 20000, 22500, 25000,
	27500, 30000, 32500, 35000, 37500,
	40000, 45000, 50000, 55000, 60000,
	90000, 120000, 180000, 270000, 360000,
	363000, 369000, 372000, 375000, 400000,
}

// cmdMedit is the Go port of do_olc's SCMD_OLC_MEDIT branch (src/olc.c).
func cmdMedit(s *Session, args []string) error {
	if s.player == nil || s.manager == nil || s.manager.world == nil {
		return fmt.Errorf("not logged in")
	}

	var buf1, buf2 string
	if len(args) > 0 {
		buf1 = args[0]
	}
	if len(args) > 1 {
		buf2 = args[1]
	}

	// C: save = !strncmp(buf1, "save", 4). The match is case-sensitive in C,
	// so unlike the room editor the verb is not normalized here.
	save := strings.HasPrefix(buf1, "save")

	if buf1 == "" {
		s.meditSend("Specify a mobile VNUM to edit.\r\n")
		return nil
	}

	var number int
	if !isASCIIDigit(firstByte(buf1)) {
		if save {
			if buf2 == "" {
				s.meditSend("Save which zone?\r\n")
				return nil
			}
			number = atoiC(buf2) * 100
		} else {
			s.meditSend("Yikes!  Stop that, someone will get hurt!\r\n")
			return nil
		}
	} else {
		number = atoiC(buf1)
	}

	if other := olcDuplicateName(s.manager, number, func(c *Session) (int, bool) {
		if c.mobEdit == nil {
			return 0, false
		}
		return c.mobEdit.number, true
	}); other != "" {
		s.meditSend(fmt.Sprintf("That mobile is currently being edited by %s.\r\n", other))
		return nil
	}

	zone, ok := olcZoneForVNum(s.manager.world, number)
	if !ok {
		s.meditSend("Sorry, there is no zone for that number!\r\n")
		return nil
	}
	if !olcAuthorized(s, zone.Number) {
		s.meditSend("You do not have permission to edit this zone.\r\n")
		return nil
	}

	if save {
		s.meditSend("Saving all mobiles in zone.\r\n")
		slog.Info("OLC: medit zone save",
			"player", s.playerName, "zone", zone.Number)
		if err := saveMeditZone(s.manager.world, zone); err != nil {
			slog.Error("medit disk save failed",
				"player", s.playerName, "zone", zone.Number, "error", err)
		}
		return nil
	}

	return s.startMedit(number, zone)
}

// startMedit begins a medit session for vnum, mirroring do_olc's
// SCMD_OLC_MEDIT branch plus medit_setup_existing / medit_setup_new.
func (s *Session) startMedit(number int, zone *parser.Zone) error {
	mob, exists := s.manager.world.SnapshotMob(number)
	if !exists {
		mob = newMeditMob(number)
	}

	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	if s.mobEdit != nil {
		s.finishMeditLocked(cleanupMeditAll)
	}

	s.mobEdit = &meditState{
		mob:        mob,
		number:     number,
		zoneNumber: zone.Number,
		isNew:      !exists,
		mode:       meditMainMenu,
	}
	s.setPlayerWritingLocked(true)
	s.meditShowMenuLocked()
	if s.player != nil {
		s.meditActLocked(game.ToRoom, "$n starts using OLC.")
	}
	return nil
}

// newMeditMob builds the working copy for a mobile that has no prototype,
// mirroring medit_setup_new (src/medit.c) layered on init_mobile: the
// unfinished-mob strings, RACE_OTHER (16), the ISNPC bit, 1d1 hit and damage
// dice, and clear_char's standing position/default position plus AC 100.
func newMeditMob(number int) parser.Mob {
	return parser.Mob{
		VNum:         number,
		Keywords:     "mob unfinished",
		ShortDesc:    "the unfinished mob",
		LongDesc:     "An unfinished mob stands here.\r\n",
		DetailedDesc: "It looks, err, unfinished.\r\n",
		Race:         16,
		Position:     8,
		DefaultPos:   8,
		AC:           100,
		HP:           parser.DiceRoll{Num: 1, Sides: 1, Plus: 0},
		Damage:       parser.DiceRoll{Num: 1, Sides: 1, Plus: 0},
		Weight:       200,
		Height:       198,
		ActionFlags:  []string{"ISNPC"},
	}
}

// finishMeditLocked mirrors cleanup_olc for CON_MEDIT (src/olc.c): it clears
// the writing flag, broadcasts the stop-editing message, and returns the
// descriptor to playing. CLEANUP_ALL (C: "everything") is the only cleanup
// mode the mobile editor uses.
func (s *Session) finishMeditLocked(mode cleanupMeditMode) {
	if s.mobEdit == nil {
		return
	}
	switch mode {
	case cleanupMeditAll:
		s.setPlayerWritingLocked(false)
		if s.player != nil {
			s.meditActLocked(game.ToRoom, "$n stops using OLC.")
		}
		s.mobEdit = nil
	}
}

// cleanupMeditMode mirrors cleanup_olc's cleanup_type parameter. medit only
// ever uses CLEANUP_ALL.
type cleanupMeditMode int

const (
	cleanupMeditAll cleanupMeditMode = iota
)

// meditSend queues raw output while a medit session is active, mirroring
// write_to_output inside the C editor.
func (s *Session) meditSend(text string) {
	if err := s.SendMessage(text); err != nil {
		slog.Error("medit output failed", "player", s.playerName, "error", err)
	}
}

func (s *Session) meditSendLocked(text string) {
	if err := s.SendMessage(text); err != nil {
		slog.Error("medit output failed", "player", s.playerName, "error", err)
	}
}

// setPlayerWritingLocked toggles the PLR_WRITING flag, mirroring C's
// SET_BIT_AR(PLR_FLAGS, PLR_WRITING) on OLC entry and its removal in
// cleanup_olc. The caller holds textEditMu.
func (s *Session) setPlayerWritingLocked(writing bool) {
	if s.player != nil {
		s.player.SetPlrFlag(game.PlrWriting, writing)
	}
}

// meditActLocked runs a room-audience message with $n/$N substitution,
// mirroring C's act(..., TO_ROOM, ...) around the OLC transitions. Act
// resolves the room from the actor, exactly as the C calls do.
func (s *Session) meditActLocked(actType int, format string) {
	if s.manager == nil || s.manager.world == nil || s.player == nil {
		return
	}
	game.Act(s.manager.world, false, s.player, nil, nil, nil, format, "", actType)
}

// meditShowMenuLocked renders the C medit_disp_menu layout (src/medit.c).
// The zone-name lookup mirrors medit_setup_existing's OLC_ZNUM-based display;
// the script name/flags come from the live shallow script (C copies the mob
// script pointer into OLC, so script edits are authoritative outside the
// working copy).
func (s *Session) meditShowMenuLocked() {
	state := s.mobEdit
	mob := &state.mob

	scriptName := mob.ScriptName
	scriptFlags := mob.LuaFunctions
	if state.isNew {
		scriptName = ""
		scriptFlags = 0
	} else if live, ok := s.manager.world.SnapshotMob(state.number); ok {
		scriptName = live.ScriptName
		scriptFlags = live.LuaFunctions
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "-- Mob number: [%5d]\r\n", state.number)
	fmt.Fprintf(&sb, "1) Sex: %-7.7s         2) Alias: %s\r\n", meditSexName(mob.Sex), mob.Keywords)
	fmt.Fprintf(&sb, "3) S-Desc: %s\r\n", mob.ShortDesc)
	fmt.Fprintf(&sb, "4) L-Desc: %s\r\n", strings.TrimSuffix(mob.LongDesc, "\r\n"))
	fmt.Fprintf(&sb, "5) D-Desc:\r\n%s", mob.DetailedDesc)
	fmt.Fprintf(&sb, "6) Level: [%4d], 7) Alignment: [%4d], 8) Hitroll: [%2d], 9) Damroll: [%2d]\r\n",
		mob.Level, mob.Alignment, meditDisplayHitroll(mob), mob.Damage.Plus)
	fmt.Fprintf(&sb, "A) NDD: [%2d], B) SDD: [%2d], C) Num HP Dice: [%2d], D) Size HP Dice: [%3d], E) HP Bonus: [%5d]\r\n",
		mob.Damage.Num, mob.Damage.Sides, mob.HP.Num, mob.HP.Sides, mob.HP.Plus)
	fmt.Fprintf(&sb, "F) Armor Class: [%3d], G) Exp: [%9d], H) Gold: [%8d]\r\n",
		mob.AC, mob.Exp, mob.Gold)
	fmt.Fprintf(&sb, "I) Position: %s\r\n", meditPosName(mob.Position))
	fmt.Fprintf(&sb, "J) Default position: %s\r\n", meditPosName(mob.DefaultPos))
	fmt.Fprintf(&sb, "K) Attack: %s\r\n", meditAttackName(mob.BareHandAttack))
	fmt.Fprintf(&sb, "L) NPC flags: %s\r\n", meditSprintBitArray(mob.ActionFlags, meditParserActionBits, meditMobFlagNames))
	fmt.Fprintf(&sb, "M) AFF flags: %s\r\n", meditSprintBitArray(mob.AffectFlags, meditParserAffectBits, meditAffFlagNames))
	fmt.Fprintf(&sb, "N) Race: %s\r\n", meditRaceName(mob.Race))
	fmt.Fprintf(&sb, "O) Noise: %s\r\n", mob.Noise)
	fmt.Fprintf(&sb, "S) Edit Mob Script: Name: %s Flags: %s\r\n", scriptName, meditSprintScriptFlags(scriptFlags))
	fmt.Fprintf(&sb, "Q) Quit\r\n")
	fmt.Fprintf(&sb, "Enter choice : ")

	s.meditSendLocked(sb.String())
	state.mode = meditMainMenu
}

func meditSexName(sex int) string {
	if sex >= 0 && sex < len(meditGenderNames) {
		return meditGenderNames[sex]
	}
	return "Error"
}

func meditPosName(pos int) string {
	if pos >= 0 && pos < len(meditPositionNames) {
		return meditPositionNames[pos]
	}
	// C indexes position_types[] out of bounds for 9-14 (undefined). The Go
	// port stores C's clamped value for .mob fidelity but refuses to read
	// past the table here; the values 9-14 are unreachable through valid
	// play and no defined C byte sequence exists to match.
	return fmt.Sprintf("unknown (%d)", pos)
}

func meditAttackName(attack int) string {
	if attack >= 0 && attack < len(meditAttackNames) {
		return meditAttackNames[attack]
	}
	return "Error"
}

func meditRaceName(race int) string {
	if race >= 0 && race < len(meditRaceNames) {
		return meditRaceNames[race]
	}
	return "Error"
}

// meditDisplayHitroll mirrors the C display expression 20 - GET_THAC0(mob): the
// parser stores the file THAC0 value, so the menu shows the hitroll the C
// server computes.
func meditDisplayHitroll(mob *parser.Mob) int {
	return 20 - mob.THAC0
}

// meditSprintBitArray mirrors sprintbitarray: each set flag renders through
// the C display table as "NAME " (trailing space after each name); an empty
// set renders "NOBITS ". The parser stores flags under its own spellings
// (meditParserActionBits / meditParserAffectBits), which share bit positions
// with the C menu tables.
func meditSprintBitArray(set []string, parserNames, displayNames []string) string {
	index := make(map[string]int, len(parserNames))
	for i, name := range parserNames {
		index[name] = i
	}
	var sb strings.Builder
	for _, name := range set {
		if i, ok := index[name]; ok && i < len(displayNames) {
			sb.WriteString(displayNames[i])
			sb.WriteString(" ")
		}
	}
	if sb.Len() == 0 {
		return "NOBITS "
	}
	return sb.String()
}

// meditSprintScriptFlags mirrors sprintbit over mscript_bits for the script
// flags segment: bit i renders mscript_bits[i] followed by a space; an empty
// set renders "NOBITS ".
func meditSprintScriptFlags(flags int) string {
	var sb strings.Builder
	for i, name := range meditScriptFlagNames {
		if flags&(1<<uint(i)) != 0 {
			sb.WriteString(name)
			sb.WriteString(" ")
		}
	}
	if sb.Len() == 0 {
		return "NOBITS "
	}
	return sb.String()
}

// meditShowSexLocked mirrors medit_disp_sex.
func (s *Session) meditShowSexLocked() {
	var sb strings.Builder
	for i, name := range meditGenderNames {
		fmt.Fprintf(&sb, "%2d) %-7.7s\r\n", i, name)
	}
	sb.WriteString("Enter sex for this mob : ")
	s.meditSendLocked(sb.String())
	s.mobEdit.mode = meditSex
}

// meditShowPositionsLocked mirrors medit_disp_positions. C terminates the
// list with position_types[NUM_POSITIONS]; the Go table holds only the nine
// real entries.
func (s *Session) meditShowPositionsLocked() {
	var sb strings.Builder
	for i, name := range meditPositionNames {
		fmt.Fprintf(&sb, "%2d) %-20.20s\r\n", i, name)
	}
	sb.WriteString("Enter position for this mob : ")
	s.meditSendLocked(sb.String())
}

// meditShowAttackTypesLocked mirrors medit_disp_attack_types.
func (s *Session) meditShowAttackTypesLocked() {
	var sb strings.Builder
	for i, name := range meditAttackNames {
		fmt.Fprintf(&sb, "%2d) %s\r\n", i, name)
	}
	sb.WriteString("Enter attack type for this mob : ")
	s.meditSendLocked(sb.String())
}

// meditShowRacesLocked mirrors medit_disp_races. C lays the 31 races out in
// two columns via get_char_cols; the captured C byte stream lays them out in
// a single column (the fixture terminal is narrower than 31 entries), so the
// Go port lists them in index order, one per line, like the other medit
// submenus.
func (s *Session) meditShowRacesLocked() {
	var sb strings.Builder
	for i, name := range meditRaceNames {
		fmt.Fprintf(&sb, "%2d) %-20.20s\r\n", i, name)
	}
	sb.WriteString("Enter race for this mob : ")
	s.meditSendLocked(sb.String())
}

// meditShowMobFlagsLocked mirrors medit_disp_mob_flags.
func (s *Session) meditShowMobFlagsLocked() {
	var sb strings.Builder
	for i, name := range meditMobFlagNames {
		fmt.Fprintf(&sb, "%2d) %-20.20s\r\n", i+1, name)
	}
	sb.WriteString("Enter npc flags (0 to quit) : ")
	s.meditSendLocked(sb.String())
}

// meditShowAffFlagsLocked mirrors medit_disp_aff_flags.
func (s *Session) meditShowAffFlagsLocked() {
	var sb strings.Builder
	for i, name := range meditAffFlagNames {
		fmt.Fprintf(&sb, "%2d) %-20.20s\r\n", i+1, name)
	}
	sb.WriteString("Enter aff flags (0 to quit) : ")
	s.meditSendLocked(sb.String())
}

// meditShowScriptMenuLocked mirrors medit_disp_script_menu (src/medit.c).
func (s *Session) meditShowScriptMenuLocked() {
	state := s.mobEdit
	scriptName := ""
	scriptFlags := 0
	if !state.isNew {
		if live, ok := s.manager.world.SnapshotMob(state.number); ok {
			scriptName = live.ScriptName
			scriptFlags = live.LuaFunctions
		}
	}
	var sb strings.Builder
	sb.WriteString("-- Mob script editor --\r\n")
	fmt.Fprintf(&sb, "1) Script name: %s\r\n", scriptName)
	fmt.Fprintf(&sb, "2) Script flags: %s\r\n", meditSprintScriptFlags(scriptFlags))
	sb.WriteString("0) Back to main menu\r\n")
	sb.WriteString("Enter choice: ")
	s.meditSendLocked(sb.String())
	state.mode = meditScriptMenu
}

// meditShowScriptFlagsLocked mirrors medit_disp_script_flags, including the
// clear-screen escape and the two-column flag layout the C loop produces
// (every second flag ends the line), followed by the "Current flags" line.
// Color codes are omitted: the Go port has no color system, matching C's
// no-color default.
func (s *Session) meditShowScriptFlagsLocked() {
	state := s.mobEdit
	scriptFlags := 0
	if !state.isNew {
		if live, ok := s.manager.world.SnapshotMob(state.number); ok {
			scriptFlags = live.LuaFunctions
		}
	}
	var sb strings.Builder
	sb.WriteString("\x1b[H\x1b[J")
	for i, name := range meditScriptFlagNames {
		fmt.Fprintf(&sb, "%2d) %-20.20s  ", i+1, name)
		if (i+1)%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&sb, "\r\nCurrent flags   : %s\r\n", meditSprintScriptFlags(scriptFlags))
	sb.WriteString("Enter script flags (0 to quit) : ")
	s.meditSendLocked(sb.String())
	state.mode = meditScriptFlags
}

// meditToggleParserFlag flips bit (0-based) in a parser storage-name flag
// set, returning the updated set. It mirrors C's TOGGLE_BIT on the working
// copy's flag array; the parser spellings are preserved for disk fidelity.
func meditToggleParserFlag(set []string, parserNames []string, bit int) []string {
	if bit < 0 || bit >= len(parserNames) {
		return set
	}
	target := parserNames[bit]
	for i, name := range set {
		if name == target {
			return append(set[:i:i], set[i+1:]...)
		}
	}
	return append(set, target)
}

// meditEnsureNPCFlag sets the ISNPC storage flag on the working copy,
// mirroring medit_parse's SET_BIT_AR(MOB_FLAGS(OLC_MOB(d)), MOB_ISNPC) at the
// top of MEDIT_CONFIRM_SAVESTRING.
func meditEnsureNPCFlag(mob *parser.Mob) {
	for _, name := range mob.ActionFlags {
		if name == "ISNPC" {
			return
		}
	}
	mob.ActionFlags = append(mob.ActionFlags, "ISNPC")
}

// handleMeditInput routes one raw input line to the active medit session,
// mirroring the CON_MEDIT case in interpreter.c (which calls medit_parse
// after skip_spaces). It returns true when the line was consumed by the
// editor.
func (s *Session) handleMeditInput(line string) bool {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	if s.mobEdit == nil {
		return false
	}
	// C skips leading spaces before medit_parse sees the line.
	s.parseMeditLocked(strings.TrimLeft(line, " \t"))
	return true
}

// parseMeditLocked is the Go port of medit_parse (src/medit.c). The input
// has already been left-trimmed, matching C's skip_spaces.
func (s *Session) parseMeditLocked(arg string) {
	state := s.mobEdit
	mob := &state.mob

	// C applies this gate to every mode above MEDIT_NUMERICAL_RESPONSE (10),
	// including MEDIT_SCRIPT_NAME (28) and MEDIT_SCRIPT_FLAGS (29). The
	// logic is quirky: it rejects empty input and a lone "-" (or
	// "-<non-digit>"), but a non-numeric string like "abc" passes and is
	// later read by atoi as 0. The port reproduces the quirk verbatim.
	if state.mode > meditNumericalResponse && !meditNumericalOK(arg) {
		s.meditSendLocked("Field must be numerical, try again : ")
		return
	}

	switch state.mode {
	case meditConfirmSaveString:
		meditEnsureNPCFlag(mob)
		switch firstByte(arg) {
		case 'y', 'Y':
			s.meditSendLocked("Saving mobile to memory.\r\n")
			s.saveMeditInternallyLocked()
			slog.Info("OLC: medit edit",
				"player", s.playerName, "mob", state.number)
			s.finishMeditLocked(cleanupMeditAll)
			return
		case 'n', 'N':
			s.finishMeditLocked(cleanupMeditAll)
			return
		default:
			s.meditSendLocked("Invalid choice!\r\n")
			s.meditSendLocked("Do you wish to save the mobile? : ")
			return
		}

	case meditMainMenu:
		s.parseMeditMainMenuLocked(arg)
		return

	case meditAlias:
		mob.Keywords = arg
	case meditSDesc:
		mob.ShortDesc = arg
	case meditLDesc:
		// C: strcpy(buf, arg); strcat(buf, "\r\n").
		mob.LongDesc = arg + "\r\n"
	case meditDDesc:
		// C: "We should never get here." The '5' choice enters the string
		// editor directly without leaving main-menu mode, so this case is
		// unreachable through valid play.
		slog.Error("medit reached D_DESC case", "player", s.playerName)
		s.finishMeditLocked(cleanupMeditAll)
		return
	case meditNPCFlags:
		i := atoiC(arg)
		if i == 0 {
			break
		}
		// C: if (!((i < 0) || (i > NUM_MOB_FLAGS))) { i--; TOGGLE_BIT_AR(...); }
		// NUM_MOB_FLAGS is 25; the menu numbers flags 1-25.
		if i > 0 && i <= len(meditMobFlagNames) {
			mob.ActionFlags = meditToggleParserFlag(mob.ActionFlags, meditParserActionBits, i-1)
		}
		s.meditShowMobFlagsLocked()
		return
	case meditAffFlags:
		i := atoiC(arg)
		if i == 0 {
			break
		}
		// C uses SET_BIT_AR here (not toggle!), despite the menu implying
		// toggling. NUM_AFF_FLAGS is 37; the menu numbers flags 1-37.
		if i > 0 && i <= len(meditAffFlagNames) {
			mob.AffectFlags = meditSetParserFlag(mob.AffectFlags, meditParserAffectBits, i-1)
		}
		s.meditShowAffFlagsLocked()
		return
	case meditNoise:
		// C: free old noise; str_dup(arg) if strlen(arg) > 2, else NULL.
		if len(arg) > 2 {
			mob.Noise = arg
		} else {
			mob.Noise = ""
		}
	case meditScriptMenu:
		i := atoiC(arg)
		switch i {
		case 0:
			break
		case 1:
			state.mode = meditScriptName
			s.meditSendLocked("Enter script name: ")
			return
		case 2:
			s.meditShowScriptFlagsLocked()
			return
		default:
			break
		}
	case meditScriptName:
		// The numerical gate above rejects empty input, so C's
		// "clear the name" branch (!strcmp(arg, "")) is unreachable.
		// C shallow-copies the mob script pointer into OLC: the name is
		// written to the LIVE prototype, not the working copy.
		s.setMeditScriptNameLocked(arg)
		s.meditShowScriptMenuLocked()
		return
	case meditScriptFlags:
		if i := atoiC(arg); i == 0 {
			s.meditShowScriptMenuLocked()
			return
		} else if i > 0 && i <= len(meditScriptFlagNames) {
			// C toggles bit (i-1) on the live shallow-copied script flags.
			s.toggleMeditScriptFlagLocked(i - 1)
		}
		s.meditShowScriptFlagsLocked()
		return

	case meditSex:
		mob.Sex = clampInt(atoiC(arg), 0, len(meditGenderNames)-1)
	case meditHitroll:
		// C sets GET_HITROLL; the parser stores file THAC0 = 20 - hitroll.
		mob.THAC0 = 20 - clampInt(atoiC(arg), 0, 127)
	case meditDamroll:
		mob.Damage.Plus = clampInt(atoiC(arg), 0, 127)
	case meditNDD:
		mob.Damage.Num = clampInt(atoiC(arg), 0, 127)
	case meditSDD:
		mob.Damage.Sides = clampInt(atoiC(arg), 0, 127)
	case meditNumHPDice:
		mob.HP.Num = clampInt(atoiC(arg), 0, 50)
	case meditSizeHPDice:
		mob.HP.Sides = clampInt(atoiC(arg), 0, 3000)
	case meditAddHP:
		mob.HP.Plus = clampInt(atoiC(arg), 0, 30000)
	case meditAC:
		mob.AC = clampInt(atoiC(arg), -200, 200)
	case meditExp:
		mob.Exp = maxInt(0, atoiC(arg))
	case meditGold:
		mob.Gold = maxInt(0, atoiC(arg))
	case meditPos:
		mob.Position = clampInt(atoiC(arg), 0, 14)
	case meditDefaultPos:
		mob.DefaultPos = clampInt(atoiC(arg), 0, 14)
	case meditAttack:
		mob.BareHandAttack = clampInt(atoiC(arg), 0, len(meditAttackNames)-1)
	case meditLevel:
		s.parseMeditLevelLocked(atoiC(arg))
	case meditAlignment:
		mob.Alignment = clampInt(atoiC(arg), -1000, 1000)
	case meditRace:
		mob.Race = clampInt(atoiC(arg), 0, len(meditRaceNames)-1)

	default:
		// C: "We should never get here." (sysserr + cleanup_olc).
		slog.Error("medit reached default case", "player", s.playerName)
		s.finishMeditLocked(cleanupMeditAll)
		return
	}

	state.olcVal = 1
	s.meditShowMenuLocked()
}

// meditNumericalOK reproduces C's medit_parse numerical gate verbatim:
// reject empty input, reject "-" alone, reject "-<non-digit>". Anything else
// (including non-numeric strings, which atoi later reads as 0) passes.
func meditNumericalOK(arg string) bool {
	if arg == "" {
		return false
	}
	if !isASCIIDigit(arg[0]) && arg[0] == '-' &&
		(len(arg) < 2 || !isASCIIDigit(arg[1])) {
		return false
	}
	return true
}

// parseMeditMainMenuLocked ports the MEDIT_MAIN_MENU switch. The choice is
// the first input byte (C switches on *arg). Text fields prompt
// "\r\nEnter new text :\r\n| ", numerical fields "\r\nEnter new value : ",
// and submenu choices display their menu immediately.
func (s *Session) parseMeditMainMenuLocked(arg string) {
	state := s.mobEdit
	choice := firstByte(arg)

	switch choice {
	case 'q', 'Q':
		if state.olcVal != 0 {
			s.meditSendLocked("Do you wish to save the changes to the mobile? (y/n) : ")
			state.mode = meditConfirmSaveString
		} else {
			s.finishMeditLocked(cleanupMeditAll)
		}
		return
	case '1':
		state.mode = meditSex
		s.meditShowSexLocked()
		return
	case '2':
		state.mode = meditAlias
	case '3':
		state.mode = meditSDesc
	case '4':
		state.mode = meditLDesc
	case '5':
		s.startMeditDescEditorLocked()
		return
	case '6':
		state.mode = meditLevel
	case '7':
		state.mode = meditAlignment
	case '8':
		state.mode = meditHitroll
	case '9':
		state.mode = meditDamroll
	case 'a', 'A':
		state.mode = meditNDD
	case 'b', 'B':
		state.mode = meditSDD
	case 'c', 'C':
		state.mode = meditNumHPDice
	case 'd', 'D':
		state.mode = meditSizeHPDice
	case 'e', 'E':
		state.mode = meditAddHP
	case 'f', 'F':
		state.mode = meditAC
	case 'g', 'G':
		state.mode = meditExp
	case 'h', 'H':
		state.mode = meditGold
	case 'i', 'I':
		state.mode = meditPos
		s.meditShowPositionsLocked()
		return
	case 'j', 'J':
		state.mode = meditDefaultPos
		s.meditShowPositionsLocked()
		return
	case 'k', 'K':
		state.mode = meditAttack
		s.meditShowAttackTypesLocked()
		return
	case 'l', 'L':
		state.mode = meditNPCFlags
		s.meditShowMobFlagsLocked()
		return
	case 'm', 'M':
		state.mode = meditAffFlags
		s.meditShowAffFlagsLocked()
		return
	case 'n', 'N':
		state.mode = meditRace
		s.meditShowRacesLocked()
		return
	case 'o', 'O':
		state.mode = meditNoise
	case 's', 'S':
		if state.isNew {
			// C: "Cannot assign a script until the mob is saved at least once."
			s.meditSendLocked("Cannot assign a script until the mob is saved at least once.\r\n")
			s.meditShowMenuLocked()
			return
		}
		s.meditShowScriptMenuLocked()
		return
	default:
		s.meditShowMenuLocked()
		return
	}

	// C tracks text vs numerical prompts with i (-1 = text, +1 = numerical).
	if state.mode == meditAlias || state.mode == meditSDesc ||
		state.mode == meditLDesc || state.mode == meditNoise {
		s.meditSendLocked("\r\nEnter new text :\r\n| ")
	} else {
		s.meditSendLocked("\r\nEnter new value : ")
	}
}

// parseMeditLevelLocked ports MEDIT_LEVEL's derived-stat computation. C sets
// the level, derives exp/HP/damage/AC/hitroll from it, then re-sets the level
// ("to handle 0s", allowing level 0 with level-1 stats). The /1.50 divisions
// are C float divisions truncated toward zero.
func (s *Session) parseMeditLevelLocked(raw int) {
	mob := &s.mobEdit.mob
	level := clampInt(raw, 1, 100)
	mob.Level = level

	if level <= len(meditExpLookup)-1 {
		mob.Exp = meditExpLookup[level]
	} else {
		// C indexes EXP_LOOKUP (41 entries) out of bounds for levels 41-100
		// (undefined). The Go port clamps the lookup instead of reproducing
		// the overread; the stored level still matches C.
		mob.Exp = meditExpLookup[len(meditExpLookup)-1]
	}

	if level > 10 {
		mob.Damage.Num = int(float64(level) / 1.50)
	} else {
		mob.Damage.Num = (level + 1) / 2
	}
	mob.Damage.Sides = 4
	if level > 10 {
		mob.Damage.Plus = int(float64(level+1) / 1.50)
	} else {
		mob.Damage.Plus = (level + 1) / 2
	}
	mob.HP.Num = level
	mob.HP.Sides = 0
	mob.HP.Plus = 0
	mob.AC = 100 - (10 * level)
	// C sets GET_HITROLL(level); the parser stores file THAC0 = 20 - hitroll.
	mob.THAC0 = 20 - level

	// C re-sets the level with a 0 floor ("to handle 0s").
	mob.Level = clampInt(raw, 0, 100)
}

// setMeditScriptNameLocked writes the script name to the LIVE prototype,
// mirroring C's shallow-copied mob script pointer (copy_mobile copies the
// pointer, so the menu edits the shared struct).
func (s *Session) setMeditScriptNameLocked(name string) {
	state := s.mobEdit
	if live, ok := s.manager.world.SnapshotMob(state.number); ok {
		live.ScriptName = name
		s.manager.world.SetMobScript(state.number, live.ScriptName, live.LuaFunctions)
	}
}

// toggleMeditScriptFlagLocked flips one script flag bit on the LIVE
// prototype, mirroring C's TOGGLE_BIT on the shallow-copied script.
func (s *Session) toggleMeditScriptFlagLocked(bit int) {
	state := s.mobEdit
	if live, ok := s.manager.world.SnapshotMob(state.number); ok {
		live.LuaFunctions ^= 1 << uint(bit)
		s.manager.world.SetMobScript(state.number, live.ScriptName, live.LuaFunctions)
	}
}

// meditSetParserFlag sets bit (0-based) in a parser storage-name flag set,
// mirroring C's SET_BIT_AR in MEDIT_AFF_FLAGS (which sets, not toggles,
// despite the menu's toggle implication).
func meditSetParserFlag(set []string, parserNames []string, bit int) []string {
	if bit < 0 || bit >= len(parserNames) {
		return set
	}
	target := parserNames[bit]
	for _, name := range set {
		if name == target {
			return set
		}
	}
	return append(set, target)
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// startMeditDescEditorLocked opens the improved string editor for the mob's
// detailed description, mirroring medit_parse's '5' choice: the help text,
// the "Enter mob description" prompt, the existing description, then the
// editor. C sets OLC_VAL(d) = 1 immediately when the editor opens (even if
// the user later aborts), and the port does the same.
func (s *Session) startMeditDescEditorLocked() {
	state := s.mobEdit
	state.olcVal = 1

	s.textEdit = &textEditState{
		field:    textEditField{name: "medit-desc", maxBytes: 1024},
		original: state.mob.DetailedDesc,
		buffer:   state.mob.DetailedDesc,
		onComplete: func(saved bool, text string) {
			s.textEditMu.Lock()
			defer s.textEditMu.Unlock()
			if s.mobEdit == nil {
				return
			}
			if saved {
				s.mobEdit.mob.DetailedDesc = text
			}
			s.meditShowMenuLocked()
		},
	}
	s.meditSendLocked("Instructions: /s or @ to save, /h for more options.\r\n" +
		"Enter mob description:\r\n\r\n" + state.mob.DetailedDesc)
}

// saveMeditInternallyLocked ports medit_save_internally (src/medit.c): the
// working copy is committed to the world (inserting or replacing the
// prototype), the editor is marked for a zone-file save, and a subsequent
// save writes the .mob file. C's rnum-shift of zone M-commands and shop
// keepers has no Go analogue: zone commands and shops key mobs by VNUM, so
// inserting under the VNUM key is the complete observable effect.
func (s *Session) saveMeditInternallyLocked() {
	state := s.mobEdit

	s.manager.world.CommitEditedMob(state.mob)

	meditSaveMu.Lock()
	meditSaveMobs[state.number] = true
	meditSaveMu.Unlock()

	// A builder-visible save writes the zone .mob file immediately, matching
	// the C flow where medit_save_internally queues OLC_SAVE_MOB and the
	// command loop's olc_save_list write follows.
	if zone, ok := olcZoneForVNum(s.manager.world, state.number); ok {
		if err := saveMeditZone(s.manager.world, zone); err != nil {
			slog.Error("medit disk save failed",
				"player", s.playerName, "zone", zone.Number, "error", err)
		}
	}
}

// saveMeditZone writes one zone's .mob file from the world prototypes,
// mirroring medit_save_to_disk (src/medit.c). Only mobs whose VNUM falls in
// the zone's [number*100, top] range are written, in ascending VNUM order
// (C walks the mob_index[] table, which is VNUM-ordered).
func saveMeditZone(world *game.World, zone *parser.Zone) error {
	parsed := world.GetParsedWorld()
	if parsed == nil || parsed.SourceDir == "" {
		return fmt.Errorf("world has no source directory")
	}
	// C's MOB_PREFIX is "world/mob"; the Go world derives lib roots from the
	// parsed source directory's parent (see World.LibTextDir).
	libDir := filepath.Join(parsed.SourceDir, "..", "mob")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		return err
	}

	var sb strings.Builder
	for _, mob := range world.SnapshotMobs() {
		if mob.VNum < zone.Number*100 || mob.VNum > zone.TopRoom {
			continue
		}
		writeMeditMob(&sb, &mob)
	}
	sb.WriteString("$\n")

	path := filepath.Join(libDir, fmt.Sprintf("%d.mob", zone.Number))
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return err
	}

	meditSaveMu.Lock()
	for _, mob := range world.SnapshotMobs() {
		if mob.VNum >= zone.Number*100 && mob.VNum <= zone.TopRoom {
			delete(meditSaveMobs, mob.VNum)
		}
	}
	meditSaveMu.Unlock()
	return nil
}

// writeMeditMob renders one mob record in the C .mob file format, mirroring
// medit_save_to_disk's fprintf sequence field for field (src/medit.c).
func writeMeditMob(sb *strings.Builder, mob *parser.Mob) {
	fmt.Fprintf(sb, "#%d\n", mob.VNum)
	fmt.Fprintf(sb, "%s~\n", mob.Keywords)
	fmt.Fprintf(sb, "%s~\n", mob.ShortDesc)
	// C strips carriage returns from the long and detailed descriptions
	// (strip_string) before writing; a NULL description writes "undefined".
	fmt.Fprintf(sb, "%s~\n", strings.ReplaceAll(mob.LongDesc, "\r", ""))
	fmt.Fprintf(sb, "%s~\n", strings.ReplaceAll(mob.DetailedDesc, "\r", ""))
	// C: mob flags (4 ints), aff flags (4 ints), alignment, then " E".
	writeMeditFlagInts(sb, mob.ActionFlags, meditParserActionBits)
	sb.WriteString(" ")
	writeMeditFlagInts(sb, mob.AffectFlags, meditParserAffectBits)
	fmt.Fprintf(sb, " %d E\n", mob.Alignment)
	// C: level, 20-hitroll (the file THAC0), AC/10, hit dice, dam dice.
	fmt.Fprintf(sb, "%d %d %d %dd%d+%d %dd%d+%d\n",
		mob.Level, mob.THAC0, mob.AC/10,
		mob.HP.Num, mob.HP.Sides, mob.HP.Plus,
		mob.Damage.Num, mob.Damage.Sides, mob.Damage.Plus)
	fmt.Fprintf(sb, "%d %d\n", mob.Gold, mob.Exp)
	fmt.Fprintf(sb, "%d %d %d\n", mob.Position, mob.DefaultPos, mob.Sex)
	if mob.BareHandAttack != 0 {
		fmt.Fprintf(sb, "BareHandAttack: %d\n", mob.BareHandAttack)
	}
	if mob.Race != 16 {
		fmt.Fprintf(sb, "Race: %d\n", mob.Race)
	}
	if mob.Noise != "" {
		fmt.Fprintf(sb, "Noise: %s\n", mob.Noise)
	}
	if mob.ScriptName != "" {
		fmt.Fprintf(sb, "Script: %s %d\n", mob.ScriptName, mob.LuaFunctions)
	}
	sb.WriteString("E\n")
}

// writeMeditFlagInts renders a flag set as four space-separated integers,
// mirroring C's "%d %d %d %d" for the MOB_FLAGS and AFF_FLAGS arrays. Bit i
// sets word i/32's bit i%32.
func writeMeditFlagInts(sb *strings.Builder, set []string, parserNames []string) {
	var words [4]int64
	index := make(map[string]int, len(parserNames))
	for i, name := range parserNames {
		index[name] = i
	}
	for _, name := range set {
		if i, ok := index[name]; ok && i < 128 {
			words[i/32] |= 1 << uint(i%32)
		}
	}
	fmt.Fprintf(sb, "%d %d %d %d", words[0], words[1], words[2], words[3])
}

// isMeditEditing reports whether a medit (CON_MEDIT) session is active and
// no descriptor string editor has taken over the line. It mirrors C's
// CON_MEDIT routing: while the D-description editor runs, lines go to the
// string editor instead of medit_parse.
func (s *Session) isMeditEditing() bool {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	return s.mobEdit != nil && s.textEdit == nil
}

// cancelMedit discards an in-progress medit session without saving, used on
// linkdead disconnect. It mirrors C's cleanup_olc(d, CLEANUP_ALL) on
// descriptor close.
func (s *Session) cancelMedit() {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	if s.mobEdit != nil {
		s.finishMeditLocked(cleanupMeditAll)
	}
}
