package session

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// This file ports CircleMUD 3.0's oedit.c (the OLC object editor) behind the
// CON_OEDIT-equivalent descriptor state. The architecture follows
// pkg/session/redit.go and pkg/session/medit.go: the descriptor owns a working
// object copy, later input lines route through its menu state, save/abort/
// disconnect mirror cleanup_olc, and the audience transitions ($n starts/stops
// using OLC) match C. All player-facing bytes are copied verbatim from
// src/oedit.c (R1).
//
// The verified C traps are reproduced deliberately and called out in comments:
// the ITEM_WEAPON val3 fallthrough (oedit.c:1333-1340), the lying
// "Enter cost (10000 max): " prompt (:1105) over a bare atoi (:1243-1245),
// type 0 being unselectable while the menu lists from 0 (:1182), the
// extras-vs-wear error asymmetry (:1191 vs :1215), and the float percent-load
// rounding/rendering (:1004-1010, :1248-1257, :968-970).

// oeditMode mirrors the OEDIT_* constants in src/olc.h (values 1..28). Keeping
// the state names local makes the descriptor transition table auditable without
// pretending that the other OLC families share this implementation.
type oeditMode uint8

const (
	oeditMainMenu oeditMode = iota + 1
	oeditEditNamelist
	oeditShortDesc
	oeditLongDesc
	oeditActDesc
	oeditType
	oeditExtras
	oeditWear
	oeditWeight
	oeditCost
	oeditCostPerDay
	oeditTimer
	oeditValue1
	oeditValue2
	oeditValue3
	oeditValue4
	oeditApply
	oeditApplyMod
	oeditExtraDescKey
	oeditConfirmSavedB
	oeditConfirmSaveString
	oeditPromptApply
	oeditExtraDescDescription
	oeditExtraDescMenu
	oeditLevel
	oeditScriptName
	oeditScriptFlags
	oeditScriptMenu
)

// C's olc.h numeric limits for the object editor.
const (
	numItemTypes    = 24 // NUM_ITEM_TYPES
	numItemFlags    = 29 // NUM_ITEM_FLAGS
	numItemWears    = 19 // NUM_ITEM_WEARS
	numApplies      = 30 // NUM_APPLIES
	numLiqTypes     = 16 // NUM_LIQ_TYPES
	numSpells       = 104
	numAffFlags     = 37 // NUM_AFF_FLAGS
	numAttackTypes  = 15 // NUM_ATTACK_TYPES
	numOScriptFlags = 3  // NUM_OSCRIPT_FLAGS
	maxObjAffect    = parser.MAX_OBJ_AFFECT

	applyRaceHate = 25 // APPLY_RACE_HATE
	applySpell    = 29 // APPLY_SPELL

)

// C ITEM_* ordinals used by the oedit value cascade (src/structs.h:420-442).
const (
	itemLight      = 1
	itemScroll     = 2
	itemWand       = 3
	itemStaff      = 4
	itemWeapon     = 5
	itemFireweapon = 6
	itemMissile    = 7
	itemArmor      = 9
	itemPotion     = 10
	itemContainer  = 15
	itemNote       = 16
	itemDrinkcon   = 17
	itemFood       = 19
	itemMoney      = 20
	itemFountain   = 23
)

// oeditState is the descriptor-owned working copy, mirroring C's
// struct olc_data for CON_OEDIT: the object under edit, the OLC number, the
// mode, OLC_VAL (selected affect slot / has-changed flag) and the extra
// description cursor.
type oeditState struct {
	obj        parser.Obj
	number     int
	zoneNumber int
	isNew      bool
	mode       oeditMode
	olcVal     int

	// timer mirrors C's obj_flags.timer, which the main menu renders and
	// OEDIT_TIMER edits. parser.Obj has no timer field: the C .obj reader
	// never loads one (db.c:1431 reads "%d %d %f" only) and the oedit writer
	// never emits one, so a freshly booted prototype is always 0. The port
	// keeps the value session-local so the in-editor bytes match C exactly;
	// the gap is recorded in docs/fidelity/depth/oedit.tsv.
	timer int

	// affects mirrors C's fixed affected[MAX_OBJ_AFFECT] array. The parser
	// stores variable-length ObjAffect slices, so slots beyond the stored
	// length are zeroed here exactly as clear_object leaves them in C.
	affects []parser.ObjAffect

	// extraMeta tracks C's NULL-ness for the ex_description linked list,
	// which the parser's plain strings cannot represent: an unset keyword or
	// description renders "<NONE>" and gates the "goto next" transition.
	extraMeta    []oeditExtraMeta
	currentExtra int

	// pendingOutput mirrors C's descriptor output buffer: one input turn's
	// write_to_output calls accumulate here and flush as a single message, so
	// the Go transport never injects a framing newline between two C sends
	// (e.g. oedit_disp_menu's two send_to_char halves). Same shape as redit.
	pendingOutput string
}

// oeditExtraMeta mirrors reditExtraMeta: whether the C node's keyword and
// description pointers were non-NULL.
type oeditExtraMeta struct {
	keywordSet     bool
	descriptionSet bool
}

// objScriptFlags names C's oscript_bits[] (src/constants.c); NUM_OSCRIPT_FLAGS
// is 3. This is the object script table, distinct from redit's rscript_bits.
var oeditScriptFlagNames = []string{"NONE", "ONCMD", "ONPULSE"}

// The display tables below are byte-faithful copies of the C constant arrays
// the oedit menus print (src/constants.c, src/fight.c). The parser's
// storage-side tables (pkg/parser) intentionally use different spellings and
// are not used for menu rendering.

// C item_types[] (src/constants.c:782); NUM_ITEM_TYPES is 24.
var oeditItemTypes = []string{
	"UNDEFINED", "LIGHT", "SCROLL", "WAND", "STAFF", "WEAPON", "FIRE WEAPON",
	"MISSILE", "TREASURE", "ARMOR", "POTION", "WORN", "OTHER", "TRASH", "TRAP",
	"CONTAINER", "NOTE", "LIQ CONTAINER", "KEY", "FOOD", "MONEY", "PEN", "BOAT",
	"FOUNTAIN",
}

// C extra_bits[] (src/constants.c:837); NUM_ITEM_FLAGS is 29.
var oeditExtraBits = []string{
	"GLOW", "HUM", "!RENT", "!DONATE", "!INVIS", "INVIS", "MAGIC", "!DROP",
	"BLESS", "!GOOD", "!EVIL", "!NEU", "!MAGE", "!CLE", "!THI", "!WAR", "!SELL",
	"NAMED", "!PSI", "!NIN", "!PAL", "!MAGUS", "!ASS", "!AVA", "RARE", "!LOCATE",
	"!RAN", "!MYS", "TWOHANDS",
}

// C wear_bits[] (src/constants.c:812); NUM_ITEM_WEARS is 19.
var oeditWearBits = []string{
	"TAKE", "FINGER", "NECK", "BODY", "HEAD", "LEGS", "FEET", "HANDS", "ARMS",
	"SHIELD", "ABOUT", "WAIST", "WRIST", "WIELD", "HOLD", "THROW", "ABLEGS",
	"FACE", "HOVER",
}

// C apply_types[] (src/constants.c:872); NUM_APPLIES is 30.
var oeditApplyTypes = []string{
	"NONE", "STR", "DEX", "INT", "WIS", "CON", "CHA", "CLASS", "LEVEL", "AGE",
	"CHAR_WEIGHT", "CHAR_HEIGHT", "MAXMANA", "MAXHIT", "MAXMOVE", "GOLD", "EXP",
	"ARMOR", "HITROLL", "DAMROLL", "SAVING_PARA", "SAVING_ROD", "SAVING_PETRI",
	"SAVING_BREATH", "SAVING_SPELL", "RACE_HATE", "HIT_REGEN", "MANA_REGEN",
	"MOVE_REGEN", "PERM_SPELL",
}

// C container_bits[] (src/constants.c:908).
var oeditContainerBits = []string{"CLOSEABLE", "PICKPROOF", "CLOSED", "LOCKED"}

// C drinks[] (src/constants.c:918); NUM_LIQ_TYPES is 16.
var oeditDrinks = []string{
	"water", "beer", "wine", "ale", "dark ale", "whiskey", "lemonade",
	"firebreather", "local speciality", "slime mold juice", "milk", "tea",
	"coffee", "blood", "salt water", "clear water",
}

// C affected_bits[] (src/constants.c:596) truncated to NUM_AFF_FLAGS (37)
// entries, which is the bound oedit_disp_spell_menu iterates for the permanent
// spell-effect apply.
var oeditAffectBits = []string{
	"BLIND", "INVIS", "DET-ALIGN", "DET-INVIS", "DET-MAGIC", "SENSE-LIFE",
	"WATERWALK", "SANCT", "GROUP", "CURSE", "INFRA", "POISON", "PROT-EVIL",
	"PROT-GOOD", "SLEEP", "!TRACK", "FLESH-ALTER", "DODGE", "SNEAK", "HIDE",
	"BERSERK", "CHARM", "FOLLOW", "WIMPY", "KUJI-KIRI", "CUTTHROAT", "FLY",
	"WEREWOLF", "VAMPIRE", "MOUNTED", "INVULN", "FLAMING", "NOTHING", "HASTE",
	"SLOW", "DREAM", "WATERBREATHE",
}

// C spells[] (src/spell_parser.c) truncated to NUM_SPELLS (104) entries. The
// C array continues into the skill names, but oedit_disp_spells_menu iterates
// only 0..NUM_SPELLS-1.
var oeditSpellNames = []string{
	"!RESERVED!", "holy ward", "shift reality", "bless", "blindness",
	"burning hands", "call lightning", "charm person", "chill touch", "clone",
	"color spray", "control weather", "create food", "create water",
	"cure blind", "cure critic", "cure light", "curse", "detect alignment",
	"detect invisibility", "detect magic", "detect poison", "dispel evil",
	"earthquake", "enchant weapon", "energy drain", "fireball", "harm", "heal",
	"invisibility", "lightning bolt", "locate object", "flame arrow", "poison",
	"protection from evil", "remove curse", "sanctuary", "shocking grasp",
	"sleep", "strength", "summon", "meteor swarm", "word of recall",
	"remove poison", "sense life", "animate dead", "dispel good", "holy shield",
	"group heal", "group recall", "infravision", "waterwalk", "mass heal",
	"fly", "lycanthropy", "vampirism", "sobriety", "group invisibility",
	"hellfire", "enchant armor", "identify", "mind poke", "mind blast",
	"chameleon", "levitate", "metalskin", "globe of invulnerability",
	"vitality", "invigorate", "lesser perception", "greater perception",
	"mind attack", "adrenaline boost", "psychic shield", "change density",
	"acid blast", "dominate", "cell adjustment", "zen", "mirror image",
	"mass dominate", "divine intervention", "mind bar", "soul leech", "mindsight",
	"transparency", "know align", "gate", "word of intellect", "lay hands",
	"mental lapse", "smokescreen", "ray of disruption", "disintegration",
	"calliope", "protection from good", "flame strike", "haste", "slow",
	"dream travel", "psiblast", "glyph of summoning", "waterbreathe",
	"!drowning!",
}

// meditRaceNames and meditAttackNames are the same C mob_races[] and
// attack_hit_text[] tables the oedit race/weapon menus print, so oedit reuses
// them rather than duplicating the arrays.

// cmdOedit is the Go port of do_olc's SCMD_OLC_OEDIT branch (src/olc.c).
func cmdOedit(s *Session, args []string) error {
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

	// C: save = !strncmp(buf1, "save", 4) — case-sensitive, like medit.
	save := strings.HasPrefix(buf1, "save")

	if buf1 == "" {
		// olc_scmd_info[SCMD_OLC_OEDIT].text is "object".
		s.oeditSend("Specify a object VNUM to edit.\r\n")
		return nil
	}

	var number int
	if !isASCIIDigit(firstByte(buf1)) {
		if save {
			if buf2 == "" {
				s.oeditSend("Save which zone?\r\n")
				return nil
			}
			number = atoiC(buf2) * 100
		} else {
			s.oeditSend("Yikes!  Stop that, someone will get hurt!\r\n")
			return nil
		}
	} else {
		number = atoiC(buf1)
	}

	if save {
		// C's duplicate scan runs on the save path too, against the zone's
		// base vnum; the save never reserves the object itself.
		if other := s.manager.objEditHolder(number); other != "" {
			s.oeditSend(fmt.Sprintf("That object is currently being edited by %s.\r\n", other))
			return nil
		}
		zone, ok := olcZoneForVNum(s.manager.world, number)
		if !ok {
			s.oeditSend("Sorry, there is no zone for that number!\r\n")
			return nil
		}
		if !olcAuthorized(s, zone.Number) {
			s.oeditSend("You do not have permission to edit this zone.\r\n")
			return nil
		}
		s.oeditSend("Saving all objects in zone.\r\n")
		slog.Info("OLC: oedit zone save",
			"player", s.playerName, "zone", zone.Number)
		if err := saveOeditZone(s.manager.world, zone); err != nil {
			slog.Error("oedit disk save failed",
				"player", s.playerName, "zone", zone.Number, "error", err)
		}
		return nil
	}

	// The duplicate-editor gate runs before zone lookup and authorization,
	// matching do_olc's descriptor scan order. The claim itself is the check,
	// so two sessions racing through cmdOedit cannot both be admitted. Every
	// refusal and failure path below releases the claim.
	if other, ok := s.manager.claimObjEdit(number, s); !ok {
		s.oeditSend(fmt.Sprintf("That object is currently being edited by %s.\r\n", other))
		return nil
	}

	zone, ok := olcZoneForVNum(s.manager.world, number)
	if !ok {
		s.manager.releaseObjEdit(number, s)
		s.oeditSend("Sorry, there is no zone for that number!\r\n")
		return nil
	}
	if !olcAuthorized(s, zone.Number) {
		s.manager.releaseObjEdit(number, s)
		s.oeditSend("You do not have permission to edit this zone.\r\n")
		return nil
	}

	if err := s.startOedit(number, zone); err != nil {
		s.manager.releaseObjEdit(number, s)
		return err
	}
	return nil
}

// startOedit begins an oedit session for vnum, mirroring do_olc's
// SCMD_OLC_OEDIT branch plus oedit_setup_existing / oedit_setup_new.
func (s *Session) startOedit(number int, zone *parser.Zone) error {
	obj, exists := s.manager.world.SnapshotObj(number)
	if !exists {
		obj = newOeditObj(number)
	}

	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	if s.oedit != nil {
		s.finishOeditLocked()
	}

	state := &oeditState{
		obj:        obj,
		number:     number,
		zoneNumber: zone.Number,
		isNew:      !exists,
		mode:       oeditMainMenu,
		affects:    make([]parser.ObjAffect, maxObjAffect),
	}
	for i, affect := range obj.Affects {
		if i >= maxObjAffect {
			break
		}
		state.affects[i] = affect
	}
	state.extraMeta = make([]oeditExtraMeta, len(obj.ExtraDescs))
	for i, extra := range obj.ExtraDescs {
		state.extraMeta[i] = oeditExtraMeta{
			keywordSet:     extra.Keywords != "",
			descriptionSet: extra.Description != "",
		}
	}

	s.oedit = state
	s.setPlayerWritingLocked(true)
	s.oeditShowMenuLocked()
	s.flushOeditOutputLocked()
	if s.player != nil {
		s.oeditActLocked("$n starts using OLC.")
	}
	return nil
}

// newOeditObj builds the working copy for an object with no prototype,
// mirroring oedit_setup_new (src/oedit.c:98-109) layered on clear_object:
// the unfinished strings, no action description, and ITEM_WEAR_TAKE.
func newOeditObj(number int) parser.Obj {
	return parser.Obj{
		VNum:      number,
		Keywords:  "unfinished object",
		ShortDesc: "an unfinished object",
		LongDesc:  "An unfinished object is lying here.",
		WearFlags: [4]int{1, 0, 0, 0}, // ITEM_WEAR_TAKE (bit 0)
	}
}

// finishOeditLocked mirrors cleanup_olc for CON_OEDIT (src/olc.c:447-500): it
// clears the writing flag, broadcasts the stop-editing message, and returns
// the descriptor to playing. C's CLEANUP_ALL and CLEANUP_STRUCTS differ only
// in which heap strings are freed, so a single path is byte-identical here.
func (s *Session) finishOeditLocked() {
	if s.oedit == nil {
		return
	}
	number := s.oedit.number
	// C flushes the pending output buffer before cleanup_olc frees the OLC
	// struct, so a "Saving object to memory." line queued for this turn still
	// reaches the player.
	s.flushOeditOutputLocked()
	s.setPlayerWritingLocked(false)
	if s.player != nil {
		s.oeditActLocked("$n stops using OLC.")
	}
	s.oedit = nil
	s.textEdit = nil
	if s.manager != nil {
		s.manager.releaseObjEdit(number, s)
	}
}

// cancelOedit discards an in-progress oedit session without saving, used on
// linkdead disconnect. It mirrors C's cleanup_olc(d, CLEANUP_ALL) on
// descriptor close.
func (s *Session) cancelOedit() {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	if s.oedit != nil {
		s.finishOeditLocked()
	}
}

// oeditSend accumulates output into the descriptor buffer while an oedit
// session is active, mirroring write_to_output inside the C editor. The buffer
// flushes once per input turn so consecutive C send_to_char calls stay
// contiguous in the byte stream.
func (s *Session) oeditSend(text string) {
	if s.oedit != nil {
		s.oedit.pendingOutput += text
		return
	}
	if err := s.SendMessage(text); err != nil {
		slog.Error("oedit output failed", "player", s.playerName, "error", err)
	}
}

func (s *Session) oeditSendLocked(text string) {
	s.oeditSend(text)
}

// flushOeditOutputLocked emits the buffered C output buffer as one message.
func (s *Session) flushOeditOutputLocked() {
	if s.oedit == nil || s.oedit.pendingOutput == "" {
		return
	}
	text := s.oedit.pendingOutput
	s.oedit.pendingOutput = ""
	if err := s.SendMessage(text); err != nil {
		slog.Error("oedit output failed", "player", s.playerName, "error", err)
	}
}

// oeditActLocked runs a room-audience message with $n substitution, mirroring
// C's act(..., TO_ROOM, ...) around the OLC transitions.
func (s *Session) oeditActLocked(format string) {
	if s.manager == nil || s.manager.world == nil || s.player == nil {
		return
	}
	game.Act(s.manager.world, false, s.player, nil, nil, nil, format, "", game.ToRoom)
}

// oeditCols mirrors get_char_cols for the OLC menus; it is the same
// color-level gate redit and medit use.
func (s *Session) oeditCols() (nrm, grn, cyn, yel string) {
	return s.reditCols()
}

// oeditSprintNBit mirrors C's sprintnbit (src/utils.c:799) for one flag word:
// each set bit renders its table name plus a trailing space, and bits past the
// table's "\n" terminator render "UNDEFINED ". The table is emulated as a
// nil-terminated list with an implicit "\n" at index len(names).
func oeditSprintNBit(bits uint32, names []string) string {
	if bits == 0 {
		return "NOBITS "
	}
	var sb strings.Builder
	nr := 0
	for b := bits; b != 0; b >>= 1 {
		if b&1 != 0 {
			if nr < len(names) {
				sb.WriteString(names[nr])
			} else {
				sb.WriteString("UNDEFINED")
			}
			sb.WriteByte(' ')
		}
		if nr < len(names) {
			nr++
		}
	}
	if sb.Len() == 0 {
		return "NOBITS "
	}
	return sb.String()
}

// oeditSprintBitArray mirrors C's sprintbitarray: concatenate the per-word
// renderings, skipping words that render nothing; an all-empty result renders
// "NOBITS ".
func oeditSprintBitArray(words [4]int, names []string) string {
	var sb strings.Builder
	for _, word := range words {
		rendered := oeditSprintNBit(uint32(word), names) // #nosec G115 -- deliberate wrap mirroring C's (bitvector_t) cast of the int word in sprintbitarray (src/utils.c:836)
		if rendered != "NOBITS " {
			sb.WriteString(rendered)
		}
	}
	if sb.Len() == 0 {
		return "NOBITS "
	}
	return sb.String()
}

// oeditSprintBit mirrors C's sprintbit (singular) over one int bitvector.
func oeditSprintBit(bits int, names []string) string {
	if bits < 0 {
		return "<INVALID BITVECTOR>"
	}
	return oeditSprintNBit(uint32(bits), names) // #nosec G115 -- bits guarded non-negative above; uint32 wrap mirrors C's unsigned bitvector_t parameter (src/structs.h:697)
}

// oeditSprintType mirrors C's sprinttype: index into a name table, or
// "UNDEFINED" past its terminator.
func oeditSprintType(kind int, names []string) string {
	if kind < 0 || kind >= len(names) {
		return "UNDEFINED"
	}
	return names[kind]
}

// oeditTableName indexes a C display table for the two special apply forms
// (mob_races for APPLY_RACE_HATE, affected_bits for APPLY_SPELL). C reads the
// modifier unchecked; an out-of-range modifier is undefined behaviour with no
// reproducible bytes, so the port substitutes a visible marker and records the
// gap in docs/fidelity/depth/oedit.tsv.
func oeditTableName(names []string, index int) string {
	if index < 0 || index >= len(names) {
		return "<UNDEFINED>"
	}
	return names[index]
}

// atofC mirrors C's atof/strtod for the leading numeric prefix: optional
// whitespace, an optional sign, digits with an optional decimal point, and an
// optional decimal exponent. Trailing garbage is ignored; no valid prefix
// yields 0. glibc's hex-float extension ("0x10") is not reproduced.
func atofC(input string) float64 {
	s := strings.TrimLeft(input, " \t\r\n\v\f")
	if s == "" {
		return 0
	}
	i := 0
	if s[0] == '+' || s[0] == '-' {
		i = 1
	}
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	sawDigit := i > start
	if i < len(s) && s[i] == '.' {
		i++
		fracStart := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		sawDigit = sawDigit || i > fracStart
	}
	if !sawDigit {
		return 0
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		k := j
		for k < len(s) && s[k] >= '0' && s[k] <= '9' {
			k++
		}
		if k > j {
			i = k
		}
	}
	value, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0
	}
	return value
}

// oeditRoundFloat is oedit.c's round_float in single precision: the int cast
// truncates toward zero before the multiply, and the multiply is int*float32.
// Doing this in float32 (not float64) is required for byte fidelity.
func oeditRoundFloat(value, precision float32) float32 {
	return float32(int(value/precision+0.5)) * precision
}

// oeditLoadText mirrors the main menu's buf2 rendering for GET_OBJ_LOAD.
func oeditLoadText(load float32) string {
	if load != 0 {
		return fmt.Sprintf("1 in %d", int(float64(100.0)/float64(load)+0.5))
	}
	return "Never"
}

// oeditShowMenuLocked renders oedit_disp_menu (src/oedit.c:929-1001) in its two
// C sends, byte for byte, including get_char_cols colors and field widths.
func (s *Session) oeditShowMenuLocked() {
	state := s.oedit
	if state == nil {
		return
	}
	obj := &state.obj
	nrm, grn, cyn, yel := s.oeditCols()

	actionDesc := "<not set>\r\n"
	if obj.ActionDesc != "" {
		actionDesc = obj.ActionDesc
	}

	var first strings.Builder
	fmt.Fprintf(&first, "\r\n-- Item number : [%s%d%s]\r\n", cyn, state.number, nrm)
	fmt.Fprintf(&first, "%s1%s) Namelist : %s%s\r\n", grn, nrm, yel, obj.Keywords)
	fmt.Fprintf(&first, "%s2%s) S-Desc   : %s%s\r\n", grn, nrm, yel, obj.ShortDesc)
	fmt.Fprintf(&first, "%s3%s) L-Desc   :-\r\n%s%s\r\n", grn, nrm, yel, obj.LongDesc)
	fmt.Fprintf(&first, "%s4%s) A-Desc   :-\r\n%s%s", grn, nrm, yel, actionDesc)
	fmt.Fprintf(&first, "%s5%s) Type        : %s%s\r\n", grn, nrm, cyn,
		oeditSprintType(obj.TypeFlag, oeditItemTypes))
	fmt.Fprintf(&first, "%s6%s) Extra flags : %s%s\r\n", grn, nrm, cyn,
		oeditSprintBitArray(obj.ExtraFlags, oeditExtraBits))
	s.oeditSendLocked(first.String())

	var second strings.Builder
	fmt.Fprintf(&second, "%s7%s) Wear flags  : %s%s\r\n", grn, nrm, cyn,
		oeditSprintBitArray(obj.WearFlags, oeditWearBits))
	fmt.Fprintf(&second, "%s8%s) Encumbrance : %s%d\r\n", grn, nrm, cyn, obj.Weight)
	fmt.Fprintf(&second, "%s9%s) Cost        : %s%d\r\n", grn, nrm, cyn, obj.Cost)
	load := float32(obj.LoadPercent)
	fmt.Fprintf(&second, "%sA%s) Percent Load: %s%s%% (%s)\r\n", grn, nrm, cyn,
		oeditFormatLoad(load), oeditLoadText(load))
	fmt.Fprintf(&second, "%sB%s) Timer       : %s%d\r\n", grn, nrm, cyn, state.timer)
	fmt.Fprintf(&second, "%sD%s) Values      : %s%d %d %d %d\r\n", grn, nrm, cyn,
		obj.Values[0], obj.Values[1], obj.Values[2], obj.Values[3])
	fmt.Fprintf(&second, "%sE%s) Applies menu\r\n", grn, nrm)
	fmt.Fprintf(&second, "%sF%s) Extra descriptions menu\r\n", grn, nrm)
	fmt.Fprintf(&second, "%sS%s) Scripts menu\r\n", grn, nrm)
	fmt.Fprintf(&second, "%sQ%s) Quit\r\n", grn, nrm)
	second.WriteString("Enter choice : ")
	s.oeditSendLocked(second.String())
	state.mode = oeditMainMenu
}

// oeditShowTypeMenuLocked mirrors oedit_disp_type_menu (src/oedit.c:861-875).
func (s *Session) oeditShowTypeMenuLocked() {
	nrm, grn, _, _ := s.oeditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	columns := 0
	for i, name := range oeditItemTypes {
		fmt.Fprintf(&sb, "%s%2d%s) %-20.20s ", grn, i, nrm, name)
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	sb.WriteString("\r\nEnter object type : ")
	s.oeditSendLocked(sb.String())
	state := s.oedit
	if state != nil {
		state.mode = oeditType
	}
}

// oeditShowExtraMenuLocked mirrors oedit_disp_extra_menu (src/oedit.c:879-899):
// flags are numbered from 1, and an invalid selection only redisplays this
// menu with no error message (:1191-1197).
func (s *Session) oeditShowExtraMenuLocked() {
	state := s.oedit
	nrm, grn, cyn, _ := s.oeditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	columns := 0
	for i, name := range oeditExtraBits {
		fmt.Fprintf(&sb, "%s%2d%s) %-20.20s ", grn, i+1, nrm, name)
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&sb, "\r\nObject flags: %s%s%s\r\nEnter object extra flag (0 to quit) : ",
		cyn, oeditSprintBitArray(state.obj.ExtraFlags, oeditExtraBits), nrm)
	s.oeditSendLocked(sb.String())
}

// oeditShowWearMenuLocked mirrors oedit_disp_wear_menu (src/oedit.c:904-925).
func (s *Session) oeditShowWearMenuLocked() {
	state := s.oedit
	nrm, grn, cyn, _ := s.oeditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	columns := 0
	for i, name := range oeditWearBits {
		fmt.Fprintf(&sb, "%s%2d%s) %-20.20s ", grn, i+1, nrm, name)
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&sb, "\r\nWear flags: %s%s%s\r\nEnter wear flag, 0 to quit : ",
		cyn, oeditSprintBitArray(state.obj.WearFlags, oeditWearBits), nrm)
	s.oeditSendLocked(sb.String())
}

// oeditShowContainerFlagsLocked mirrors oedit_disp_container_flags_menu
// (src/oedit.c:458-473) in its two C sends.
func (s *Session) oeditShowContainerFlagsLocked() {
	state := s.oedit
	nrm, grn, cyn, _ := s.oeditCols()
	s.oeditSendLocked("\r\n")
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s1%s) CLOSEABLE\r\n", grn, nrm)
	fmt.Fprintf(&sb, "%s2%s) PICKPROOF\r\n", grn, nrm)
	fmt.Fprintf(&sb, "%s3%s) CLOSED\r\n", grn, nrm)
	fmt.Fprintf(&sb, "%s4%s) LOCKED\r\n", grn, nrm)
	fmt.Fprintf(&sb, "Container flags: %s%s%s\r\n", cyn,
		oeditSprintBit(state.obj.Values[1], oeditContainerBits), nrm)
	sb.WriteString("Enter flag, 0 to quit : ")
	s.oeditSendLocked(sb.String())
}

// oeditFormatLoad mirrors C's "%.2f" printf, where the float argument is
// promoted to double before formatting.
func oeditFormatLoad(load float32) string {
	return fmt.Sprintf("%.2f", float64(load))
}

// oeditShowScriptMenuLocked mirrors oedit_disp_script_menu (src/oedit.c:475-502).
// C resolves the script through obj_index[rnum], so a never-saved object
// (GET_OBJ_RNUM == NOTHING) is refused and falls back to the main menu.
func (s *Session) oeditShowScriptMenuLocked() {
	state := s.oedit
	if state.isNew {
		s.oeditSendLocked("\r\nCannot assign a script until the object is saved at least once.\r\n")
		s.oeditShowMenuLocked()
		return
	}
	name := "None"
	flags := 0
	if live, ok := s.manager.world.SnapshotObj(state.number); ok {
		if live.ScriptName != "" {
			name = live.ScriptName
		}
		flags = live.LuaFunctions
	}
	nrm, grn, _, yel := s.oeditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	fmt.Fprintf(&sb, "%s1%s) Name: %s%s\r\n", grn, nrm, yel, name)
	fmt.Fprintf(&sb, "%s2%s) Script Flags: %s%s%s\r\n", grn, nrm, yel,
		oeditSprintBit(flags, oeditScriptFlagNames), nrm)
	sb.WriteString("Enter choice (0 to quit) : ")
	s.oeditSendLocked(sb.String())
	state.mode = oeditScriptMenu
}

// oeditShowScriptFlagsLocked mirrors oedit_disp_script_flags
// (src/oedit.c:504-526): a real clear-screen escape, the two-column flag list
// over oscript_bits, and the current-flags line.
func (s *Session) oeditShowScriptFlagsLocked() {
	state := s.oedit
	nrm, grn, cyn, _ := s.oeditCols()
	flags := 0
	if !state.isNew {
		if live, ok := s.manager.world.SnapshotObj(state.number); ok {
			flags = live.LuaFunctions
		}
	}
	var sb strings.Builder
	sb.WriteString("\x1b[H\x1b[J")
	columns := 0
	for i, name := range oeditScriptFlagNames {
		fmt.Fprintf(&sb, "%s%2d%s) %-20.20s  ", grn, i+1, nrm, name)
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&sb, "\r\nCurrent flags   : %s%s%s\r\n", cyn,
		oeditSprintBit(flags, oeditScriptFlagNames), nrm)
	sb.WriteString("Enter script flags (0 to quit) : ")
	s.oeditSendLocked(sb.String())
	state.mode = oeditScriptFlags
}

// oeditShowExtraDescMenuLocked mirrors oedit_disp_extradesc_menu
// (src/oedit.c:529-556). Note this is NOT redit's extra-desc menu: it carries
// the "Extra desc menu" header, its own "0) Quit" line, and the
// "<Not set>\r\n" string that already contains the newline.
func (s *Session) oeditShowExtraDescMenuLocked() {
	state := s.oedit
	if state.currentExtra >= len(state.obj.ExtraDescs) {
		state.currentExtra = 0
	}
	extra := state.obj.ExtraDescs[state.currentExtra]
	meta := state.extraMeta[state.currentExtra]

	keyword := "<NONE>"
	if meta.keywordSet {
		keyword = extra.Keywords
	}
	description := "<NONE>"
	if meta.descriptionSet {
		description = editorCRLF(extra.Description)
	}
	gotoNext := "<Not set>\r\n"
	if state.currentExtra+1 < len(state.obj.ExtraDescs) {
		gotoNext = "Set."
	}

	nrm, grn, _, yel := s.oeditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	sb.WriteString("Extra desc menu\r\n")
	fmt.Fprintf(&sb, "%s1%s) Keyword: %s%s\r\n", grn, nrm, yel, keyword)
	fmt.Fprintf(&sb, "%s2%s) Description:\r\n%s%s\r\n", grn, nrm, yel, description)
	fmt.Fprintf(&sb, "%s3%s) Goto next description: %s\r\n", grn, nrm, gotoNext)
	fmt.Fprintf(&sb, "%s0%s) Quit\r\n", grn, nrm)
	sb.WriteString("Enter choice : ")
	s.oeditSendLocked(sb.String())
	state.mode = oeditExtraDescMenu
}

// oeditShowPromptApplyMenuLocked mirrors oedit_disp_prompt_apply_menu
// (src/oedit.c:559-593): per-slot rendering with APPLY_RACE_HATE and
// APPLY_SPELL special forms, "%+d" for plain modifiers, and "None." for empty
// slots. C sends a blank line before the list and the trailing prompt
// separately.
func (s *Session) oeditShowPromptApplyMenuLocked() {
	state := s.oedit
	nrm, grn, _, _ := s.oeditCols()
	s.oeditSendLocked("\r\n")
	var sb strings.Builder
	for i := 0; i < maxObjAffect; i++ {
		affect := state.affects[i]
		if affect.Location != 0 {
			applyName := oeditSprintType(affect.Location, oeditApplyTypes)
			switch affect.Location {
			case applyRaceHate:
				fmt.Fprintf(&sb, " %s%d%s) %s to %s\r\n", grn, i+1, nrm,
					oeditTableName(meditRaceNames, affect.Modifier), applyName)
			case applySpell:
				fmt.Fprintf(&sb, " %s%d%s) %s to %s\r\n", grn, i+1, nrm,
					oeditTableName(oeditAffectBits, affect.Modifier), applyName)
			default:
				fmt.Fprintf(&sb, " %s%d%s) %+d to %s\r\n", grn, i+1, nrm,
					affect.Modifier, applyName)
			}
		} else {
			fmt.Fprintf(&sb, " %s%d%s) None.\r\n", grn, i+1, nrm)
		}
	}
	s.oeditSendLocked(sb.String())
	s.oeditSendLocked("\r\nEnter affection to modify (0 to quit) : ")
	state.mode = oeditPromptApply
}

// oeditShowLiquidTypeLocked mirrors oedit_liquid_type (src/oedit.c:596-615):
// two columns, drink names in yel, numbered from 0.
func (s *Session) oeditShowLiquidTypeLocked() {
	nrm, grn, _, yel := s.oeditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	columns := 0
	for i, name := range oeditDrinks {
		fmt.Fprintf(&sb, " %s%2d%s) %s%-20.20s ", grn, i, nrm, yel, name)
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&sb, "\r\n%sEnter drink type : ", nrm)
	s.oeditSendLocked(sb.String())
	s.oedit.mode = oeditValue3
}

// oeditShowApplyMenuLocked mirrors oedit_disp_apply_menu (src/oedit.c:618-635):
// two columns over apply_types, numbered from 0, then the apply-type prompt.
func (s *Session) oeditShowApplyMenuLocked() {
	nrm, grn, _, _ := s.oeditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	columns := 0
	for i, name := range oeditApplyTypes {
		fmt.Fprintf(&sb, "%s%2d%s) %-20.20s ", grn, i, nrm, name)
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	sb.WriteString("\r\nEnter apply type (0 is no apply) : ")
	s.oeditSendLocked(sb.String())
	s.oedit.mode = oeditApply
}

// oeditShowSpellMenuLocked mirrors oedit_disp_spell_menu (src/oedit.c:638-655):
// the permanent-spell-effect variant, two columns over affected_bits with no
// leading blank line before the prompt.
func (s *Session) oeditShowSpellMenuLocked() {
	nrm, grn, _, _ := s.oeditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	columns := 0
	for i := 0; i < numAffFlags; i++ {
		fmt.Fprintf(&sb, "%s%2d%s) %-20.20s ", grn, i, nrm, oeditAffectBits[i])
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	sb.WriteString("Enter perm spell effect : ")
	s.oeditSendLocked(sb.String())
	s.oedit.mode = oeditApplyMod
}

// oeditShowRaceMenuLocked mirrors oedit_disp_race_menu (src/oedit.c:657-674).
func (s *Session) oeditShowRaceMenuLocked() {
	nrm, grn, _, _ := s.oeditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	columns := 0
	for i, name := range meditRaceNames {
		fmt.Fprintf(&sb, "%s%2d%s) %-20.20s ", grn, i, nrm, name)
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	sb.WriteString("Enter mob race : ")
	s.oeditSendLocked(sb.String())
	s.oedit.mode = oeditApplyMod
}

// oeditShowWeaponMenuLocked mirrors oedit_disp_weapon_menu (src/oedit.c:678-694).
func (s *Session) oeditShowWeaponMenuLocked() {
	nrm, grn, _, _ := s.oeditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	columns := 0
	for i, name := range meditAttackNames {
		fmt.Fprintf(&sb, "%s%2d%s) %-20.20s ", grn, i, nrm, name)
		columns++
		if columns%2 == 0 {
			sb.WriteString("\r\n")
		}
	}
	sb.WriteString("\r\nEnter weapon type : ")
	s.oeditSendLocked(sb.String())
}

// oeditShowSpellsMenuLocked mirrors oedit_disp_spells_menu (src/oedit.c:697-714):
// THREE columns (% 3), spell names in yel, numbered from 0.
func (s *Session) oeditShowSpellsMenuLocked() {
	nrm, grn, _, yel := s.oeditCols()
	var sb strings.Builder
	sb.WriteString("\r\n")
	columns := 0
	for i := 0; i < numSpells; i++ {
		fmt.Fprintf(&sb, "%s%2d%s) %s%-20.20s ", grn, i, nrm, yel, oeditSpellNames[i])
		columns++
		if columns%3 == 0 {
			sb.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&sb, "\r\n%sEnter spell choice (0 for none) : ", nrm)
	s.oeditSendLocked(sb.String())
}

// oeditShowVal1MenuLocked mirrors oedit_disp_val1_menu (src/oedit.c:717-759):
// type-driven cascade where LIGHT skips to val3 and WEAPON/MISSILE/FIREWEAPON
// skip val0.
func (s *Session) oeditShowVal1MenuLocked() {
	state := s.oedit
	state.mode = oeditValue1
	switch state.obj.TypeFlag {
	case itemLight:
		// C: "values 0 and 1 are unused.. jump to 2". Note the mode stays
		// OEDIT_VALUE_3 (set by the callee at :801), not OEDIT_VALUE_1.
		s.oeditShowVal3MenuLocked()
	case itemScroll, itemWand, itemStaff, itemPotion:
		s.oeditSendLocked("Spell level : ")
	case itemMissile, itemFireweapon, itemWeapon:
		s.oeditShowVal2MenuLocked()
	case itemArmor:
		s.oeditSendLocked("Apply to AC : ")
	case itemContainer:
		s.oeditSendLocked("Max weight to contain : ")
	case itemDrinkcon, itemFountain:
		s.oeditSendLocked("Max drink units : ")
	case itemFood:
		s.oeditSendLocked("Hours to fill stomach : ")
	case itemMoney:
		s.oeditSendLocked("Number of gold coins : ")
	case itemNote:
		// C: case ITEM_NOTE falls into default (language is unused).
		s.oeditShowMenuLocked()
	default:
		s.oeditShowMenuLocked()
	}
}

// oeditShowVal2MenuLocked mirrors oedit_disp_val2_menu (src/oedit.c:761-795).
func (s *Session) oeditShowVal2MenuLocked() {
	state := s.oedit
	state.mode = oeditValue2
	switch state.obj.TypeFlag {
	case itemScroll, itemPotion:
		s.oeditShowSpellsMenuLocked()
	case itemWand, itemStaff:
		s.oeditSendLocked("Max number of charges : ")
	case itemMissile, itemFireweapon, itemWeapon:
		s.oeditSendLocked("Number of damage dice : ")
	case itemFood:
		// C: "values 2 and 3 are unused, jump to 4. how odd".
		s.oeditShowVal4MenuLocked()
	case itemContainer:
		s.oeditShowContainerFlagsLocked()
	case itemDrinkcon, itemFountain:
		s.oeditSendLocked("Initial drink units : ")
	default:
		s.oeditShowMenuLocked()
	}
}

// oeditShowVal3MenuLocked mirrors oedit_disp_val3_menu (src/oedit.c:797-832).
func (s *Session) oeditShowVal3MenuLocked() {
	state := s.oedit
	state.mode = oeditValue3
	switch state.obj.TypeFlag {
	case itemLight:
		s.oeditSendLocked("Number of hours (0 = burnt, -1 is infinite) : ")
	case itemScroll, itemPotion:
		s.oeditShowSpellsMenuLocked()
	case itemWand, itemStaff:
		s.oeditSendLocked("Number of charges remaining : ")
	case itemMissile, itemFireweapon, itemWeapon:
		s.oeditSendLocked("Size of damage dice : ")
	case itemContainer:
		s.oeditSendLocked("Vnum of key to open container (-1 for no key) : ")
	case itemDrinkcon, itemFountain:
		s.oeditShowLiquidTypeLocked()
	default:
		s.oeditShowMenuLocked()
	}
}

// oeditShowVal4MenuLocked mirrors oedit_disp_val4_menu (src/oedit.c:834-858).
func (s *Session) oeditShowVal4MenuLocked() {
	state := s.oedit
	state.mode = oeditValue4
	switch state.obj.TypeFlag {
	case itemScroll, itemPotion, itemWand, itemStaff:
		s.oeditShowSpellsMenuLocked()
	case itemWeapon:
		s.oeditShowWeaponMenuLocked()
	case itemDrinkcon, itemFountain, itemFood:
		s.oeditSendLocked("Poisoned (0 = not poison) : ")
	default:
		s.oeditShowMenuLocked()
	}
}

// oeditToggleExtraFlag XORs one bit of the object's extra flag word 0,
// mirroring TOGGLE_BIT_AR over the 4-int array (NUM_ITEM_FLAGS is 29, so only
// word 0 is reachable).
func oeditToggleExtraFlag(obj *parser.Obj, bit int) {
	if bit < 0 || bit >= numItemFlags {
		return
	}
	obj.ExtraFlags[bit/32] ^= 1 << uint(bit%32)
}

// oeditToggleWearFlag XORs one bit of the object's wear flag word 0.
func oeditToggleWearFlag(obj *parser.Obj, bit int) {
	if bit < 0 || bit >= numItemWears {
		return
	}
	obj.WearFlags[bit/32] ^= 1 << uint(bit%32)
}

// strUDup mirrors C's str_udup (src/olc.c:520): an empty or NULL text becomes
// the literal "undefined"; there is no trimming.
func strUDup(text string) string {
	if text == "" {
		return "undefined"
	}
	return text
}

// parseOeditMainMenuLocked ports the OEDIT_MAIN_MENU switch (src/oedit.c:1049-1160).
// The choice is the first input byte (C switches on *arg), and the C case ends
// with a bare return, so this menu never falls through to the trailing
// has-changed redisplay.
func (s *Session) parseOeditMainMenuLocked(arg string) {
	state := s.oedit
	switch firstByte(arg) {
	case 'q', 'Q':
		if state.olcVal != 0 {
			state.mode = oeditConfirmSaveString
			s.oeditSendLocked("Do you wish to save this object internally? : ")
		} else {
			s.finishOeditLocked()
		}
	case '1':
		state.mode = oeditEditNamelist
		s.oeditSendLocked("Enter namelist : ")
	case '2':
		state.mode = oeditShortDesc
		s.oeditSendLocked("Enter short desc : ")
	case '3':
		state.mode = oeditLongDesc
		s.oeditSendLocked("Enter long desc :-\r\n| ")
	case '4':
		s.startOeditActDescEditorLocked()
	case '5':
		s.oeditShowTypeMenuLocked()
	case '6':
		state.mode = oeditExtras
		s.oeditShowExtraMenuLocked()
	case '7':
		state.mode = oeditWear
		s.oeditShowWearMenuLocked()
	case '8':
		state.mode = oeditWeight
		s.oeditSendLocked("Enter weight : ")
	case '9':
		// C: "Enter cost (10000 max): " — there is no maximum; OEDIT_COST is
		// a bare atoi (src/oedit.c:1105, :1243-1245). Keep both.
		state.mode = oeditCost
		s.oeditSendLocked("Enter cost (10000 max): ")
	case 'a', 'A':
		state.mode = oeditCostPerDay
		s.oeditSendLocked("Enter percent chance\r\nof the item loading : ")
	case 'b', 'B':
		state.mode = oeditTimer
		s.oeditSendLocked("Enter timer : ")
	case 'c', 'C':
		// C's object-level flag prompt is compiled out; it just redraws.
		s.oeditShowMenuLocked()
	case 'd', 'D':
		state.obj.Values = [4]int{}
		s.oeditShowVal1MenuLocked()
	case 'e', 'E':
		s.oeditShowPromptApplyMenuLocked()
	case 'f', 'F':
		if len(state.obj.ExtraDescs) == 0 {
			applyOLC(olc.Operation{Kind: olc.OpAddObjExtraDescription, Obj: &state.obj, Index: -1})
			state.extraMeta = append(state.extraMeta, oeditExtraMeta{})
		}
		state.currentExtra = 0
		s.oeditShowExtraDescMenuLocked()
	case 's', 'S':
		state.mode = oeditScriptMenu
		s.oeditShowScriptMenuLocked()
	default:
		s.oeditShowMenuLocked()
	}
}

// parseOeditLocked is the Go port of oedit_parse (src/oedit.c:1016-1553). The
// input has already been left-trimmed, matching C's skip_spaces.
func (s *Session) parseOeditLocked(arg string) {
	state := s.oedit
	obj := &state.obj

	switch state.mode {
	case oeditConfirmSaveString:
		switch firstByte(arg) {
		case 'y', 'Y':
			s.oeditSendLocked("Saving object to memory.\r\n")
			s.saveOeditInternallyLocked()
			slog.Info("OLC: oedit edit",
				"player", s.playerName, "obj", state.number)
			s.finishOeditLocked()
			return
		case 'n', 'N':
			s.finishOeditLocked()
			return
		default:
			s.oeditSendLocked("Invalid choice!\r\n")
			// C's retry prompt drops the ": " the first prompt carries
			// (src/oedit.c:1043-1045).
			s.oeditSendLocked("Do you wish to save this object internally?\r\n")
			return
		}

	case oeditMainMenu:
		s.parseOeditMainMenuLocked(arg)
		return

	case oeditEditNamelist:
		applyOLC(olc.Operation{Kind: olc.OpSetObjKeywords, Obj: obj, Text: arg})
	case oeditShortDesc:
		applyOLC(olc.Operation{Kind: olc.OpSetObjShortDescription, Obj: obj, Text: arg})
	case oeditLongDesc:
		applyOLC(olc.Operation{Kind: olc.OpSetObjLongDescription, Obj: obj, Text: arg})
	case oeditActDesc:
		// C: "We should never get here." The string editor owns the line.
		slog.Error("oedit reached ACTDESC case", "player", s.playerName)
		s.finishOeditLocked()
		return
	case oeditType:
		number := atoiC(arg)
		if number < 1 || number >= numItemTypes {
			// C: type 0 is not selectable even though the menu lists it.
			s.oeditSendLocked("Invalid choice, try again : ")
			return
		}
		obj.TypeFlag = number
	case oeditExtras:
		number := atoiC(arg)
		if number < 0 || number > numItemFlags {
			// C's extras error path only redisplays the menu — no message
			// (src/oedit.c:1191-1197), unlike the wear path.
			s.oeditShowExtraMenuLocked()
			return
		}
		if number == 0 {
			break
		}
		oeditToggleExtraFlag(obj, number-1)
		s.oeditShowExtraMenuLocked()
		return
	case oeditWear:
		number := atoiC(arg)
		if number < 0 || number > numItemWears {
			s.oeditSendLocked("That's not a valid choice!\r\n")
			s.oeditShowWearMenuLocked()
			return
		}
		if number == 0 {
			break
		}
		oeditToggleWearFlag(obj, number-1)
		s.oeditShowWearMenuLocked()
		return
	case oeditWeight:
		obj.Weight = atoiC(arg)
	case oeditCost:
		obj.Cost = atoiC(arg)
	case oeditCostPerDay:
		load := oeditRoundFloat(float32(atofC(arg)), 0.01)
		if load < 0.0 {
			load = 0.0
		} else if load > 100.0 {
			load = 100.0
		}
		obj.LoadPercent = float64(load)
	case oeditTimer:
		state.timer = atoiC(arg)
	case oeditLevel:
		// C's object level flags are compiled out.
	case oeditValue1:
		applyOLC(olc.Operation{Kind: olc.OpSetObjValue1, Obj: obj, Value: atoiC(arg)})
		state.olcVal = 1
		s.oeditShowVal2MenuLocked()
		return
	case oeditValue2:
		number := atoiC(arg)
		switch obj.TypeFlag {
		case itemScroll, itemPotion:
			if number < 0 || number >= numSpells {
				s.oeditShowVal2MenuLocked()
			} else {
				applyOLC(olc.Operation{Kind: olc.OpSetObjValue2, Obj: obj, Value: number})
				s.oeditShowVal3MenuLocked()
			}
		case itemContainer:
			// C re-reads arg (src/oedit.c:1299) and treats 1..4 as toggles.
			number = atoiC(arg)
			switch {
			case number < 0 || number > 4:
				s.oeditShowContainerFlagsLocked()
			case number != 0:
				applyOLC(olc.Operation{Kind: olc.OpToggleObjContainerFlag, Obj: obj, Value: number - 1})
				s.oeditShowVal2MenuLocked()
			default:
				s.oeditShowVal3MenuLocked()
			}
		default:
			applyOLC(olc.Operation{Kind: olc.OpSetObjValue2, Obj: obj, Value: number})
			s.oeditShowVal3MenuLocked()
		}
		return
	case oeditValue3:
		number := atoiC(arg)
		applyOLC(olc.Operation{Kind: olc.OpSetObjValue3, Obj: obj, Value: number})
		s.oeditShowVal4MenuLocked()
		return
	case oeditValue4:
		number := atoiC(arg)
		applyOLC(olc.Operation{Kind: olc.OpSetObjValue4, Obj: obj, Value: number})
	case oeditPromptApply:
		number := atoiC(arg)
		if number == 0 {
			break
		}
		if number < 0 || number > maxObjAffect {
			s.oeditShowPromptApplyMenuLocked()
			return
		}
		state.olcVal = number - 1
		s.oeditShowApplyMenuLocked()
		return
	case oeditApply:
		number := atoiC(arg)
		switch {
		case number == 0:
			applyOLC(olc.Operation{
				Kind:    olc.OpSetObjAffect,
				Affects: &state.affects,
				Index:   state.olcVal,
				Affect:  &parser.ObjAffect{},
			})
			s.oeditShowPromptApplyMenuLocked()
		case number < 0 || number >= numApplies:
			s.oeditShowApplyMenuLocked()
		case number == applySpell:
			affect := state.affects[state.olcVal]
			affect.Location = number
			applyOLC(olc.Operation{Kind: olc.OpSetObjAffect, Affects: &state.affects, Index: state.olcVal, Affect: &affect})
			s.oeditShowSpellMenuLocked()
		case number == applyRaceHate:
			affect := state.affects[state.olcVal]
			affect.Location = number
			applyOLC(olc.Operation{Kind: olc.OpSetObjAffect, Affects: &state.affects, Index: state.olcVal, Affect: &affect})
			s.oeditShowRaceMenuLocked()
		default:
			affect := state.affects[state.olcVal]
			affect.Location = number
			applyOLC(olc.Operation{Kind: olc.OpSetObjAffect, Affects: &state.affects, Index: state.olcVal, Affect: &affect})
			s.oeditSendLocked("Modifier : ")
			state.mode = oeditApplyMod
		}
		return
	case oeditApplyMod:
		affect := state.affects[state.olcVal]
		affect.Modifier = atoiC(arg)
		applyOLC(olc.Operation{Kind: olc.OpSetObjAffect, Affects: &state.affects, Index: state.olcVal, Affect: &affect})
		s.oeditShowPromptApplyMenuLocked()
		return
	case oeditScriptMenu:
		switch atoiC(arg) {
		case 0:
			break
		case 1:
			state.mode = oeditScriptName
			s.oeditSendLocked("Enter script name: ")
			return
		case 2:
			s.oeditShowScriptFlagsLocked()
			return
		default:
			break
		}
	case oeditScriptName:
		// The script lives in the live object index (C's obj_index[rnum]).
		s.setOeditScriptNameLocked(arg)
		s.oeditShowScriptMenuLocked()
		return
	case oeditScriptFlags:
		if i := atoiC(arg); i == 0 {
			s.oeditShowScriptMenuLocked()
			return
		} else if i > 0 && i <= numOScriptFlags {
			s.toggleOeditScriptFlagLocked(i - 1)
		}
		s.oeditShowScriptFlagsLocked()
		return
	case oeditExtraDescKey:
		if state.currentExtra < len(obj.ExtraDescs) {
			applyOLC(olc.Operation{
				Kind:  olc.OpSetObjExtraKeywords,
				Obj:   obj,
				Index: state.currentExtra,
				Text:  strUDup(arg),
			})
			state.extraMeta[state.currentExtra].keywordSet = true
		}
		s.oeditShowExtraDescMenuLocked()
		return
	case oeditExtraDescMenu:
		switch atoiC(arg) {
		case 0:
			// If something was left out, unlink this node and everything
			// after it (C sets the pointing pointer to NULL).
			if state.currentExtra < len(obj.ExtraDescs) {
				meta := state.extraMeta[state.currentExtra]
				if !meta.keywordSet || !meta.descriptionSet {
					for len(obj.ExtraDescs) > state.currentExtra {
						applyOLC(olc.Operation{Kind: olc.OpRemoveObjExtraDescription, Obj: obj, Index: state.currentExtra})
					}
					state.extraMeta = state.extraMeta[:state.currentExtra]
				}
			}
		case 1:
			state.mode = oeditExtraDescKey
			s.oeditSendLocked("Enter keywords, separated by spaces :-\r\n| ")
			return
		case 2:
			s.startOeditExtraDescEditorLocked()
			return
		case 3:
			// Only advance if this node is complete (src/oedit.c:1522-1538).
			if state.currentExtra < len(obj.ExtraDescs) {
				meta := state.extraMeta[state.currentExtra]
				if meta.keywordSet && meta.descriptionSet {
					if state.currentExtra+1 < len(obj.ExtraDescs) {
						state.currentExtra++
					} else {
						applyOLC(olc.Operation{Kind: olc.OpAddObjExtraDescription, Obj: obj, Index: -1})
						state.extraMeta = append(state.extraMeta, oeditExtraMeta{})
						state.currentExtra++
					}
				}
			}
			s.oeditShowExtraDescMenuLocked()
			return
		default:
			s.oeditShowExtraDescMenuLocked()
			return
		}
	default:
		slog.Error("oedit reached default case", "player", s.playerName)
	}

	// C: "If we get here, we have changed something."
	state.olcVal = 1
	s.oeditShowMenuLocked()
}

// handleOeditInput routes one raw input line to the active oedit session,
// mirroring the CON_OEDIT case in interpreter.c (which calls oedit_parse after
// skip_spaces). It returns true when the line was consumed by the editor. Like
// the CON_REDIT route it also owns the improved string editor's lines while
// one is open, so the buffered output always flushes in the same turn.
func (s *Session) handleOeditInput(line string) bool {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	if s.oedit == nil {
		return false
	}
	line = strings.TrimLeft(line, " \t")
	if s.textEdit != nil {
		s.handleTextEditInputLocked(line)
		s.flushOeditOutputLocked()
		return true
	}
	s.parseOeditLocked(line)
	s.flushOeditOutputLocked()
	return true
}

// IsOeditEditing reports whether an oedit (CON_OEDIT) session owns this
// descriptor, including while its string editor is active. Exported for the
// telnet listener's input dispatch: like CON_REDIT and CON_MEDIT, CON_OEDIT
// owns every complete input line including bare <ENTER>.
func (s *Session) IsOeditEditing() bool { return s.isOeditEditing() }

// isOeditEditing reports whether an oedit session owns this descriptor.
func (s *Session) isOeditEditing() bool {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	return s.oedit != nil
}

// isObjEditing reports whether an oedit session owns this descriptor. It
// mirrors SendPrompt's redit and medit suppression: C's make_prompt writes
// "] " while d->str is set and no ordinary playing prompt while an OLC menu
// owns the input.
func (s *Session) isObjEditing() bool {
	return s.isOeditEditing()
}

// setOeditScriptNameLocked writes the script name to the LIVE prototype,
// mirroring C's obj_index[rnum].script indirection.
func (s *Session) setOeditScriptNameLocked(name string) {
	state := s.oedit
	if live, ok := s.manager.world.SnapshotObj(state.number); ok {
		live.ScriptName = name
		s.manager.world.SetObjScript(state.number, live.ScriptName, live.LuaFunctions)
	}
}

// toggleOeditScriptFlagLocked flips one script flag bit on the LIVE prototype,
// mirroring C's TOGGLE_BIT on obj_index[rnum].script->lua_functions.
func (s *Session) toggleOeditScriptFlagLocked(bit int) {
	state := s.oedit
	if live, ok := s.manager.world.SnapshotObj(state.number); ok {
		live.LuaFunctions ^= 1 << uint(bit)
		s.manager.world.SetObjScript(state.number, live.ScriptName, live.LuaFunctions)
	}
}

// startOeditActDescEditorLocked opens the improved string editor for the
// object's action description, mirroring oedit_parse's '4' choice
// (src/oedit.c:1076-1086). C sends the editor help, the prompt, and the
// existing description, then sets OLC_VAL immediately (even if the user later
// aborts).
func (s *Session) startOeditActDescEditorLocked() {
	state := s.oedit
	state.mode = oeditActDesc
	state.olcVal = 1

	// The improved string editor works on CRLF text; the parser stores
	// descriptions with LF endings, so normalize on the way in and back.
	initial := editorCRLF(state.obj.ActionDesc)
	s.textEdit = &textEditState{
		field:      textEditField{name: "oedit-actdesc", maxBytes: olc.MaxMessage},
		original:   initial,
		buffer:     initial,
		roomEditor: true,
		onComplete: func(action textEditAction, buffer, original string) {
			// Called with textEditMu held (handleTextEditInputLocked).
			if s.oedit == nil {
				return
			}
			if action == textEditSave {
				applyOLC(olc.Operation{
					Kind: olc.OpSetObjActionDescription,
					Obj:  &s.oedit.obj,
					Text: editorToRoomText(buffer),
				})
			}
			// Abort intentionally leaves the working copy untouched, as C's
			// string_add restored d->backstr before oedit_string_cleanup.
			s.oedit.mode = oeditMainMenu
			s.oeditShowMenuLocked()
		},
	}
	s.oeditSendLocked("Instructions: /s or @ to save, /h for more options.\r\n" +
		"Enter action description:\r\n\r\n" + initial)
}

// startOeditExtraDescEditorLocked opens the improved string editor for the
// current extra description, mirroring oedit_parse's '2' choice inside
// OEDIT_EXTRADESC_MENU (src/oedit.c:1510-1520).
func (s *Session) startOeditExtraDescEditorLocked() {
	state := s.oedit
	state.mode = oeditExtraDescDescription
	state.olcVal = 1

	initial := ""
	if state.currentExtra < len(state.obj.ExtraDescs) && state.extraMeta[state.currentExtra].descriptionSet {
		initial = editorCRLF(state.obj.ExtraDescs[state.currentExtra].Description)
	}
	s.textEdit = &textEditState{
		field:      textEditField{name: "oedit-extradesc", maxBytes: olc.MaxExtraDesc},
		original:   initial,
		buffer:     initial,
		roomEditor: true,
		onComplete: func(action textEditAction, buffer, original string) {
			// Called with textEditMu held (handleTextEditInputLocked).
			if s.oedit == nil {
				return
			}
			if action == textEditSave && s.oedit.currentExtra < len(s.oedit.obj.ExtraDescs) {
				applyOLC(olc.Operation{
					Kind:  olc.OpSetObjExtraDescription,
					Extra: &s.oedit.obj.ExtraDescs[s.oedit.currentExtra],
					Text:  editorToRoomText(buffer),
				})
				s.oedit.extraMeta[s.oedit.currentExtra].descriptionSet = true
			}
			s.oedit.mode = oeditExtraDescMenu
			s.oeditShowExtraDescMenuLocked()
		},
	}
	s.oeditSendLocked("Instructions: /s or @ to save, /h for more options.\r\n" +
		"Enter the extra description:\r\n\r\n" + initial)
}

// saveOeditInternallyLocked ports oedit_save_internally (src/oedit.c:163-335):
// the working copy is committed to the world (inserting or replacing the
// prototype) and every live instance of the vnum is swapped onto the full
// edited struct while its runtime placement fields survive. C performs no disk
// write here; the .obj file is written by the separate save command or
// saveall. C's rnum shifting of zone commands, boards, and shop products has
// no Go analogue because the Go world keys those by VNUM (see CommitEditedObj).
func (s *Session) saveOeditInternallyLocked() {
	state := s.oedit
	obj := state.obj
	obj.VNum = state.number

	// Compact non-zero affect slots back into the parser's slice form. C's
	// disk writer skips location 0 the same way (:438-443).
	obj.Affects = obj.Affects[:0]
	for i := 0; i < maxObjAffect; i++ {
		if state.affects[i].Location == 0 {
			continue
		}
		obj.Affects = append(obj.Affects, state.affects[i])
	}

	// C's script fields live in obj_index[], not obj_proto[], so they are
	// untouched by the working-copy commit; preserve the live values.
	if live, ok := s.manager.world.SnapshotObj(state.number); ok {
		obj.ScriptName = live.ScriptName
		obj.LuaFunctions = live.LuaFunctions
	}

	// Hold the zone save lock across the world commit and the dirty-marker
	// set so the pair is atomic against a concurrent saveOeditZone: without
	// this, a commit landing between the save's snapshot and its marker
	// cleanup would be cleared from the save list without being persisted.
	saveMu := zoneSaveLock(state.zoneNumber)
	saveMu.Lock()
	s.manager.world.CommitEditedObj(obj)
	s.manager.world.RefreshLiveObjInstances(state.number, obj)
	markOLCDirty(olcKindObject, state.zoneNumber)
	saveMu.Unlock()
}

// saveOeditZone writes one zone's .obj file from the world prototypes,
// mirroring oedit_save_to_disk (src/oedit.c:338-451). Only objects whose VNUM
// falls in the zone's [number*100, top] range are written, in ascending VNUM
// order (C walks the obj_index[] table, which is VNUM-ordered).
func saveOeditZone(world *game.World, zone *parser.Zone) error {
	saveMu := zoneSaveLock(zone.Number)
	saveMu.Lock()
	defer saveMu.Unlock()
	return saveOeditZoneLocked(world, zone)
}

func saveOeditZoneLocked(world *game.World, zone *parser.Zone) error {
	parsed := world.GetParsedWorld()
	if parsed == nil || parsed.SourceDir == "" {
		return fmt.Errorf("world has no source directory")
	}
	// C's OBJ_PREFIX is "world/obj" relative to the lib directory; the Go
	// world derives lib roots from the parsed source directory (SourceDir is
	// the "world" directory itself).
	libDir := filepath.Join(parsed.SourceDir, "obj")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		return err
	}

	// Serialize the whole snapshot -> write -> save-list cleanup against
	// concurrent saves of this zone and against working-copy commits (see
	// saveOeditInternallyLocked): a commit landing mid-save is either fully
	// inside the snapshot or keeps its dirty marker, never silently marked
	// saved.
	objs := world.SnapshotObjs()
	var sb strings.Builder
	for i := range objs {
		obj := &objs[i]
		if obj.VNum < zone.Number*100 || obj.VNum > zone.TopRoom {
			continue
		}
		writeOeditObj(&sb, obj)
	}
	sb.WriteString("$~\n")

	path := filepath.Join(libDir, fmt.Sprintf("%d.obj", zone.Number))
	if err := atomicWriteFile(path, []byte(sb.String()), 0o666); err != nil {
		return err
	}

	clearOLCDirty(olcKindObject, zone.Number)
	return nil
}

// writeOeditObj renders one object record in the C .obj format, mirroring
// oedit_save_to_disk's fprintf sequence field for field (src/oedit.c:372-448).
// The order is four tilde strings (name, short, long, action), the 9-int
// type/extra/wear line, the 4-int values line, "weight cost load" with the load
// as %.2f, then optional S, E, and A blocks and the shared "$~" terminator.
func writeOeditObj(sb *strings.Builder, obj *parser.Obj) {
	// "undefined" is C's NULL fallback for the three mandatory strings only;
	// an empty action description writes nothing between its tildes.
	name := obj.Keywords
	if name == "" {
		name = "undefined"
	}
	short := obj.ShortDesc
	if short == "" {
		short = "undefined"
	}
	long := obj.LongDesc
	if long == "" {
		long = "undefined"
	}
	action := stripOeditString(obj.ActionDesc)

	fmt.Fprintf(sb, "#%d\n%s~\n%s~\n%s~\n%s~\n", obj.VNum, name, short, long, action)
	fmt.Fprintf(sb, "%d %d %d %d %d %d %d %d %d\n",
		obj.TypeFlag,
		obj.ExtraFlags[0], obj.ExtraFlags[1], obj.ExtraFlags[2], obj.ExtraFlags[3],
		obj.WearFlags[0], obj.WearFlags[1], obj.WearFlags[2], obj.WearFlags[3])
	fmt.Fprintf(sb, "%d %d %d %d\n",
		obj.Values[0], obj.Values[1], obj.Values[2], obj.Values[3])
	fmt.Fprintf(sb, "%d %d %s\n", obj.Weight, obj.Cost,
		oeditFormatLoad(float32(obj.LoadPercent)))

	if obj.ScriptName != "" {
		fmt.Fprintf(sb, "S %s %d\n", obj.ScriptName, obj.LuaFunctions)
	}
	for _, exDesc := range obj.ExtraDescs {
		// C's corrupt-ex_desc sanity check skips a node missing either half.
		if exDesc.Keywords == "" || exDesc.Description == "" {
			continue
		}
		fmt.Fprintf(sb, "E\n%s~\n%s~\n", exDesc.Keywords,
			stripOeditString(exDesc.Description))
	}
	for _, affect := range obj.Affects {
		if affect.Location != 0 {
			fmt.Fprintf(sb, "A\n%d %d\n", affect.Location, affect.Modifier)
		}
	}
}

// stripOeditString mirrors C's strip_string (src/olc.c:428-438): copy the text
// while dropping every '\r'.
func stripOeditString(text string) string {
	return strings.ReplaceAll(text, "\r", "")
}
