package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeMeditTestWorld builds a world with zone 30 (rooms 3000-3099, top 3099)
// and two mob prototypes (3001, 3002) for medit tests.
func makeMeditTestWorld(t *testing.T) *game.World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 3000, Name: "Medit Test Room", Zone: 30},
		},
		Zones: []parser.Zone{
			{Number: 30, Name: "Medit Test Zone", TopRoom: 3099},
		},
		Mobs: []parser.Mob{
			{
				VNum:         3001,
				Keywords:     "goblin guard",
				ShortDesc:    "a goblin guard",
				LongDesc:     "A scrawny goblin stands watch.\r\n",
				DetailedDesc: "It looks mean.\r\n",
				Sex:          1,
				Race:         2,
				Level:        5,
				Alignment:    -100,
				THAC0:        15,
				AC:           50,
				HP:           parser.DiceRoll{Num: 2, Sides: 8, Plus: 10},
				Damage:       parser.DiceRoll{Num: 1, Sides: 6, Plus: 2},
				Gold:         50,
				Exp:          1000,
				Position:     8,
				DefaultPos:   8,
				ActionFlags:  []string{"ISNPC", "SENTINEL"},
				ScriptName:   "goblin_script",
				LuaFunctions: 4, // GREET
			},
			{
				VNum:        3002,
				Keywords:    "rat",
				ShortDesc:   "a rat",
				LongDesc:    "A rat scurries.\r\n",
				Race:        15,
				ActionFlags: []string{"ISNPC"},
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

func makeMeditTestSession(t *testing.T, m *Manager, name string, level int) *Session {
	t.Helper()
	s := makeCommandTestSession(t, m, name, level, 3000)
	// Builders at LVL_SET_BUILD (35) bypass the zone check; use a plain
	// builder confined to zone 30 for permission tests.
	s.olcZone = 30
	return s
}

func TestMeditRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["medit"]
	if !ok {
		t.Fatal("medit command has no C gate")
	}
	// C interpreter.c:553: { "medit", POS_DEAD, do_olc, LVL_BUILDER, SCMD_OLC_MEDIT }
	if gate.MinLevel != 31 || gate.MinPosition != 0 {
		t.Fatalf("medit gate = (%d,%d), want (31,0)", gate.MinLevel, gate.MinPosition)
	}
	if _, ok := cmdRegistry.Lookup("medit"); !ok {
		t.Fatal("medit command is not registered")
	}
}

func TestMeditEntryGates(t *testing.T) {
	w := makeMeditTestWorld(t)
	m := newTestManager(t, w, nil)

	builder := makeMeditTestSession(t, m, "Meditbuilder", 31)

	// No argument.
	if err := cmdMedit(builder, nil); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); got != "Specify a mobile VNUM to edit.\r\n" {
		t.Fatalf("no-arg = %q", got)
	}

	// Nonnumeric argument.
	if err := cmdMedit(builder, []string{"abc"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); got != "Yikes!  Stop that, someone will get hurt!\r\n" {
		t.Fatalf("nonnumeric = %q", got)
	}

	// Unknown zone (vnum 99999 is in no zone).
	if err := cmdMedit(builder, []string{"99999"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); got != "Sorry, there is no zone for that number!\r\n" {
		t.Fatalf("unknown-zone = %q", got)
	}

	// Permission: builder confined to zone 30 cannot edit zone 31's range.
	// (Zone 31 doesn't exist in the test world, so use a vnum in no zone —
	// instead test with a second zone.)
}

func TestMeditPermissionGate(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 3000, Name: "Zone 30 Room", Zone: 30}},
		Zones: []parser.Zone{
			{Number: 30, Name: "Zone Thirty", TopRoom: 3099},
			{Number: 31, Name: "Zone Thirty-One", TopRoom: 3199},
		},
		Mobs:      []parser.Mob{{VNum: 3101, Keywords: "z31 mob", ActionFlags: []string{"ISNPC"}}},
		SourceDir: t.TempDir(),
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	m := newTestManager(t, w, nil)

	// Plain builder (level 31) assigned to zone 30.
	builder := makeCommandTestSession(t, m, "Permbuilder", 31, 3000)
	builder.olcZone = 30
	if err := cmdMedit(builder, []string{"3101"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, builder); got != "You do not have permission to edit this zone.\r\n" {
		t.Fatalf("permission = %q", got)
	}

	// LVL_SET_BUILD (35) bypasses the zone check.
	admin := makeCommandTestSession(t, m, "Permadmin", 35, 3000)
	admin.olcZone = 30
	if err := cmdMedit(admin, []string{"3101"}); err != nil {
		t.Fatal(err)
	}
	// Should enter the editor (menu output), not the permission error.
	got := readMsgText(t, admin)
	if !strings.HasPrefix(got, "\r\n-- Mob Number:  [3101]") {
		t.Fatalf("admin edit = %q", got)
	}
	admin.cancelMedit()
}

func TestMeditDuplicateGate(t *testing.T) {
	w := makeMeditTestWorld(t)
	m := newTestManager(t, w, nil)

	first := makeMeditTestSession(t, m, "Firstedit", 35)
	m.mu.Lock()
	m.sessions["Firstedit"] = first
	m.mu.Unlock()
	if err := cmdMedit(first, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	readMsgText(t, first) // drain menu

	second := makeMeditTestSession(t, m, "Secondedit", 35)
	m.mu.Lock()
	m.sessions["Secondedit"] = second
	m.mu.Unlock()
	if err := cmdMedit(second, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, second); got != "That mobile is currently being edited by Firstedit.\r\n" {
		t.Fatalf("duplicate = %q", got)
	}
	first.cancelMedit()
}

func TestMeditNewMobDefaults(t *testing.T) {
	w := makeMeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeMeditTestSession(t, m, "Newmobedit", 35)

	// 3050 has no prototype: new-mobile defaults.
	if err := cmdMedit(s, []string{"3050"}); err != nil {
		t.Fatal(err)
	}
	menu := readMsgText(t, s)

	if !strings.Contains(menu, "-- Mob Number:  [3050]") {
		t.Fatalf("menu missing vnum: %q", menu)
	}
	// C medit_setup_new: "mob unfinished", RACE_OTHER (16) = "Other".
	if !strings.Contains(menu, "2) Alias: mob unfinished") {
		t.Fatalf("menu missing alias: %q", menu)
	}
	if !strings.Contains(menu, "N) Race      : Other") {
		t.Fatalf("menu missing race Other: %q", menu)
	}
	// init_mobile: 1d1 HP, 1d1 damage, AC 100 (clear_char), standing.
	if !strings.Contains(menu, "C) Num HP Dice: [   1],  D) Size HP Dice: [   1]") {
		t.Fatalf("menu missing HP dice: %q", menu)
	}
	if !strings.Contains(menu, "F) Armor Class: [ 100]") {
		t.Fatalf("menu missing AC: %q", menu)
	}

	s.textEditMu.Lock()
	state := s.mobEdit
	s.textEditMu.Unlock()
	if state == nil || !state.isNew {
		t.Fatal("new mob not marked isNew")
	}
	if state.mob.LongDesc != "An unfinished mob stands here.\r\n" {
		t.Fatalf("long desc = %q", state.mob.LongDesc)
	}
	if len(state.mob.ActionFlags) != 1 || state.mob.ActionFlags[0] != "ISNPC" {
		t.Fatalf("action flags = %v", state.mob.ActionFlags)
	}
	s.cancelMedit()
}

func TestMeditFieldEditing(t *testing.T) {
	w := makeMeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeMeditTestSession(t, m, "Fieldedit", 35)

	if err := cmdMedit(s, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	readMsgText(t, s) // drain menu

	// Change alias (choice 2 → text prompt → new value → menu).
	s.handleMeditInput("2")
	if got := readMsgText(t, s); got != "\r\nEnter new text :\r\n| " {
		t.Fatalf("alias prompt = %q", got)
	}
	s.handleMeditInput("goblin sentry")
	menu := readMsgText(t, s)
	if !strings.Contains(menu, "2) Alias: goblin sentry") {
		t.Fatalf("alias not updated: %q", menu)
	}

	// Change sex (choice 1 → sex menu → 2 = Female).
	s.handleMeditInput("1")
	sexMenu := readMsgText(t, s)
	if !strings.Contains(sexMenu, " 2) Female") {
		t.Fatalf("sex menu = %q", sexMenu)
	}
	s.handleMeditInput("2")
	menu = readMsgText(t, s)
	if !strings.Contains(menu, "1) Sex: Female") {
		t.Fatalf("sex not updated: %q", menu)
	}

	// Numerical gate: empty input at a numerical mode.
	s.handleMeditInput("6") // level
	readMsgText(t, s)       // drain "Enter new value"
	s.handleMeditInput("")
	if got := readMsgText(t, s); got != "Field must be numerical, try again : " {
		t.Fatalf("numerical gate = %q", got)
	}

	// Level 12: derived stats (C float division: 12/1.5=8).
	s.handleMeditInput("12")
	menu = readMsgText(t, s)
	if !strings.Contains(menu, "6) Level:       [  12]") {
		t.Fatalf("level not updated: %q", menu)
	}
	s.textEditMu.Lock()
	mob := s.mobEdit.mob
	s.textEditMu.Unlock()
	if mob.Damage.Num != 8 || mob.Damage.Sides != 4 {
		t.Fatalf("level-12 damage dice = %dd%d, want 8d4", mob.Damage.Num, mob.Damage.Sides)
	}
	if mob.Exp != 9000 { // EXP_LOOKUP[12]
		t.Fatalf("level-12 exp = %d, want 9000", mob.Exp)
	}
	if mob.AC != 100-120 {
		t.Fatalf("level-12 AC = %d, want -20", mob.AC)
	}

	s.cancelMedit()
}

func TestMeditSaveAndDiscard(t *testing.T) {
	w := makeMeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeMeditTestSession(t, m, "Saveedit", 35)

	if err := cmdMedit(s, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	readMsgText(t, s)

	// Make a change, then quit → confirm prompt.
	s.handleMeditInput("2")
	readMsgText(t, s)
	s.handleMeditInput("changed alias")
	readMsgText(t, s)
	s.handleMeditInput("q")
	if got := readMsgText(t, s); got != "Do you wish to save the changes to the mobile? (y/n) : " {
		t.Fatalf("quit confirm = %q", got)
	}

	// Invalid choice → "Invalid choice!" + re-prompt (two messages).
	s.handleMeditInput("maybe")
	got1 := readMsgText(t, s)
	got2 := readMsgText(t, s)
	if got1 != "Invalid choice!\r\n" || got2 != "Do you wish to save the mobile? : " {
		t.Fatalf("invalid choice = %q %q", got1, got2)
	}

	// Discard with 'n': live prototype unchanged.
	s.handleMeditInput("n")
	if live, ok := w.SnapshotMob(3001); !ok || live.Keywords != "goblin guard" {
		t.Fatalf("discard changed live mob: %+v", live)
	}

	// Re-edit, change, save with 'y'.
	if err := cmdMedit(s, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	readMsgText(t, s)
	s.handleMeditInput("2")
	readMsgText(t, s)
	s.handleMeditInput("saved alias")
	readMsgText(t, s)
	s.handleMeditInput("q")
	readMsgText(t, s)
	s.handleMeditInput("y")
	if got := readMsgText(t, s); got != "Saving mobile to memory.\r\n" {
		t.Fatalf("save confirm = %q", got)
	}
	if live, ok := w.SnapshotMob(3001); !ok || live.Keywords != "saved alias" {
		t.Fatalf("save did not commit: %+v", live)
	}
}

func TestMeditCleanQuit(t *testing.T) {
	w := makeMeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeMeditTestSession(t, m, "Cleanquit", 35)

	if err := cmdMedit(s, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	readMsgText(t, s)
	// No changes → 'q' exits silently (no confirm prompt).
	s.handleMeditInput("q")
	// No output expected; session should be closed.
	s.textEditMu.Lock()
	closed := s.mobEdit == nil
	s.textEditMu.Unlock()
	if !closed {
		t.Fatal("clean quit did not close the session")
	}
}

func TestMeditScriptMenuNewMobRejected(t *testing.T) {
	w := makeMeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeMeditTestSession(t, m, "Scriptnew", 35)

	if err := cmdMedit(s, []string{"3050"}); err != nil {
		t.Fatal(err)
	}
	readMsgText(t, s)
	s.handleMeditInput("s")
	got := readMsgText(t, s)
	if !strings.Contains(got, "Cannot assign a script until the mob is saved at least once.\r\n") {
		t.Fatalf("new-mob script = %q", got)
	}
	s.cancelMedit()
}

func TestMeditScriptShallowBehavior(t *testing.T) {
	w := makeMeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeMeditTestSession(t, m, "Scriptshallow", 35)

	if err := cmdMedit(s, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	readMsgText(t, s)

	// Open script menu, set name.
	s.handleMeditInput("s")
	scriptMenu := readMsgText(t, s)
	if !strings.Contains(scriptMenu, "1) Name: goblin_script") {
		t.Fatalf("script menu = %q", scriptMenu)
	}
	s.handleMeditInput("1")
	if got := readMsgText(t, s); got != "Enter script name: " {
		t.Fatalf("script name prompt = %q", got)
	}
	s.handleMeditInput("new_script")
	readMsgText(t, s) // script menu redisplay

	// C shallow-copies the script pointer: the LIVE prototype sees the name
	// immediately, even before the mob is saved.
	if live, ok := w.SnapshotMob(3001); !ok || live.ScriptName != "new_script" {
		t.Fatalf("shallow script name not live: %+v", live)
	}

	// Toggle a script flag (2 = BRIBE).
	s.handleMeditInput("2")
	flagsMenu := readMsgText(t, s)
	if !strings.Contains(flagsMenu, "Enter script flags (0 to quit) : ") {
		t.Fatalf("script flags menu = %q", flagsMenu)
	}
	s.handleMeditInput("2") // toggle BRIBE (bit 1)
	readMsgText(t, s)
	if live, ok := w.SnapshotMob(3001); !ok || live.LuaFunctions != (4^2) {
		t.Fatalf("shallow script flags not live: %+v", live)
	}

	// Quit without saving the mob: script changes persist (C behavior).
	s.handleMeditInput("0") // back to script menu
	readMsgText(t, s)
	s.handleMeditInput("0") // back to main menu
	readMsgText(t, s)
	s.handleMeditInput("q")
	readMsgText(t, s)
	s.handleMeditInput("n") // discard mob changes
	if live, ok := w.SnapshotMob(3001); !ok || live.ScriptName != "new_script" {
		t.Fatalf("script name reverted on discard: %+v", live)
	}
}

func TestMeditDisconnectCleanup(t *testing.T) {
	w := makeMeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeMeditTestSession(t, m, "Disconnectedit", 35)

	if err := cmdMedit(s, []string{"3001"}); err != nil {
		t.Fatal(err)
	}
	readMsgText(t, s)
	s.handleMeditInput("2")
	readMsgText(t, s)
	s.handleMeditInput("unsaved change")
	readMsgText(t, s)

	// Disconnect discards without saving.
	s.cancelMedit()
	if live, ok := w.SnapshotMob(3001); !ok || live.Keywords != "goblin guard" {
		t.Fatalf("disconnect saved changes: %+v", live)
	}
	s.textEditMu.Lock()
	closed := s.mobEdit == nil
	writing := s.player.GetFlags()&(1<<uint(game.PlrWriting)) != 0
	s.textEditMu.Unlock()
	if !closed || writing {
		t.Fatal("disconnect did not clean up session state")
	}
}

func TestMeditSaveCommand(t *testing.T) {
	w := makeMeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeMeditTestSession(t, m, "Zonesave", 35)

	// "medit save <zone>": zone 30 → number = 3000.
	if err := cmdMedit(s, []string{"save", "30"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "Saving all mobiles in zone.\r\n" {
		t.Fatalf("zone save = %q", got)
	}

	// "medit save" without a zone number.
	if err := cmdMedit(s, []string{"save"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "Save which zone?\r\n" {
		t.Fatalf("save no-arg = %q", got)
	}
}

func TestMeditFlagToggle(t *testing.T) {
	w := makeMeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeMeditTestSession(t, m, "Flagtoggle", 35)

	if err := cmdMedit(s, []string{"3002"}); err != nil {
		t.Fatal(err)
	}
	readMsgText(t, s)

	// NPC flags: mob 3002 has ISNPC. Toggle 6 (AGGR).
	s.handleMeditInput("l")
	flagsMenu := readMsgText(t, s)
	if !strings.Contains(flagsMenu, " 6) AGGR") {
		t.Fatalf("npc flags menu = %q", flagsMenu)
	}
	s.handleMeditInput("6")
	readMsgText(t, s) // flags menu redisplay
	s.textEditMu.Lock()
	hasAggr := false
	for _, f := range s.mobEdit.mob.ActionFlags {
		if f == "AGGRESSIVE" {
			hasAggr = true
		}
	}
	s.textEditMu.Unlock()
	if !hasAggr {
		t.Fatal("AGGR flag not set after toggle")
	}

	// Toggle again → removed.
	s.handleMeditInput("6")
	readMsgText(t, s)
	s.textEditMu.Lock()
	hasAggr = false
	for _, f := range s.mobEdit.mob.ActionFlags {
		if f == "AGGRESSIVE" {
			hasAggr = true
		}
	}
	s.textEditMu.Unlock()
	if hasAggr {
		t.Fatal("AGGR flag not cleared after second toggle")
	}

	// 0 → back to main menu.
	s.handleMeditInput("0")
	menu := readMsgText(t, s)
	if !strings.HasPrefix(menu, "\r\n-- Mob Number:  [3002]") {
		t.Fatalf("flag quit != main menu: %q", menu)
	}
	s.cancelMedit()
}
