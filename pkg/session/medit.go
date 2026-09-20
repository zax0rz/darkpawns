package session

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
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

	if save {
		// C's duplicate scan runs on the save path too, against the zone's
		// base vnum; the save never reserves the mob itself.
		if other := s.manager.mobEditHolder(number); other != "" {
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
		s.meditSend("Saving all mobiles in zone.\r\n")
		slog.Info("OLC: medit zone save",
			"player", s.playerName, "zone", zone.Number)
		if err := saveMeditZone(s.manager.world, zone); err != nil {
			slog.Error("medit disk save failed",
				"player", s.playerName, "zone", zone.Number, "error", err)
		}
		return nil
	}

	// The duplicate-editor gate runs before zone lookup and authorization,
	// matching do_olc's descriptor scan order. The claim itself is the
	// check, so two sessions racing through cmdMedit cannot both be
	// admitted. Every refusal and failure path below releases the claim.
	if other, ok := s.manager.claimMobEdit(number, s); !ok {
		s.meditSend(fmt.Sprintf("That mobile is currently being edited by %s.\r\n", other))
		return nil
	}

	zone, ok := olcZoneForVNum(s.manager.world, number)
	if !ok {
		s.manager.releaseMobEdit(number, s)
		s.meditSend("Sorry, there is no zone for that number!\r\n")
		return nil
	}
	if !olcAuthorized(s, zone.Number) {
		s.manager.releaseMobEdit(number, s)
		s.meditSend("You do not have permission to edit this zone.\r\n")
		return nil
	}

	if err := s.startMedit(number, zone); err != nil {
		s.manager.releaseMobEdit(number, s)
		return err
	}
	return nil
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
	number := s.mobEdit.number
	switch mode {
	case cleanupMeditAll:
		s.setPlayerWritingLocked(false)
		if s.player != nil {
			s.meditActLocked(game.ToRoom, "$n stops using OLC.")
		}
		s.mobEdit = nil
	}
	if s.manager != nil {
		s.manager.releaseMobEdit(number, s)
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

// meditDisplayLongDesc mirrors C's fread_string output for the mob long
// description: the stored string always ends with "\r\n" (the line's newline
// is preserved before the "~" terminator). The Go parser strips it, so the
// menu display restores it for byte fidelity.
func meditDisplayLongDesc(longDesc string) string {
	if !strings.HasSuffix(longDesc, "\r\n") {
		return longDesc + "\r\n"
	}
	return longDesc
}

// meditShowMenuLocked renders medit_disp_menu (src/medit.c:611-678) byte for
// byte, including get_char_cols colors and the field widths/precisions of
// the C format strings. The zone-name lookup mirrors medit_setup_existing's
// OLC_ZNUM-based display; the script name/flags come from the live shallow
// script (C copies the mob script pointer into OLC, so script edits are
// authoritative outside the working copy).
func (s *Session) meditShowMenuLocked() {
	state := s.mobEdit
	mob := &state.mob
	nrm, grn, cyn, yel := s.reditCols()

	var sb strings.Builder
	// Direct port of medit_disp_menu (src/medit.c:611-678). The HP dice are
	// C's GET_HIT/GET_MANA/GET_MOVE; hitroll displays as 20 - THAC0.
	fmt.Fprintf(&sb, "\r\n-- Mob Number:  [%s%d%s]\r\n", cyn, state.number, nrm)
	fmt.Fprintf(&sb, "%s1%s) Sex: %s%-7.7s%s            %s2%s) Alias: %s%s\r\n",
		grn, nrm, yel, meditSexName(mob.Sex), nrm,
		grn, nrm, yel, mob.Keywords)
	fmt.Fprintf(&sb, "%s3%s) S-Desc: %s%s\r\n", grn, nrm, yel, mob.ShortDesc)
	fmt.Fprintf(&sb, "%s4%s) L-Desc:-\r\n%s%s", grn, nrm, yel, meditDisplayLongDesc(mob.LongDesc))
	fmt.Fprintf(&sb, "%s5%s) D-Desc:-\r\n%s%s", grn, nrm, yel, mob.DetailedDesc)
	fmt.Fprintf(&sb, "%s6%s) Level:       [%s%4d%s],  %s7%s) Alignment:    [%s%4d%s]\r\n",
		grn, nrm, cyn, mob.Level, nrm,
		grn, nrm, cyn, mob.Alignment, nrm)
	fmt.Fprintf(&sb, "%s8%s) Hitroll:     [%s%4d%s],  %s9%s) Damroll:      [%s%4d%s]\r\n",
		grn, nrm, cyn, meditDisplayHitroll(mob), nrm,
		grn, nrm, cyn, mob.Damage.Plus, nrm)
	fmt.Fprintf(&sb, "%sA%s) NumDamDice:  [%s%4d%s],  %sB%s) SizeDamDice:  [%s%4d%s]\r\n",
		grn, nrm, cyn, mob.Damage.Num, nrm,
		grn, nrm, cyn, mob.Damage.Sides, nrm)
	fmt.Fprintf(&sb, "%sC%s) Num HP Dice: [%s%4d%s],  %sD%s) Size HP Dice: [%s%4d%s],  %sE%s) HP Bonus: [%s%5d%s]\r\n",
		grn, nrm, cyn, mob.HP.Num, nrm,
		grn, nrm, cyn, mob.HP.Sides, nrm,
		grn, nrm, cyn, mob.HP.Plus, nrm)
	fmt.Fprintf(&sb, "%sF%s) Armor Class: [%s%4d%s],  %sG%s) Exp:     [%s%9d%s],  %sH%s) Gold:  [%s%8d%s]\r\n",
		grn, nrm, cyn, mob.AC, nrm,
		grn, nrm, cyn, mob.Exp, nrm,
		grn, nrm, cyn, mob.Gold, nrm)

	noise := mob.Noise
	if noise == "" {
		noise = "None"
	}
	fmt.Fprintf(&sb, "%sI%s) Position  : %s%s\r\n", grn, nrm, yel, meditPosName(mob.Position))
	fmt.Fprintf(&sb, "%sJ%s) Default   : %s%s\r\n", grn, nrm, yel, meditPosName(mob.DefaultPos))
	fmt.Fprintf(&sb, "%sK%s) Attack    : %s%s\r\n", grn, nrm, yel, meditAttackName(mob.BareHandAttack))
	fmt.Fprintf(&sb, "%sL%s) NPC Flags : %s%s\r\n", grn, nrm, cyn,
		meditSprintBitArray(mob.ActionFlags, meditParserActionBits, meditMobFlagNames))
	fmt.Fprintf(&sb, "%sM%s) AFF Flags : %s%s\r\n", grn, nrm, cyn,
		meditSprintBitArray(mob.AffectFlags, meditParserAffectBits, meditAffFlagNames))
	fmt.Fprintf(&sb, "%sN%s) Race      : %s%s\r\n", grn, nrm, cyn, meditRaceName(mob.Race))
	fmt.Fprintf(&sb, "%sO%s) Noise     : %s%s\r\n", grn, nrm, cyn, noise)
	fmt.Fprintf(&sb, "%sS%s) Script Menu   \r\n", grn, nrm)
	fmt.Fprintf(&sb, "%sQ%s) Quit\r\n", grn, nrm)
	sb.WriteString("Enter choice : ")
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
	inSet := make(map[string]bool, len(set))
	for _, name := range set {
		inSet[name] = true
	}
	var sb strings.Builder
	// C's sprintnbit walks bits 0..n in order; iterate the parser bit table
	// (which mirrors the C bit positions) rather than the set's storage order.
	for i, parserName := range parserNames {
		if i >= len(displayNames) {
			break
		}
		if inSet[parserName] {
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
// flags segment: bit i renders mscript_bits[i] followed by a space. C's
// sprintbit (singular) falls back to "NOBITS " for an empty set.
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
// meditShowSexLocked mirrors medit_disp_sex (src/medit.c:458-476): a leading
// blank line, one "%s%2d%s) %s" line per gender with get_char_cols colors,
// and the "Enter gender number : " prompt.
func (s *Session) meditShowSexLocked() {
	nrm, grn, _, _ := s.reditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	for i, name := range meditGenderNames {
		fmt.Fprintf(&sb, "%s%2d%s) %s\r\n", grn, i, nrm, name)
	}
	sb.WriteString("Enter gender number : ")
	s.meditSendLocked(sb.String())
}

// meditShowPositionsLocked mirrors medit_disp_positions (src/medit.c:458-476):
// single column, "%s%2d%s) %s" per position. C's trailing
// position_types[NUM_POSITIONS] is "" and the display loop skips empty names,
// so only the nine real positions print.
func (s *Session) meditShowPositionsLocked() {
	nrm, grn, _, _ := s.reditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	for i, name := range meditPositionNames {
		fmt.Fprintf(&sb, "%s%2d%s) %s\r\n", grn, i, nrm, name)
	}
	sb.WriteString("Enter position number : ")
	s.meditSendLocked(sb.String())
}

// meditShowAttackTypesLocked mirrors medit_disp_attack_types (src/medit.c:497-514).
func (s *Session) meditShowAttackTypesLocked() {
	nrm, grn, _, _ := s.reditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	for i, name := range meditAttackNames {
		fmt.Fprintf(&sb, "%s%2d%s) %s\r\n", grn, i, nrm, name)
	}
	sb.WriteString("Enter attack type : ")
	s.meditSendLocked(sb.String())
}

// meditShowRacesLocked mirrors medit_disp_races (src/medit.c:590-610): one
// "%s%2d%s) %-20.20s  " per race numbered from 0, with "\r\n" appended after
// every second entry; a trailing odd entry gets no newline of its own.
func (s *Session) meditShowRacesLocked() {
	nrm, grn, _, _ := s.reditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	columns := 0
	for i, name := range meditRaceNames {
		fmt.Fprintf(&sb, "%s%2d%s) %-20.20s  ", grn, i, nrm, name)
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	sb.WriteString("\r\nEnter mob race : ")
	s.meditSendLocked(sb.String())
}

// meditShowMobFlagsLocked mirrors medit_disp_mob_flags (src/medit.c:516-540):
// one "%s%2d%s) %-20.20s  " per flag numbered from 1, "\r\n" after every
// second entry, then "\r\nCurrent flags : %s%s%s\r\n" (cyn around the
// sprintbitarray output) and "Enter mob flags (0 to quit) : ".
func (s *Session) meditShowMobFlagsLocked() {
	nrm, grn, cyn, _ := s.reditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	columns := 0
	for i, name := range meditMobFlagNames {
		fmt.Fprintf(&sb, "%s%2d%s) %-20.20s  ", grn, i+1, nrm, name)
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&sb, "\r\nCurrent flags : %s%s%s\r\n", cyn,
		meditSprintBitArray(s.mobEdit.mob.ActionFlags, meditParserActionBits, meditMobFlagNames), nrm)
	sb.WriteString("Enter mob flags (0 to quit) : ")
	s.meditSendLocked(sb.String())
}

// meditShowAffFlagsLocked mirrors medit_disp_aff_flags (src/medit.c:542-562):
// same layout as the NPC flags, but the current-flags line has three spaces
// after "flags" and the prompt is "Enter aff flags (0 to quit) : ".
func (s *Session) meditShowAffFlagsLocked() {
	nrm, grn, cyn, _ := s.reditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	columns := 0
	for i, name := range meditAffFlagNames {
		fmt.Fprintf(&sb, "%s%2d%s) %-20.20s  ", grn, i+1, nrm, name)
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&sb, "\r\nCurrent flags   : %s%s%s\r\n", cyn,
		meditSprintBitArray(s.mobEdit.mob.AffectFlags, meditParserAffectBits, meditAffFlagNames), nrm)
	sb.WriteString("Enter aff flags (0 to quit) : ")
	s.meditSendLocked(sb.String())
}

// meditShowScriptMenuLocked mirrors medit_disp_script_menu (src/medit.c:679-710).
// A new (never-saved) mob gets "\r\nCannot assign a script until the mob is
// saved at least once.\r\n" followed by the main menu; otherwise the two
// script-menu lines with get_char_cols colors and "Enter choice (0 to quit) : ".
func (s *Session) meditShowScriptMenuLocked() {
	state := s.mobEdit
	nrm, grn, _, yel := s.reditCols()
	if state.isNew {
		s.meditSendLocked("\r\nCannot assign a script until the mob is saved at least once.\r\n")
		s.meditShowMenuLocked()
		return
	}
	scriptName := "None"
	scriptFlags := 0
	if live, ok := s.manager.world.SnapshotMob(state.number); ok {
		if live.ScriptName != "" {
			scriptName = live.ScriptName
		}
		scriptFlags = live.LuaFunctions
	}
	var sb strings.Builder
	sb.WriteString("\r\n")
	fmt.Fprintf(&sb, "%s1%s) Name: %s%s\r\n", grn, nrm, yel, scriptName)
	fmt.Fprintf(&sb, "%s2%s) Script Flags: %s%s%s\r\n", grn, nrm, yel, meditSprintScriptFlags(scriptFlags), nrm)
	sb.WriteString("Enter choice (0 to quit) : ")
	s.meditSendLocked(sb.String())
	state.mode = meditScriptMenu
}

// meditShowScriptFlagsLocked mirrors medit_disp_script_flags (src/medit.c:564-588):
// the ANSI clear-screen sequence (ESC[H ESC[J, verified as real escape bytes
// in the C source), the two-column flag list, "\r\nCurrent flags   : %s%s%s\r\n"
// and "Enter script flags (0 to quit) : ".
func (s *Session) meditShowScriptFlagsLocked() {
	state := s.mobEdit
	nrm, grn, cyn, _ := s.reditCols()
	scriptFlags := 0
	if !state.isNew {
		if live, ok := s.manager.world.SnapshotMob(state.number); ok {
			scriptFlags = live.LuaFunctions
		}
	}
	var sb strings.Builder
	sb.WriteString("\x1b[H\x1b[J")
	columns := 0
	for i, name := range meditScriptFlagNames {
		fmt.Fprintf(&sb, "%s%2d%s) %-20.20s  ", grn, i+1, nrm, name)
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&sb, "\r\nCurrent flags   : %s%s%s\r\n", cyn, meditSprintScriptFlags(scriptFlags), nrm)
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
		applyOLC(olc.Operation{Kind: olc.OpSetMobKeywords, Mob: mob, Text: arg})
	case meditSDesc:
		applyOLC(olc.Operation{Kind: olc.OpSetMobShortDescription, Mob: mob, Text: arg})
	case meditLDesc:
		// C: strcpy(buf, arg); strcat(buf, "\r\n").
		applyOLC(olc.Operation{Kind: olc.OpSetMobLongDescription, Mob: mob, Text: arg})
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
		// C toggles the AFF bit (TOGGLE_BIT_AR), mirroring the NPC and
		// script flag menus. The menu numbers flags 1-37.
		if i > 0 && i <= len(meditAffFlagNames) {
			mob.AffectFlags = meditToggleParserFlag(mob.AffectFlags, meditParserAffectBits, i-1)
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
		applyOLC(olc.Operation{Kind: olc.OpSetMobSex, Mob: mob, Value: atoiC(arg)})
	case meditHitroll:
		// C sets GET_HITROLL; the parser stores file THAC0 = 20 - hitroll.
		applyOLC(olc.Operation{Kind: olc.OpSetMobHitroll, Mob: mob, Value: atoiC(arg)})
	case meditDamroll:
		applyOLC(olc.Operation{Kind: olc.OpSetMobDamroll, Mob: mob, Value: atoiC(arg)})
	case meditNDD:
		applyOLC(olc.Operation{Kind: olc.OpSetMobNumDamageDice, Mob: mob, Value: atoiC(arg)})
	case meditSDD:
		applyOLC(olc.Operation{Kind: olc.OpSetMobSizeDamageDice, Mob: mob, Value: atoiC(arg)})
	case meditNumHPDice:
		applyOLC(olc.Operation{Kind: olc.OpSetMobNumHPDice, Mob: mob, Value: atoiC(arg)})
	case meditSizeHPDice:
		applyOLC(olc.Operation{Kind: olc.OpSetMobSizeHPDice, Mob: mob, Value: atoiC(arg)})
	case meditAddHP:
		applyOLC(olc.Operation{Kind: olc.OpSetMobAddHP, Mob: mob, Value: atoiC(arg)})
	case meditAC:
		applyOLC(olc.Operation{Kind: olc.OpSetMobAC, Mob: mob, Value: atoiC(arg)})
	case meditExp:
		applyOLC(olc.Operation{Kind: olc.OpSetMobExp, Mob: mob, Value: atoiC(arg)})
	case meditGold:
		applyOLC(olc.Operation{Kind: olc.OpSetMobGold, Mob: mob, Value: atoiC(arg)})
	case meditPos:
		applyOLC(olc.Operation{Kind: olc.OpSetMobPosition, Mob: mob, Value: atoiC(arg)})
	case meditDefaultPos:
		applyOLC(olc.Operation{Kind: olc.OpSetMobDefaultPosition, Mob: mob, Value: atoiC(arg)})
	case meditAttack:
		applyOLC(olc.Operation{Kind: olc.OpSetMobAttack, Mob: mob, Value: atoiC(arg)})
	case meditLevel:
		applyOLC(olc.Operation{Kind: olc.OpSetMobLevel, Mob: mob, Value: atoiC(arg)})
	case meditAlignment:
		applyOLC(olc.Operation{Kind: olc.OpSetMobAlignment, Mob: mob, Value: atoiC(arg)})
	case meditRace:
		applyOLC(olc.Operation{Kind: olc.OpSetMobRace, Mob: mob, Value: atoiC(arg)})

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
		state.mode = meditScriptMenu
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

// startMeditDescEditorLocked opens the improved string editor for the mob's
// detailed description, mirroring medit_parse's '5' choice: the help text,
// the "Enter mob description" prompt, the existing description, then the
// editor. C sets OLC_VAL(d) = 1 immediately when the editor opens (even if
// the user later aborts), and the port does the same.
func (s *Session) startMeditDescEditorLocked() {
	state := s.mobEdit
	state.olcVal = 1

	// The improved string editor works on CRLF text; the parser stores
	// DetailedDesc with LF endings, so normalize on the way in and back.
	initial := editorCRLF(state.mob.DetailedDesc)
	s.textEdit = &textEditState{
		field:      textEditField{name: "medit-desc", maxBytes: olc.MaxMobDesc},
		original:   initial,
		buffer:     initial,
		roomEditor: true,
		onComplete: func(action textEditAction, buffer, original string) {
			s.textEditMu.Lock()
			defer s.textEditMu.Unlock()
			if s.mobEdit == nil {
				return
			}
			if action == textEditSave {
				applyOLC(olc.Operation{
					Kind: olc.OpSetMobDetailedDescription,
					Mob:  &s.mobEdit.mob,
					Text: editorToRoomText(buffer),
				})
			}
			s.meditShowMenuLocked()
		},
	}
	s.meditSendLocked("Instructions: /s or @ to save, /h for more options.\r\n" +
		"Enter mob description:\r\n\r\n" + initial)
}

// saveMeditInternallyLocked ports medit_save_internally (src/medit.c): the
// working copy is committed to the world (inserting or replacing the
// prototype), standing live instances get the five C-specified display
// strings, and the zone is added to the OLC save list. C performs no disk
// write here; the .mob file is written by the separate save command
// (medit_save_to_disk) or saveall. C's rnum-shift of zone M-commands and
// shop keepers has no Go analogue: zone commands and shops key mobs by VNUM,
// so inserting under the VNUM key is the complete observable effect.
func (s *Session) saveMeditInternallyLocked() {
	state := s.mobEdit

	// Hold the zone save lock across the world commit and the dirty-marker
	// set so the pair is atomic against a concurrent saveMeditZone: without
	// this, a commit landing between the save's snapshot and its marker
	// cleanup would be cleared from the save list without being persisted.
	saveMu := zoneSaveLock(state.zoneNumber)
	saveMu.Lock()
	s.manager.world.CommitEditedMob(state.mob)
	s.manager.world.RefreshLiveMobStrings(state.number, state.mob)
	markOLCDirty(olcKindMob, state.zoneNumber)
	saveMu.Unlock()
}

// saveMeditZone writes one zone's .mob file from the world prototypes,
// mirroring medit_save_to_disk (src/medit.c). Only mobs whose VNUM falls in
// the zone's [number*100, top] range are written, in ascending VNUM order
// (C walks the mob_index[] table, which is VNUM-ordered).
func saveMeditZone(world *game.World, zone *parser.Zone) error {
	saveMu := zoneSaveLock(zone.Number)
	saveMu.Lock()
	defer saveMu.Unlock()
	return saveMeditZoneLocked(world, zone)
}

func saveMeditZoneLocked(world *game.World, zone *parser.Zone) error {
	parsed := world.GetParsedWorld()
	if parsed == nil || parsed.SourceDir == "" {
		return fmt.Errorf("world has no source directory")
	}
	// C's MOB_PREFIX is "world/mob": the mob files are a sibling of the wld
	// directory inside the world directory (SourceDir is the "world"
	// directory itself — same resolution as saveOeditZone's "obj"). The old
	// "../mob" form resolved to <lib>/mob, a directory the loader never
	// reads, so disk saves silently landed where boots never find them.
	libDir := filepath.Join(parsed.SourceDir, "mob")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		return err
	}

	// Serialize the whole snapshot -> write -> save-list cleanup against
	// concurrent saves of this zone and against working-copy commits (see
	// saveMeditInternallyLocked): a commit landing mid-save is either fully
	// inside the snapshot or keeps its dirty marker, never silently marked
	// saved.
	mobs := world.SnapshotMobs()
	var sb strings.Builder
	for i := range mobs {
		mob := &mobs[i]
		if mob.VNum < zone.Number*100 || mob.VNum > zone.TopRoom {
			continue
		}
		writeMeditMob(&sb, mob)
	}
	sb.WriteString("$\n")

	path := filepath.Join(libDir, fmt.Sprintf("%d.mob", zone.Number))
	if err := atomicWriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return err
	}

	clearOLCDirty(olcKindMob, zone.Number)
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

// IsMeditEditing reports whether a medit (CON_MEDIT) session is active and
// no descriptor string editor has taken over the line. Exported for the
// telnet listener's input dispatch: like CON_TEDIT and CON_REDIT, CON_MEDIT
// owns every complete input line including bare <ENTER>.
func (s *Session) IsMeditEditing() bool { return s.isMeditEditing() }

// isMeditEditing reports whether a medit (CON_MEDIT) session is active and
// no descriptor string editor has taken over the line. It mirrors C's
// CON_MEDIT routing: while the D-description editor runs, lines go to the
// string editor instead of medit_parse.
func (s *Session) isMeditEditing() bool {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	return s.mobEdit != nil && s.textEdit == nil
}

// isMobEditing reports whether a medit session owns this descriptor,
// including while its D-description string editor is active. It mirrors
// SendPrompt's redit suppression: C's make_prompt writes "] " while d->str
// is set and no ordinary prompt while an OLC menu owns the input.
func (s *Session) isMobEditing() bool {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	return s.mobEdit != nil
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
