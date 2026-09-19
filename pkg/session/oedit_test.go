package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeOeditTestWorld builds a world with zone 30 (objects 3000-3099, top 3099)
// and a weapon prototype (3001) plus a container (3002) for oedit tests.
func makeOeditTestWorld(t *testing.T) *game.World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 3000, Name: "Oedit Test Room", Zone: 30},
		},
		Zones: []parser.Zone{
			{Number: 30, Name: "Oedit Test Zone", TopRoom: 3099},
		},
		Objs: []parser.Obj{
			{
				VNum:        3001,
				Keywords:    "sword rusty",
				ShortDesc:   "a rusty sword",
				LongDesc:    "A rusty sword lies here.",
				TypeFlag:    5, // ITEM_WEAPON
				WearFlags:   [4]int{8193, 0, 0, 0},
				Values:      [4]int{0, 3, 4, 11},
				Weight:      5,
				Cost:        100,
				LoadPercent: 50,
				Affects:     []parser.ObjAffect{{Location: 18, Modifier: 2}},
			},
			{
				VNum:        3002,
				Keywords:    "sack leather",
				ShortDesc:   "a leather sack",
				LongDesc:    "A leather sack lies here.",
				TypeFlag:    15, // ITEM_CONTAINER
				WearFlags:   [4]int{1, 0, 0, 0},
				Values:      [4]int{50, 0, 0, 0},
				Weight:      2,
				Cost:        5,
				LoadPercent: 100,
			},
			{
				VNum:        3003,
				Keywords:    "torch",
				ShortDesc:   "a torch",
				LongDesc:    "A torch lies here.",
				TypeFlag:    1, // ITEM_LIGHT
				WearFlags:   [4]int{1, 0, 0, 0},
				Values:      [4]int{0, 0, 4, 0},
				Weight:      1,
				Cost:        2,
				LoadPercent: 0,
			},
			{
				VNum:        3004,
				Keywords:    "bread",
				ShortDesc:   "a loaf of bread",
				LongDesc:    "A loaf of bread lies here.",
				TypeFlag:    19, // ITEM_FOOD
				WearFlags:   [4]int{1, 0, 0, 0},
				Values:      [4]int{6, 0, 0, 0},
				Weight:      1,
				Cost:        1,
				LoadPercent: 100,
			},
		},
		SourceDir: t.TempDir(),
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	return w
}

// makeOeditTestSession builds a builder session confined to zone 30.
func makeOeditTestSession(t *testing.T, m *Manager, name string, level int) *Session {
	t.Helper()
	s := makeCommandTestSession(t, m, name, level, 3000)
	s.olcZone = 30
	return s
}

func TestOeditRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["oedit"]
	if !ok {
		t.Fatal("oedit command has no C gate")
	}
	// C interpreter.c:592: { "oedit", POS_DEAD, do_olc, LVL_BUILDER, SCMD_OLC_OEDIT }
	if gate.MinLevel != 31 || gate.MinPosition != 0 {
		t.Fatalf("oedit gate = (%d,%d), want (31,0)", gate.MinLevel, gate.MinPosition)
	}
	if _, ok := cmdRegistry.Lookup("oedit"); !ok {
		t.Fatal("oedit command is not registered")
	}
}

func TestOeditEntryGates(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)

	builder := makeOeditTestSession(t, m, "Oeditbuilder", 35)

	// No argument: olc_scmd_info[SCMD_OLC_OEDIT].text is "object".
	if err := cmdOedit(builder, nil); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); got != "Specify a object VNUM to edit.\r\n" {
		t.Fatalf("no-arg = %q", got)
	}

	// Nonnumeric argument.
	if err := cmdOedit(builder, []string{"abc"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); got != "Yikes!  Stop that, someone will get hurt!\r\n" {
		t.Fatalf("nonnumeric = %q", got)
	}

	// Unknown zone (vnum 99999 is in no zone).
	if err := cmdOedit(builder, []string{"99999"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); got != "Sorry, there is no zone for that number!\r\n" {
		t.Fatalf("unknown-zone = %q", got)
	}

	// Permission: a builder confined to zone 31 may not edit zone 30's range.
	confined := makeCommandTestSession(t, m, "Confined", 31, 3000)
	confined.olcZone = 31
	if err := cmdOedit(confined, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, confined); got != "You do not have permission to edit this zone.\r\n" {
		t.Fatalf("permission = %q", got)
	}
	// The refused entry must not leave a reservation behind.
	if holder := m.objEditHolder(3001); holder != "" {
		t.Fatalf("refused entry left reservation held by %q", holder)
	}

	// A valid object enters the editor and releases on quit.
	if err := cmdOedit(builder, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); !strings.HasPrefix(got, "\r\n-- Item number : [3001]") {
		t.Fatalf("main menu = %q", got)
	}
	if holder := m.objEditHolder(3001); holder != "Oeditbuilder" {
		t.Fatalf("editor reservation holder = %q, want Oeditbuilder", holder)
	}
	// An unchanged object's 'q' is C's silent cleanup_olc(CLEANUP_ALL): the
	// editor exits with no player-facing bytes at all.
	builder.handleOeditInput("q")
	if holder := m.objEditHolder(3001); holder != "" {
		t.Fatalf("quit left reservation held by %q", holder)
	}
	builder.textEditMu.Lock()
	closed := builder.oedit == nil
	builder.textEditMu.Unlock()
	if !closed {
		t.Fatal("quit did not clear the oedit state")
	}
}

func TestOeditNoZoneSaveGate(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	builder := makeOeditTestSession(t, m, "Zonesave", 35)

	// "oedit save 30" writes zone 30's .obj file (number = 30*100).
	if err := cmdOedit(builder, []string{"save", "30"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); got != "Saving all objects in zone.\r\n" {
		t.Fatalf("zone save = %q", got)
	}

	// "oedit save" without a zone number.
	if err := cmdOedit(builder, []string{"save"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); got != "Save which zone?\r\n" {
		t.Fatalf("save no-arg = %q", got)
	}
}

// openOedit enters the editor for vnum and consumes the initial main menu,
// returning it.
func openOedit(t *testing.T, s *Session, vnum string) string {
	t.Helper()
	if err := cmdOedit(s, []string{vnum}); err != nil {
		t.Fatalf("oedit %s: %v", vnum, err)
	}
	return readMsgText(t, s)
}

// TestOeditWeaponVal3Fallthrough is the instruction-attention test: C's
// ITEM_WEAPON case in OEDIT_VALUE_3 (oedit.c:1333-1340) sets 1..50 and then
// falls through the missing break into WAND/STAFF, which overwrite min/max to
// 0..20. Weapons are clamped 0-20, never 1-50.
func TestOeditWeaponVal3Fallthrough(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  int
	}{
		{"50", 20}, // would be 50 under the "intended" 1..50 range
		{"25", 20},
		{"20", 20},
		{"7", 7},
		{"-5", 0},
	} {
		t.Run("input-"+tc.input, func(t *testing.T) {
			w := makeOeditTestWorld(t)
			m := newTestManager(t, w, nil)
			s := makeOeditTestSession(t, m, "Weaponval", 35)
			openOedit(t, s, "3001")

			// 'd' clears values and enters the cascade. WEAPON skips val0, so
			// the first prompt is val2's damage-dice prompt.
			s.handleOeditInput("d")
			if got := readMsgText(t, s); got != "Number of damage dice : " {
				t.Fatalf("value cascade first prompt = %q", got)
			}
			s.handleOeditInput("3")
			if got := readMsgText(t, s); got != "Size of damage dice : " {
				t.Fatalf("val3 prompt = %q", got)
			}
			s.handleOeditInput(tc.input)
			// val4 for a weapon is the attack-type menu.
			if got := readMsgText(t, s); !strings.HasSuffix(got, "Enter weapon type : ") {
				t.Fatalf("weapon menu = %q", got)
			}
			s.handleOeditInput("0")
			menu := readMsgText(t, s)

			s.textEditMu.Lock()
			got := s.oedit.obj.Values[2]
			s.textEditMu.Unlock()
			if got != tc.want {
				t.Fatalf("val3 for input %q = %d, want %d (C: %q)",
					tc.input, got, tc.want, menu)
			}
		})
	}
}

// TestOeditCostPromptHasNoMaximum pins both halves of the cost trap: the
// prompt claims a 10000 maximum (oedit.c:1105) and the parser is a bare atoi
// with no clamp (oedit.c:1243-1245).
func TestOeditCostPromptHasNoMaximum(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeOeditTestSession(t, m, "Costgod", 35)
	openOedit(t, s, "3001")

	s.handleOeditInput("9")
	if got := readMsgText(t, s); got != "Enter cost (10000 max): " {
		t.Fatalf("cost prompt = %q", got)
	}
	s.handleOeditInput("999999")
	menu := readMsgText(t, s)
	if !strings.Contains(menu, "9\x1b[0m) Cost        : \x1b[36m999999") &&
		!strings.Contains(menu, "9) Cost        : 999999") {
		t.Fatalf("cost not stored unclamped: %q", menu)
	}
	s.textEditMu.Lock()
	got := s.oedit.obj.Cost
	s.textEditMu.Unlock()
	if got != 999999 {
		t.Fatalf("cost = %d, want 999999", got)
	}
}

// TestOeditTypeMenuListsZeroButRejectsIt pins oedit.c:1180-1188: the type menu
// numbers entries from 0, yet 0 (and NUM_ITEM_TYPES itself) are refused with
// "Invalid choice, try again : ".
func TestOeditTypeMenuListsZeroButRejectsIt(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeOeditTestSession(t, m, "Typegod", 35)
	openOedit(t, s, "3001")

	s.handleOeditInput("5")
	menu := readMsgText(t, s)
	if !strings.HasPrefix(menu, "\r\n 0) UNDEFINED") || !strings.Contains(menu, "23) FOUNTAIN") {
		t.Fatalf("type menu = %q", menu)
	}
	if !strings.HasSuffix(menu, "\r\nEnter object type : ") {
		t.Fatalf("type menu prompt = %q", menu)
	}

	for _, rejected := range []string{"0", "24", "99", "abc"} {
		s.handleOeditInput(rejected)
		if got := readMsgText(t, s); got != "Invalid choice, try again : " {
			t.Fatalf("type %q = %q", rejected, got)
		}
	}
	s.textEditMu.Lock()
	before := s.oedit.obj.TypeFlag
	s.textEditMu.Unlock()
	if before != 5 {
		t.Fatalf("rejected type changed the prototype: %d", before)
	}

	// A legal type sticks and returns to the main menu.
	s.handleOeditInput("9")
	if got := readMsgText(t, s); !strings.HasPrefix(got, "\r\n-- Item number : [3001]") {
		t.Fatalf("after type select = %q", got)
	}
	s.textEditMu.Lock()
	after := s.oedit.obj.TypeFlag
	s.textEditMu.Unlock()
	if after != 9 {
		t.Fatalf("type = %d, want 9 (ARMOR)", after)
	}
}

// TestOeditExtrasVsWearErrorAsymmetry pins oedit.c:1191-1197 against
// oedit.c:1215-1222: an invalid extra-flag number silently redisplays the
// extras menu, while an invalid wear-flag number emits
// "That's not a valid choice!\r\n" before its menu.
func TestOeditExtrasVsWearErrorAsymmetry(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeOeditTestSession(t, m, "Flaggod", 35)
	openOedit(t, s, "3001")

	s.handleOeditInput("6")
	extrasMenu := readMsgText(t, s)
	if !strings.HasPrefix(extrasMenu, "\r\n 1) GLOW") ||
		!strings.HasSuffix(extrasMenu, "Enter object extra flag (0 to quit) : ") {
		t.Fatalf("extras menu = %q", extrasMenu)
	}

	// Invalid: exactly the menu again, with no error line.
	s.handleOeditInput("30")
	if got := readMsgText(t, s); got != extrasMenu {
		t.Fatalf("invalid extras = %q, want the bare menu %q", got, extrasMenu)
	}
	// Still in OEDIT_EXTRAS, so a valid toggle works.
	s.handleOeditInput("1")
	if got := readMsgText(t, s); !strings.Contains(got, "Object flags: GLOW ") {
		t.Fatalf("GLOW toggle = %q", got)
	}
	// 0 leaves to the main menu.
	s.handleOeditInput("0")
	if got := readMsgText(t, s); !strings.HasPrefix(got, "\r\n-- Item number : [3001]") {
		t.Fatalf("extras quit = %q", got)
	}

	s.handleOeditInput("7")
	wearMenu := readMsgText(t, s)
	if !strings.HasPrefix(wearMenu, "\r\n 1) TAKE") {
		t.Fatalf("wear menu = %q", wearMenu)
	}
	s.handleOeditInput("20")
	if got := readMsgText(t, s); got != "That's not a valid choice!\r\n"+wearMenu {
		t.Fatalf("invalid wear = %q", got)
	}
}

// TestOeditPercentLoadFloatSemantics pins oedit.c's float load path: atof,
// round_float(0.01) in single precision, clamp 0..100, then the menu's two
// renderings ("%.2f" and "1 in %d" from (int)(100.0/load + 0.5), or "Never").
func TestOeditPercentLoadFloatSemantics(t *testing.T) {
	for _, tc := range []struct {
		input    string
		percent  string
		loadText string
	}{
		{"12.567", "12.57", "1 in 8"},
		{"0", "0.00", "Never"},
		{"150", "100.00", "1 in 1"},
		{"-5", "0.00", "Never"},
		{"abc", "0.00", "Never"},
		{"10.5", "10.50", "1 in 10"},
	} {
		t.Run("input-"+tc.input, func(t *testing.T) {
			w := makeOeditTestWorld(t)
			m := newTestManager(t, w, nil)
			s := makeOeditTestSession(t, m, "Loadgod", 35)
			openOedit(t, s, "3001")

			s.handleOeditInput("a")
			if got := readMsgText(t, s); got != "Enter percent chance\r\nof the item loading : " {
				t.Fatalf("load prompt = %q", got)
			}
			s.handleOeditInput(tc.input)
			menu := readMsgText(t, s)
			want := "Percent Load: " + tc.percent + "% (" + tc.loadText + ")\r\n"
			if !strings.Contains(menu, want) {
				t.Fatalf("load menu for %q = %q, want it to contain %q", tc.input, menu, want)
			}
		})
	}
}

// TestOeditValueCascadeSkipsByType pins the per-type jumps at
// oedit.c:717-858: LIGHT lands on val3, WEAPON/MISSILE/FIREWEAPON skip val0,
// FOOD skips vals 2-3, and CONTAINER's val2 is the flag menu.
func TestOeditValueCascadeSkipsByType(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)

	// LIGHT: straight to val3.
	light := makeOeditTestSession(t, m, "Lightgod", 35)
	openOedit(t, light, "3003")
	light.handleOeditInput("d")
	if got := readMsgText(t, light); got != "Number of hours (0 = burnt, -1 is infinite) : " {
		t.Fatalf("light val1 = %q", got)
	}
	light.handleOeditInput("-1")
	// val4 for LIGHT is the default case, so the main menu returns.
	if got := readMsgText(t, light); !strings.HasPrefix(got, "\r\n-- Item number : [3003]") {
		t.Fatalf("light after val3 = %q", got)
	}

	// FOOD: val1 prompt, then val4's poisoned prompt (vals 2-3 skipped).
	food := makeOeditTestSession(t, m, "Foodgod", 35)
	openOedit(t, food, "3004")
	food.handleOeditInput("d")
	if got := readMsgText(t, food); got != "Hours to fill stomach : " {
		t.Fatalf("food val1 = %q", got)
	}
	food.handleOeditInput("5")
	if got := readMsgText(t, food); got != "Poisoned (0 = not poison) : " {
		t.Fatalf("food val2 -> val4 = %q", got)
	}

	// CONTAINER: val1 weight, val2 container flags, val3 key vnum.
	container := makeOeditTestSession(t, m, "Sackgod", 35)
	openOedit(t, container, "3002")
	container.handleOeditInput("d")
	if got := readMsgText(t, container); got != "Max weight to contain : " {
		t.Fatalf("container val1 = %q", got)
	}
	container.handleOeditInput("50")
	flagsMenu := readMsgText(t, container)
	if !strings.HasPrefix(flagsMenu, "\r\n1) CLOSEABLE\r\n") ||
		!strings.HasSuffix(flagsMenu, "Container flags: NOBITS \r\nEnter flag, 0 to quit : ") {
		t.Fatalf("container flags menu = %q", flagsMenu)
	}
	container.handleOeditInput("1")
	if got := readMsgText(t, container); !strings.Contains(got, "Container flags: CLOSEABLE ") {
		t.Fatalf("container flag toggle = %q", got)
	}
	container.handleOeditInput("0")
	if got := readMsgText(t, container); got != "Vnum of key to open container (-1 for no key) : " {
		t.Fatalf("container val2 -> val3 = %q", got)
	}
	container.handleOeditInput("-1")
	if got := readMsgText(t, container); !strings.HasPrefix(got, "\r\n-- Item number : [3002]") {
		t.Fatalf("container val3 -> main = %q", got)
	}
}

// TestOeditExtraDescMenuIsNotRedits pins oedit.c:529-556: this is its own menu
// (header, a "0) Quit" line, "Enter choice : ") and its "Goto next
// description:" value embeds a CRLF inside the "<Not set>" string, unlike
// redit's "<NOT SET>".
func TestOeditExtraDescMenuIsNotRedits(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeOeditTestSession(t, m, "Extragod", 35)
	openOedit(t, s, "3001")

	wantEmpty := "\r\nExtra desc menu\r\n" +
		"1) Keyword: <NONE>\r\n" +
		"2) Description:\r\n<NONE>\r\n" +
		"3) Goto next description: <Not set>\r\n\r\n" +
		"0) Quit\r\n" +
		"Enter choice : "

	s.handleOeditInput("f")
	if got := readMsgText(t, s); got != wantEmpty {
		t.Fatalf("fresh extra-desc menu = %q, want %q", got, wantEmpty)
	}

	s.handleOeditInput("1")
	if got := readMsgText(t, s); got != "Enter keywords, separated by spaces :-\r\n| " {
		t.Fatalf("keyword prompt = %q", got)
	}
	// C's str_udup turns an empty keyword into the literal "undefined".
	s.handleOeditInput("")
	if got := readMsgText(t, s); !strings.Contains(got, "1) Keyword: undefined\r\n") {
		t.Fatalf("empty keyword = %q", got)
	}

	s.handleOeditInput("2")
	if got := readMsgText(t, s); got != "Instructions: /s or @ to save, /h for more options.\r\n"+
		"Enter the extra description:\r\n\r\n" {
		t.Fatalf("extra-desc editor = %q", got)
	}
	// The improved editor does not echo lines.
	s.handleOeditInput("It is rusty.")
	s.handleOeditInput("@")
	saved := readMsgText(t, s)
	if !strings.Contains(saved, "2) Description:\r\nIt is rusty.\r\n") {
		t.Fatalf("saved description menu = %q", saved)
	}
	if !strings.Contains(saved, "3) Goto next description: <Not set>\r\n") {
		t.Fatalf("single node should still report <Not set>: %q", saved)
	}

	// 3 advances to a fresh node only because both halves are set.
	s.handleOeditInput("3")
	if got := readMsgText(t, s); !strings.Contains(got, "3) Goto next description: <Not set>\r\n") {
		t.Fatalf("second node menu = %q", got)
	}
	s.textEditMu.Lock()
	nodes := len(s.oedit.obj.ExtraDescs)
	cur := s.oedit.currentExtra
	s.textEditMu.Unlock()
	if nodes != 2 || cur != 1 {
		t.Fatalf("extra desc nodes=%d current=%d, want 2/1", nodes, cur)
	}

	// 0 on the incomplete node truncates it, then returns to the main menu.
	s.handleOeditInput("0")
	if got := readMsgText(t, s); !strings.HasPrefix(got, "\r\n-- Item number : [3001]") {
		t.Fatalf("extra-desc quit = %q", got)
	}
	s.textEditMu.Lock()
	nodes = len(s.oedit.obj.ExtraDescs)
	s.textEditMu.Unlock()
	if nodes != 1 {
		t.Fatalf("incomplete node not truncated: %d nodes", nodes)
	}
}

// TestOeditApplyPromptForms pins oedit.c:559-593: per-slot prompt rendering
// with "%+d" plain modifiers, "None." for empty slots, and the special
// APPLY_RACE_HATE / APPLY_SPELL forms that print the modifier's table name.
func TestOeditApplyPromptForms(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeOeditTestSession(t, m, "Applygod", 35)
	openOedit(t, s, "3001")

	wantPrompt := "\r\n 1) +2 to HITROLL\r\n"
	for i := 2; i <= maxObjAffect; i++ {
		wantPrompt += " " + string(rune('0'+i)) + ") None.\r\n"
	}
	wantPrompt += "\r\nEnter affection to modify (0 to quit) : "
	s.handleOeditInput("e")
	if got := readMsgText(t, s); got != wantPrompt {
		t.Fatalf("prompt-apply menu = %q, want %q", got, wantPrompt)
	}

	s.handleOeditInput("2")
	applyMenu := readMsgText(t, s)
	if !strings.HasPrefix(applyMenu, "\r\n 0) NONE") ||
		!strings.HasSuffix(applyMenu, "\r\nEnter apply type (0 is no apply) : ") {
		t.Fatalf("apply menu = %q", applyMenu)
	}

	// APPLY_RACE_HATE routes to the race menu and renders "<race> to RACE_HATE".
	s.handleOeditInput("25")
	if got := readMsgText(t, s); !strings.HasSuffix(got, "Enter mob race : ") {
		t.Fatalf("race menu = %q", got)
	}
	s.handleOeditInput("5") // Rakshasa (mob_races[5])
	if got := readMsgText(t, s); !strings.Contains(got, " 2) Rakshasa to RACE_HATE\r\n") {
		t.Fatalf("race-hate apply = %q", got)
	}
	s.handleOeditInput("0")
	if got := readMsgText(t, s); !strings.HasPrefix(got, "\r\n-- Item number : [3001]") {
		t.Fatalf("apply quit = %q", got)
	}

	// APPLY_SPELL routes to the perm-effect menu (affected_bits, 37 entries).
	s.handleOeditInput("e")
	_ = readMsgText(t, s)
	s.handleOeditInput("3")
	_ = readMsgText(t, s)
	s.handleOeditInput("29")
	spellMenu := readMsgText(t, s)
	if !strings.HasPrefix(spellMenu, "\r\n 0) BLIND") ||
		!strings.HasSuffix(spellMenu, "Enter perm spell effect : ") {
		t.Fatalf("perm-spell menu = %q", spellMenu)
	}
	if !strings.Contains(spellMenu, "36) WATERBREATHE") {
		t.Fatalf("perm-spell menu missing the NUM_AFF_FLAGS tail: %q", spellMenu)
	}
	s.handleOeditInput("8") // GROUP
	if got := readMsgText(t, s); !strings.Contains(got, " 3) GROUP to PERM_SPELL\r\n") {
		t.Fatalf("perm-spell apply = %q", got)
	}
	s.handleOeditInput("0")
	_ = readMsgText(t, s)
}

// TestOeditConfirmSavePromptAsymmetry pins oedit.c:1043-1045: the first save
// prompt ends with " : " and the invalid-input retry does not.
func TestOeditConfirmSavePromptAsymmetry(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeOeditTestSession(t, m, "Savesave", 35)
	openOedit(t, s, "3001")

	s.handleOeditInput("1")
	_ = readMsgText(t, s)
	s.handleOeditInput("shiny sword")
	_ = readMsgText(t, s)

	s.handleOeditInput("q")
	if got := readMsgText(t, s); got != "Do you wish to save this object internally? : " {
		t.Fatalf("first save prompt = %q", got)
	}
	s.handleOeditInput("z")
	if got := readMsgText(t, s); got != "Invalid choice!\r\nDo you wish to save this object internally?\r\n" {
		t.Fatalf("retry save prompt = %q", got)
	}
	s.handleOeditInput("y")
	if got := readMsgText(t, s); got != "Saving object to memory.\r\n" {
		t.Fatalf("save message = %q", got)
	}
	if live, ok := w.SnapshotObj(3001); !ok || live.Keywords != "shiny sword" {
		t.Fatalf("live prototype after save = %#v", live)
	}
}

// TestOeditNewObjectDefaults pins oedit_setup_new (oedit.c:98-109) on top of
// clear_object: the unfinished strings, ITEM_WEAR_TAKE, and everything else
// zero.
func TestOeditNewObjectDefaults(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeOeditTestSession(t, m, "Newobj", 35)

	menu := openOedit(t, s, "3010")
	want := "\r\n-- Item number : [3010]\r\n" +
		"1) Namelist : unfinished object\r\n" +
		"2) S-Desc   : an unfinished object\r\n" +
		"3) L-Desc   :-\r\nAn unfinished object is lying here.\r\n" +
		"4) A-Desc   :-\r\n<not set>\r\n" +
		"5) Type        : UNDEFINED\r\n" +
		"6) Extra flags : NOBITS \r\n" +
		"7) Wear flags  : TAKE \r\n" +
		"8) Encumbrance : 0\r\n" +
		"9) Cost        : 0\r\n" +
		"A) Percent Load: 0.00% (Never)\r\n" +
		"B) Timer       : 0\r\n" +
		"D) Values      : 0 0 0 0\r\n" +
		"E) Applies menu\r\n" +
		"F) Extra descriptions menu\r\n" +
		"S) Scripts menu\r\n" +
		"Q) Quit\r\n" +
		"Enter choice : "
	if menu != want {
		t.Fatalf("new-object menu = %q, want %q", menu, want)
	}

	// A never-saved object cannot be given a script (oedit.c:475-486).
	s.handleOeditInput("s")
	got := readMsgText(t, s)
	if !strings.HasPrefix(got, "\r\nCannot assign a script until the object is saved at least once.\r\n") {
		t.Fatalf("new script gate = %q", got)
	}
	if !strings.Contains(got, "Enter choice : ") {
		t.Fatalf("new script gate did not fall back to the menu: %q", got)
	}
}

// TestOeditWorkingCopySaveAndAbort proves edits stay descriptor-local until the
// save confirmation is accepted.
func TestOeditWorkingCopySaveAndAbort(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)

	abort := makeOeditTestSession(t, m, "Abortedit", 35)
	openOedit(t, abort, "3001")
	abort.handleOeditInput("1")
	_ = readMsgText(t, abort)
	abort.handleOeditInput("discarded name")
	_ = readMsgText(t, abort)
	abort.handleOeditInput("q")
	_ = readMsgText(t, abort)
	abort.handleOeditInput("n")
	if live, ok := w.SnapshotObj(3001); !ok || live.Keywords != "sword rusty" {
		t.Fatalf("abort published the working copy: %#v", live)
	}

	commit := makeOeditTestSession(t, m, "Commitedit", 35)
	openOedit(t, commit, "3001")
	commit.handleOeditInput("1")
	_ = readMsgText(t, commit)
	commit.handleOeditInput("committed name")
	_ = readMsgText(t, commit)
	commit.handleOeditInput("q")
	_ = readMsgText(t, commit)
	commit.handleOeditInput("y")
	_ = readMsgText(t, commit)
	if live, ok := w.SnapshotObj(3001); !ok || live.Keywords != "committed name" {
		t.Fatalf("commit did not publish: %#v", live)
	}
}

// TestOeditScriptShallowBehavior proves the script menu edits the LIVE object
// index (C's GET_OBJ_SCRIPT is obj_index[rnum].script), so script changes
// survive discarding the rest of the working copy.
func TestOeditScriptShallowBehavior(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeOeditTestSession(t, m, "Scriptgod", 35)
	openOedit(t, s, "3001")

	s.handleOeditInput("s")
	if got := readMsgText(t, s); got != "\r\n1) Name: None\r\n2) Script Flags: NOBITS \r\nEnter choice (0 to quit) : " {
		t.Fatalf("script menu = %q", got)
	}
	s.handleOeditInput("1")
	if got := readMsgText(t, s); got != "Enter script name: " {
		t.Fatalf("script name prompt = %q", got)
	}
	s.handleOeditInput("sword_script")
	if got := readMsgText(t, s); !strings.Contains(got, "1) Name: sword_script\r\n") {
		t.Fatalf("script menu after name = %q", got)
	}
	s.handleOeditInput("2")
	flagsMenu := readMsgText(t, s)
	if !strings.HasPrefix(flagsMenu, "\x1b[H\x1b[J") ||
		!strings.HasSuffix(flagsMenu, "Current flags   : NOBITS \r\nEnter script flags (0 to quit) : ") {
		t.Fatalf("script flags menu = %q", flagsMenu)
	}
	s.handleOeditInput("2") // ONCMD is bit 1
	if got := readMsgText(t, s); !strings.Contains(got, "Current flags   : ONCMD ") {
		t.Fatalf("script flag toggle = %q", got)
	}

	// Discard the object edit; the script fields must survive.
	s.handleOeditInput("0")
	_ = readMsgText(t, s)
	s.handleOeditInput("0")
	_ = readMsgText(t, s)
	s.handleOeditInput("q")
	_ = readMsgText(t, s)
	s.handleOeditInput("n")
	if live, ok := w.SnapshotObj(3001); !ok || live.ScriptName != "sword_script" || live.LuaFunctions != 2 {
		t.Fatalf("shallow script not live after discard: %#v", live)
	}
}

// TestOeditDisconnectCleanup mirrors C's cleanup_olc(CLEANUP_ALL) on descriptor
// close: the working copy is dropped without committing and PLR_WRITING clears.
func TestOeditDisconnectCleanup(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeOeditTestSession(t, m, "Disconnect", 35)
	openOedit(t, s, "3001")
	s.handleOeditInput("1")
	_ = readMsgText(t, s)
	s.handleOeditInput("unsaved change")
	_ = readMsgText(t, s)

	s.cancelOedit()
	if live, ok := w.SnapshotObj(3001); !ok || live.Keywords != "sword rusty" {
		t.Fatalf("disconnect saved changes: %#v", live)
	}
	s.textEditMu.Lock()
	closed := s.oedit == nil
	writing := s.player.GetFlags()&(1<<uint(game.PlrWriting)) != 0
	s.textEditMu.Unlock()
	if !closed || writing {
		t.Fatal("disconnect did not clean up session state")
	}
	if holder := m.objEditHolder(3001); holder != "" {
		t.Fatalf("disconnect left reservation held by %q", holder)
	}
}

// TestOeditNewObjectCommitAndDiskSave proves a new object commits to the live
// world, is written in C .obj form, and parses back after the restart boundary.
func TestOeditNewObjectCommitAndDiskSave(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeOeditTestSession(t, m, "Diskobj", 35)

	openOedit(t, s, "3010")
	s.handleOeditInput("1")
	_ = readMsgText(t, s)
	s.handleOeditInput("a brand new thing")
	_ = readMsgText(t, s)
	s.handleOeditInput("q")
	_ = readMsgText(t, s)
	s.handleOeditInput("y")
	if got := readMsgText(t, s); got != "Saving object to memory.\r\n" {
		t.Fatalf("save message = %q", got)
	}
	live, ok := w.SnapshotObj(3010)
	if !ok || live.Keywords != "a brand new thing" {
		t.Fatalf("new object not committed: %#v", live)
	}

	if err := cmdOedit(s, []string{"save", "30"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "Saving all objects in zone.\r\n" {
		t.Fatalf("zone save = %q", got)
	}

	dir := w.GetParsedWorld().SourceDir
	data, err := os.ReadFile(filepath.Join(dir, "obj", "30.obj"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)

	// A never-edited field falls back to C's literal "undefined"; an empty
	// action description writes nothing between its tildes.
	wantRecord := "#3010\n" +
		"a brand new thing~\n" +
		"an unfinished object~\n" +
		"An unfinished object is lying here.~\n" +
		"~\n" +
		"0 0 0 0 0 1 0 0 0\n" +
		"0 0 0 0\n" +
		"0 0 0.00\n"
	if !strings.Contains(text, wantRecord) {
		t.Fatalf("saved .obj missing new record:\nwant %q\nin %q", wantRecord, text)
	}
	if !strings.HasSuffix(text, "$~\n") {
		t.Fatalf(".obj missing $~ terminator: %q", text)
	}
	// Zone neighbours are re-emitted from the live prototypes, affects last.
	wantWeapon := "#3001\n" +
		"sword rusty~\n" +
		"a rusty sword~\n" +
		"A rusty sword lies here.~\n" +
		"~\n" +
		"5 0 0 0 0 8193 0 0 0\n" +
		"0 3 4 11\n" +
		"5 100 50.00\n" +
		"A\n18 2\n"
	if !strings.Contains(text, wantWeapon) {
		t.Fatalf("saved .obj missing weapon record:\nwant %q\nin %q", wantWeapon, text)
	}

	objs, err := parser.ParseObjFile(filepath.Join(dir, "obj", "30.obj"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, obj := range objs {
		if obj.VNum == 3010 {
			found = true
			if obj.Keywords != "a brand new thing" || obj.WearFlags != [4]int{1, 0, 0, 0} {
				t.Fatalf("reparsed 3010 = %#v", obj)
			}
		}
	}
	if !found {
		t.Fatal("3010 not found after reparsing the saved zone")
	}
}

// TestOeditRefreshLiveInstancesPreservesRuntimePlacement pins
// oedit_save_internally's object_list sweep (oedit.c:180-198): every live
// instance takes the edited full struct while its runtime placement survives,
// and the instance-level overrides C would overwrite are cleared.
func TestOeditRefreshLiveInstancesPreservesRuntimePlacement(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)

	inst, err := w.SpawnObject(3001, 3000)
	if err != nil {
		t.Fatal(err)
	}
	if inst.Location.Kind != game.ObjInRoom || inst.RoomVNum != 3000 {
		t.Fatalf("spawn placement = %#v", inst.Location)
	}
	override := [4]int{9, 9, 9, 9}
	inst.ValuesOverride = &override
	inst.ExtraFlagsOverride = [4]int{7, 0, 0, 0}

	s := makeOeditTestSession(t, m, "Liveobj", 35)
	openOedit(t, s, "3001")
	s.handleOeditInput("1")
	_ = readMsgText(t, s)
	s.handleOeditInput("edited weapon")
	_ = readMsgText(t, s)
	s.handleOeditInput("q")
	_ = readMsgText(t, s)
	s.handleOeditInput("y")
	_ = readMsgText(t, s)

	if inst.Prototype == nil || inst.Prototype.Keywords != "edited weapon" {
		t.Fatalf("live instance prototype not swapped: %#v", inst.Prototype)
	}
	if inst.Location.Kind != game.ObjInRoom || inst.RoomVNum != 3000 {
		t.Fatalf("runtime placement lost: %#v", inst.Location)
	}
	if inst.ValuesOverride != nil || inst.ExtraFlagsOverride != ([4]int{}) {
		t.Fatalf("instance overrides survived C's full-struct swap: values=%v extras=%v",
			inst.ValuesOverride, inst.ExtraFlagsOverride)
	}
}

// TestOeditMenuColorsMatchGetCharCols pins the three OLC color levels for the
// object menus. get_char_cols emits escapes only at C_NRM, which Go models as
// the PRF_COLOR_2 bit; creation's single "Y" sets both bits, so levels 2 and 3
// render identically and level 0 carries no escape bytes at all.
func TestOeditMenuColorsMatchGetCharCols(t *testing.T) {
	const colorMain = "\r\n-- Item number : [\x1b[36m3001\x1b[0m]\r\n" +
		"\x1b[32m1\x1b[0m) Namelist : \x1b[33msword rusty\r\n" +
		"\x1b[32m2\x1b[0m) S-Desc   : \x1b[33ma rusty sword\r\n" +
		"\x1b[32m3\x1b[0m) L-Desc   :-\r\n\x1b[33mA rusty sword lies here.\r\n" +
		"\x1b[32m4\x1b[0m) A-Desc   :-\r\n\x1b[33m<not set>\r\n" +
		"\x1b[32m5\x1b[0m) Type        : \x1b[36mWEAPON\r\n" +
		"\x1b[32m6\x1b[0m) Extra flags : \x1b[36mNOBITS \r\n" +
		"\x1b[32m7\x1b[0m) Wear flags  : \x1b[36mTAKE WIELD \r\n" +
		"\x1b[32m8\x1b[0m) Encumbrance : \x1b[36m5\r\n" +
		"\x1b[32m9\x1b[0m) Cost        : \x1b[36m100\r\n" +
		"\x1b[32mA\x1b[0m) Percent Load: \x1b[36m50.00% (1 in 2)\r\n" +
		"\x1b[32mB\x1b[0m) Timer       : \x1b[36m0\r\n" +
		"\x1b[32mD\x1b[0m) Values      : \x1b[36m0 3 4 11\r\n" +
		"\x1b[32mE\x1b[0m) Applies menu\r\n" +
		"\x1b[32mF\x1b[0m) Extra descriptions menu\r\n" +
		"\x1b[32mS\x1b[0m) Scripts menu\r\n" +
		"\x1b[32mQ\x1b[0m) Quit\r\n" +
		"Enter choice : "

	for _, tc := range []struct {
		name  string
		level int
	}{
		{"color-off", 0},
		{"color-normal", 2},
		{"color-complete", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := makeOeditTestWorld(t)
			m := newTestManager(t, w, nil)
			s := makeOeditTestSession(t, m, "Colorgod", 35)
			switch tc.level {
			case 2:
				s.player.SetPlrFlag(game.PrfColor2, true)
			case 3:
				s.player.SetPlrFlag(game.PrfColor1, true)
				s.player.SetPlrFlag(game.PrfColor2, true)
			}

			if err := ExecuteCommand(s, "oedit", []string{"3001"}); err != nil {
				t.Fatal(err)
			}
			wantMain := colorMain
			if tc.level < 2 {
				strip := func(s string) string {
					s = strings.ReplaceAll(s, "\x1b[0m", "")
					s = strings.ReplaceAll(s, "\x1b[32m", "")
					s = strings.ReplaceAll(s, "\x1b[33m", "")
					return strings.ReplaceAll(s, "\x1b[36m", "")
				}
				wantMain = strip(colorMain)
			}
			got := readMsgText(t, s)
			if got != wantMain {
				t.Fatalf("main menu = %q, want %q", got, wantMain)
			}
			if tc.level < 2 && strings.ContainsAny(got, "\x1b") {
				t.Fatalf("level-0 menu invented escape bytes: %q", got)
			}

			// The type menu is the widest two-column table; check its color
			// placement and padding.
			s.handleOeditInput("5")
			typeMenu := readMsgText(t, s)
			if tc.level >= 2 {
				if !strings.HasPrefix(typeMenu, "\r\n\x1b[32m 0\x1b[0m) UNDEFINED            \x1b[32m 1\x1b[0m) LIGHT") {
					t.Fatalf("type menu header = %q", typeMenu)
				}
			} else if !strings.HasPrefix(typeMenu, "\r\n 0) UNDEFINED             1) LIGHT") {
				t.Fatalf("plain type menu header = %q", typeMenu)
			}

			// Container flags use the cyan sprintbit form and a separate
			// leading blank line in C.
			s.handleOeditInput("0")
			_ = readMsgText(t, s)
			s.handleOeditInput("7")
			_ = readMsgText(t, s)
			s.handleOeditInput("0")
			_ = readMsgText(t, s)
			s.handleOeditInput("q")
			_ = readMsgText(t, s)
		})
	}
}

// TestOeditConcurrentEntryAdmitsExactlyOneEditor proves the atomic
// reservation, mirroring do_olc's single-threaded duplicate scan.
func TestOeditConcurrentEntryAdmitsExactlyOneEditor(t *testing.T) {
	w := makeOeditTestWorld(t)
	m := newTestManager(t, w, nil)
	const contenders = 8
	sessions := make([]*Session, contenders)
	for i := range sessions {
		sessions[i] = makeOeditTestSession(t, m, fmt.Sprintf("Oraced%d", i), 35)
	}

	admitted := make([]bool, contenders)
	refused := make([]bool, contenders)
	var wg sync.WaitGroup
	for i := range sessions {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := cmdOedit(sessions[i], []string{"3001"}); err != nil {
				t.Errorf("contender %d: %v", i, err)
				return
			}
			select {
			case msg := <-sessions[i].send:
				var sm ServerMessage
				if err := json.Unmarshal(msg, &sm); err != nil {
					t.Errorf("contender %d: %v", i, err)
					return
				}
				data, _ := json.Marshal(sm.Data)
				var event EventData
				if err := json.Unmarshal(data, &event); err != nil {
					t.Errorf("contender %d: %v", i, err)
					return
				}
				switch {
				case strings.Contains(event.Text, "-- Item number : [3001]"):
					admitted[i] = true
				case strings.HasPrefix(event.Text, "That object is currently being edited by Oraced"):
					refused[i] = true
				default:
					t.Errorf("contender %d output = %q", i, event.Text)
				}
			case <-time.After(5 * time.Second):
				t.Errorf("contender %d: no admission response", i)
			}
		}(i)
	}
	wg.Wait()

	admittedCount := 0
	for i := range admitted {
		if admitted[i] {
			admittedCount++
		}
		if admitted[i] && refused[i] {
			t.Errorf("contender %d both admitted and refused", i)
		}
		if !admitted[i] && !refused[i] {
			t.Errorf("contender %d neither admitted nor refused", i)
		}
	}
	if admittedCount != 1 {
		t.Fatalf("admitted %d editors, want exactly 1", admittedCount)
	}
}
