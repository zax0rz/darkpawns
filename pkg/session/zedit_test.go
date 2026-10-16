package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func makeZeditTestWorld(t *testing.T) *game.World {
	t.Helper()
	worldPath := t.TempDir()
	if err := os.Mkdir(filepath.Join(worldPath, "zon"), 0o755); err != nil {
		t.Fatal(err)
	}
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 3000, Name: "Zedit Room", Zone: 30},
			{VNum: 3001, Name: "Other Room", Zone: 30},
		},
		Mobs: []parser.Mob{
			{VNum: 3002, ShortDesc: "a zedit mob"},
		},
		Objs: []parser.Obj{
			{VNum: 3003, ShortDesc: "a zedit object"},
			{VNum: 3004, ShortDesc: "a zedit container"},
		},
		Zones: []parser.Zone{{
			Number: 30, Name: "Zedit Zone", TopRoom: 3099, Lifespan: 30, ResetMode: 2,
			Commands: []parser.ZoneCommand{
				{Command: "M", Arg1: 3002, Arg2: 1, Arg3: 3000},
				{Command: "G", IfFlag: 1, Arg1: 3003, Arg2: 1},
				{Command: "*"},
				{Command: "O", Arg1: 3003, Arg2: 1, Arg3: 3001},
			},
		}},
		SourceDir: worldPath,
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.WorldPath = worldPath
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

func makeZeditTestSession(t *testing.T, m *Manager, name string, level int) *Session {
	t.Helper()
	s := makeCommandTestSession(t, m, name, level, 3000)
	s.olcZone = 30
	return s
}

func TestZeditRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["zedit"]
	if !ok {
		t.Fatal("zedit command has no C gate")
	}
	if gate.MinLevel != game.LVL_IMMORT || gate.MinPosition != combat.PosDead {
		t.Fatalf("zedit gate = (%d,%d), want (%d,%d)", gate.MinLevel, gate.MinPosition, game.LVL_IMMORT, combat.PosDead)
	}
	entry, ok := cmdRegistry.Lookup("zedit")
	if !ok {
		t.Fatal("zedit command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("zedit registry gate = (%d,%d), want (%d,%d)", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

// TestZeditNewZoneRequiresNumber pins src/olc.c:138 through the ZEDIT entry
// handler. A high god's incomplete "new" request gets C's exact prompt.
func TestZeditNewZoneRequiresNumber(t *testing.T) {
	w := makeZeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeZeditTestSession(t, m, "Zeditimpl", game.LVL_IMPL)

	if err := cmdZedit(s, []string{"new"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "Specify a new zone number.\r\n" {
		t.Fatalf("incomplete new-zone request = %q", got)
	}
}

// TestZeditMissingRoomRefusal pins src/olc.c:243 for a room number inside a
// real zone whose room record is absent.
func TestZeditMissingRoomRefusal(t *testing.T) {
	w := makeZeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeZeditTestSession(t, m, "Zeditimpl", game.LVL_IMPL)

	if err := cmdZedit(s, []string{"3005"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "That room does not exist.\r\n" {
		t.Fatalf("missing-room refusal = %q", got)
	}
}

func TestZeditEntryAndNoDirtyQuit(t *testing.T) {
	w := makeZeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeZeditTestSession(t, m, "Zeditbuilder", game.LVL_IMPL)

	if err := cmdZedit(s, nil); err != nil {
		t.Fatal(err)
	}
	menu := readMsgText(t, s)
	if !strings.HasPrefix(menu, "\r\nRoom number: 3000        Room zone: 30\r\nZ) Zone name   : Zedit Zone\r\n") {
		t.Fatalf("entry menu = %q", menu)
	}
	if got := s.zedit.zone.Commands; len(got) != 3 || got[0].Command != "M" || got[1].Command != "G" || got[2].Command != "*" {
		t.Fatalf("room-filtered commands = %+v", got)
	}

	s.handleZeditInput("q")
	if got, want := readMsgText(t, s), "No changes made.\r\n"; got != want {
		t.Fatalf("clean quit = %q, want %q", got, want)
	}
	if s.isZoneEditing() || s.player.GetFlags()&(1<<uint(game.PlrWriting)) != 0 {
		t.Fatal("clean quit left ZEDIT active")
	}
}

func TestZeditDuplicateGateAndCleanup(t *testing.T) {
	w := makeZeditTestWorld(t)
	m := newTestManager(t, w, nil)
	first := makeZeditTestSession(t, m, "Zeditfirst", game.LVL_IMPL)
	second := makeZeditTestSession(t, m, "Zeditsecond", game.LVL_IMPL)

	if err := cmdZedit(first, []string{"3000"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, first)
	if err := cmdZedit(second, []string{"3000"}); err != nil {
		t.Fatal(err)
	}
	if got, want := readMsgText(t, second), "That room is currently being edited by Zeditfirst (telnet; idle 0s).\r\n"; got != want {
		t.Fatalf("duplicate gate = %q, want %q", got, want)
	}
	first.handleZeditInput("q")
	if got, want := readMsgText(t, first), "No changes made.\r\n"; got != want {
		t.Fatalf("first quit = %q, want %q", got, want)
	}
	if err := cmdZedit(second, []string{"3000"}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(readMsgText(t, second), "\r\nRoom number: 3000") {
		t.Fatal("duplicate reservation was not released after clean quit")
	}
	second.handleZeditInput("q")
	_ = readMsgText(t, second)
}

func TestZeditMenuColorsAtCLevels(t *testing.T) {
	wantPlain := "\r\nRoom number: 3000        Room zone: 30\r\n" +
		"Z) Zone name   : Zedit Zone\r\n" +
		"L) Lifespan    : 30 minutes\r\n" +
		"T) Top of zone : 3099\r\n" +
		"R) Reset Mode  : Normal reset.\r\n" +
		"[Command list]\r\n" +
		"0 - Load a zedit mob [3002], Max : 1\r\n" +
		"1 -  then Give it a zedit object [3003], Max : 1\r\n" +
		"2 - <Unknown Command>\r\n" +
		"3 - <END OF LIST>\r\n" +
		"N) New command.\r\nE) Edit a command.\r\nD) Delete a command.\r\nQ) Quit\r\nEnter your choice : "
	for _, tc := range []struct {
		name   string
		color1 bool
		color2 bool
		want   string
	}{
		{name: "off", want: wantPlain},
		{name: "normal", color1: true, color2: true, want: "\r\nRoom number: \x1b[36m3000\x1b[0m        Room zone: \x1b[36m30\r\n" +
			"\x1b[32mZ\x1b[0m) Zone name   : \x1b[33mZedit Zone\r\n" +
			"\x1b[32mL\x1b[0m) Lifespan    : \x1b[33m30 minutes\r\n" +
			"\x1b[32mT\x1b[0m) Top of zone : \x1b[33m3099\r\n" +
			"\x1b[32mR\x1b[0m) Reset Mode  : \x1b[33mNormal reset.\x1b[0m\r\n" +
			"[Command list]\r\n" +
			"\x1b[0m0 - \x1b[33mLoad a zedit mob [\x1b[36m3002\x1b[33m], Max : 1\r\n" +
			"\x1b[0m1 - \x1b[33m then Give it a zedit object [\x1b[36m3003\x1b[33m], Max : 1\r\n" +
			"\x1b[0m2 - \x1b[33m<Unknown Command>\r\n" +
			"\x1b[0m3 - <END OF LIST>\r\n" +
			"\x1b[32mN\x1b[0m) New command.\r\n\x1b[32mE\x1b[0m) Edit a command.\r\n\x1b[32mD\x1b[0m) Delete a command.\r\n\x1b[32mQ\x1b[0m) Quit\r\nEnter your choice : "},
		{name: "complete", color1: true, color2: true, want: "\r\nRoom number: \x1b[36m3000\x1b[0m        Room zone: \x1b[36m30\r\n" +
			"\x1b[32mZ\x1b[0m) Zone name   : \x1b[33mZedit Zone\r\n" +
			"\x1b[32mL\x1b[0m) Lifespan    : \x1b[33m30 minutes\r\n" +
			"\x1b[32mT\x1b[0m) Top of zone : \x1b[33m3099\r\n" +
			"\x1b[32mR\x1b[0m) Reset Mode  : \x1b[33mNormal reset.\x1b[0m\r\n" +
			"[Command list]\r\n" +
			"\x1b[0m0 - \x1b[33mLoad a zedit mob [\x1b[36m3002\x1b[33m], Max : 1\r\n" +
			"\x1b[0m1 - \x1b[33m then Give it a zedit object [\x1b[36m3003\x1b[33m], Max : 1\r\n" +
			"\x1b[0m2 - \x1b[33m<Unknown Command>\r\n" +
			"\x1b[0m3 - <END OF LIST>\r\n" +
			"\x1b[32mN\x1b[0m) New command.\r\n\x1b[32mE\x1b[0m) Edit a command.\r\n\x1b[32mD\x1b[0m) Delete a command.\r\n\x1b[32mQ\x1b[0m) Quit\r\nEnter your choice : "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := makeZeditTestWorld(t)
			m := newTestManager(t, w, nil)
			s := makeZeditTestSession(t, m, "Zeditcolors", game.LVL_IMPL)
			s.player.SetPlrFlag(game.PrfColor1, tc.color1)
			s.player.SetPlrFlag(game.PrfColor2, tc.color2)
			if err := cmdZedit(s, []string{"3000"}); err != nil {
				t.Fatal(err)
			}
			got := readMsgText(t, s)
			if got != tc.want {
				t.Fatalf("menu = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestZeditStaleCarryAndCommandSave(t *testing.T) {
	w := makeZeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeZeditTestSession(t, m, "Zeditsave", game.LVL_IMPL)
	if err := cmdZedit(s, []string{"3000"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)

	// Append a new M command. New accepts pos == len(commands), while the
	// immediate command-type quiz starts with position 3 (if_flag applies).
	s.handleZeditInput("n")
	_ = readMsgText(t, s)
	s.handleZeditInput("3")
	_ = readMsgText(t, s)
	s.handleZeditInput("M")
	if got := readMsgText(t, s); got != "Is this command dependent on the success of the previous one? (y/n)\r\n" {
		t.Fatalf("if-flag prompt = %q", got)
	}
	s.handleZeditInput("n")
	if got := readMsgText(t, s); got != "Input mob's vnum : " {
		t.Fatalf("arg1 prompt = %q", got)
	}
	s.handleZeditInput("3002")
	if got := readMsgText(t, s); got != "Input the maximum number that can exist on the mud : " {
		t.Fatalf("mobile max-count prompt = %q", got)
	}
	s.handleZeditInput("2")
	menu := readMsgText(t, s)
	if !strings.Contains(menu, "3 - Load a zedit mob [3002], Max : 2\r\n") {
		t.Fatalf("new command menu = %q", menu)
	}

	s.handleZeditInput("q")
	if got := readMsgText(t, s); got != "Do you wish to save the changes to the zone info? (y/n) : " {
		t.Fatalf("save prompt = %q", got)
	}
	s.handleZeditInput("y")
	if got := readMsgText(t, s); got != "Saving zone info in memory.\r\n" {
		t.Fatalf("save result = %q", got)
	}

	zone, ok := w.SnapshotZone(30)
	if !ok {
		t.Fatal("zone disappeared")
	}
	if len(zone.Commands) != 4 || zone.Commands[0].Command != "M" || zone.Commands[1].Command != "G" || zone.Commands[2].Command != "M" || zone.Commands[3].Command != "O" {
		t.Fatalf("committed commands = %+v", zone.Commands)
	}

	if err := saveZeditZone(w, &zone); err != nil {
		t.Fatalf("saveZeditZone: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(w.WorldPath, "zon", "30.zon"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "M 0 3002 1 3000\n") || !strings.Contains(text, "G 1 3003 1 -1\n") || strings.Contains(text, "* ") {
		t.Fatalf("written zone = %q", text)
	}
}

// TestZeditObjectLoadPrompts pins src/zedit.c:740 and :776 through the real
// interactive parser: selecting O emits the object VNUM prompt, then the
// maximum-count prompt after accepting an existing object.
func TestZeditObjectLoadPrompts(t *testing.T) {
	w := makeZeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeZeditTestSession(t, m, "Zeditobject", game.LVL_IMPL)
	if err := cmdZedit(s, []string{"3000"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)

	s.handleZeditInput("n")
	_ = readMsgText(t, s)
	s.handleZeditInput("3")
	_ = readMsgText(t, s)
	s.handleZeditInput("O")
	if got := readMsgText(t, s); got != "Is this command dependent on the success of the previous one? (y/n)\r\n" {
		t.Fatalf("if-flag prompt = %q", got)
	}
	s.handleZeditInput("n")
	if got := readMsgText(t, s); got != "Input object vnum : " {
		t.Fatalf("object VNUM prompt = %q", got)
	}
	s.handleZeditInput("3003")
	if got := readMsgText(t, s); got != "Input the maximum number that can exist on the mud : " {
		t.Fatalf("object max-count prompt = %q", got)
	}
}

func TestZeditActivityOutputPrompts(t *testing.T) {
	newSession := func(t *testing.T, name string) *Session {
		t.Helper()
		w := makeZeditTestWorld(t)
		m := newTestManager(t, w, nil)
		s := makeZeditTestSession(t, m, name, game.LVL_IMPL)
		if err := cmdZedit(s, []string{"3000"}); err != nil {
			t.Fatal(err)
		}
		_ = readMsgText(t, s)
		return s
	}
	appendCommand := func(t *testing.T, s *Session, command string) {
		t.Helper()
		s.handleZeditInput("n")
		if got, want := readMsgText(t, s), "What number in the list should the new command be? : "; got != want {
			t.Fatalf("new-command prompt = %q, want %q", got, want)
		}
		s.handleZeditInput("3")
		_ = readMsgText(t, s)
		s.handleZeditInput(command)
		if got, want := readMsgText(t, s), "Is this command dependent on the success of the previous one? (y/n)\r\n"; got != want {
			t.Fatalf("if-flag prompt = %q, want %q", got, want)
		}
		s.handleZeditInput("n")
	}

	t.Run("remove selector and both vnum prompts", func(t *testing.T) {
		s := newSession(t, "Zeditremove-prompts")
		appendCommand(t, s, "R")
		if got, want := readMsgText(t, s), "Input 0 for a mobile, 1 for an object: "; got != want {
			t.Fatalf("remove selector = %q, want %q", got, want)
		}
		s.handleZeditInput("1")
		if got, want := readMsgText(t, s), "Input object's vnum : "; got != want {
			t.Fatalf("object-removal VNUM prompt = %q, want %q", got, want)
		}

		s = newSession(t, "Zeditremove-mobile")
		appendCommand(t, s, "R")
		_ = readMsgText(t, s)
		s.handleZeditInput("0")
		if got, want := readMsgText(t, s), "Input mobile's vnum : "; got != want {
			t.Fatalf("mobile-removal VNUM prompt = %q, want %q", got, want)
		}
	})

	t.Run("loop selector and start repeat prompt", func(t *testing.T) {
		s := newSession(t, "Zeditloop-prompts")
		appendCommand(t, s, "L")
		if got, want := readMsgText(t, s), "Input 0 for loop start, 1 for loop finish: "; got != want {
			t.Fatalf("loop selector = %q, want %q", got, want)
		}
		s.handleZeditInput("0")
		if got, want := readMsgText(t, s), "Input the number of times to repeat the loop : "; got != want {
			t.Fatalf("loop repeat prompt = %q, want %q", got, want)
		}
	})

	t.Run("equipment list and prompt", func(t *testing.T) {
		s := newSession(t, "Zeditequip-prompts")
		appendCommand(t, s, "E")
		if got, want := readMsgText(t, s), "Input object vnum : "; got != want {
			t.Fatalf("equipment object VNUM prompt = %q, want %q", got, want)
		}
		s.handleZeditInput("3003")
		if got, want := readMsgText(t, s), "Input the maximum number that can exist on the mud : "; got != want {
			t.Fatalf("equipment max-count prompt = %q, want %q", got, want)
		}
		s.handleZeditInput("1")
		got := readMsgText(t, s)
		var want strings.Builder
		equipment := []string{
			"Used as light", "Worn on right finger", "Worn on left finger", "First worn around Neck",
			"Second worn around Neck", "Worn on body", "Worn on head", "Worn on legs",
			"Worn on feet", "Worn on hands", "Worn on arms", "Worn as shield",
			"Worn about body", "Worn around waist", "Worn around right wrist", "Worn around left wrist",
			"Wielded", "Held", "Wielded for throwing", "Worn about legs", "Worn on face", "Hovering near head",
		}
		for i := 0; i < len(equipment); i += 2 {
			second := ""
			if i+1 < len(equipment) {
				second = equipment[i+1]
			}
			fmt.Fprintf(&want, "%2d) %26.26s %2d) %26.26s\r\n", i, equipment[i], i+1, second)
		}
		want.WriteString("Input location to equip : ")
		if got != want.String() {
			t.Fatalf("equipment choices = %q, want %q", got, want.String())
		}
	})

	t.Run("put-container prompt", func(t *testing.T) {
		s := newSession(t, "Zeditput-prompts")
		appendCommand(t, s, "P")
		if got, want := readMsgText(t, s), "Input object vnum : "; got != want {
			t.Fatalf("put object VNUM prompt = %q, want %q", got, want)
		}
		s.handleZeditInput("3003")
		if got, want := readMsgText(t, s), "Input the maximum number that can exist on the mud : "; got != want {
			t.Fatalf("put max-count prompt = %q, want %q", got, want)
		}
		s.handleZeditInput("1")
		if got, want := readMsgText(t, s), "Input the vnum of the container : "; got != want {
			t.Fatalf("container VNUM prompt = %q, want %q", got, want)
		}
	})

	for _, tc := range []struct {
		choice string
		want   string
	}{
		{choice: "e", want: "Which command do you wish to change? : "},
		{choice: "z", want: "Enter new zone name : "},
	} {
		t.Run("main menu "+tc.choice, func(t *testing.T) {
			s := newSession(t, "Zeditmenu-prompt")
			s.handleZeditInput(tc.choice)
			if got := readMsgText(t, s); got != tc.want {
				t.Fatalf("main-menu prompt = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestZeditActivityRetryOutputs(t *testing.T) {
	tests := []struct {
		name    string
		command string
		mode    zeditMode
		arg2    int
		input   string
		want    string
	}{
		{name: "command type", command: "*", mode: zeditCommandType, input: "?", want: "Invalid choice, try again : "},
		{name: "if flag", command: "M", mode: zeditIfFlag, input: "?", want: "Try again : "},
		{name: "nonnumeric arg1", command: "M", mode: zeditArg1, input: "x", want: "Must be a numeric value, try again : "},
		{name: "missing mobile", command: "M", mode: zeditArg1, input: "999999", want: "That mobile does not exist, try again : "},
		{name: "missing object", command: "O", mode: zeditArg1, input: "999999", want: "That object does not exist, try again : "},
		{name: "nonnumeric arg2", command: "M", mode: zeditArg2, input: "x", want: "Must be a numeric value, try again : "},
		{name: "door direction", command: "D", mode: zeditArg2, input: "9", want: "Try again : "},
		{name: "remove selector", command: "R", mode: zeditArg2, input: "9", want: "Try again : "},
		{name: "loop selector", command: "L", mode: zeditArg2, input: "9", want: "Try again : "},
		{name: "nonnumeric arg3", command: "E", mode: zeditArg3, input: "x", want: "Must be a numeric value, try again : "},
		{name: "equipment location", command: "E", mode: zeditArg3, input: "99", want: "Try again : "},
		{name: "missing container", command: "P", mode: zeditArg3, input: "999999", want: "That object does not exist, try again : "},
		{name: "door state", command: "D", mode: zeditArg3, input: "3", want: "Try again : "},
		{name: "zero loop repeat", command: "L", mode: zeditArg3, arg2: 0, input: "0", want: "The loop must repeat at least once, try again : "},
		{name: "missing object removal", command: "R", mode: zeditArg3, arg2: 1, input: "999999", want: "That object does not exist, try again : "},
		{name: "missing mobile removal", command: "R", mode: zeditArg3, arg2: 0, input: "999999", want: "That mobile does not exist, try again : "},
		{name: "lifespan", command: "", mode: zeditZoneLife, input: "241", want: "Try again (0-240) : "},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := makeZeditTestWorld(t)
			m := newTestManager(t, w, nil)
			s := makeZeditTestSession(t, m, "Zeditretry", game.LVL_IMPL)
			if err := cmdZedit(s, []string{"3000"}); err != nil {
				t.Fatal(err)
			}
			_ = readMsgText(t, s)
			state := s.zedit
			state.zone.Commands = []parser.ZoneCommand{{Command: tc.command, Arg1: 3000, Arg2: tc.arg2}}
			state.position = 0
			state.mode = tc.mode
			s.handleZeditInput(tc.input)
			if got := readMsgText(t, s); got != tc.want {
				t.Fatalf("retry output = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestZeditBoundaryQuirks(t *testing.T) {
	if got := zeditSetupCommands([]parser.ZoneCommand{
		{Command: "M", Arg3: 3000},
		{Command: "G", Arg1: 3003},
		{Command: "O", Arg3: 3001},
		{Command: "P", Arg3: 3004},
		{Command: "*"},
	}, 3000); len(got) != 2 || got[0].Command != "M" || got[1].Command != "G" {
		t.Fatalf("stale setup carry = %+v", got)
	}
	if got := zeditSetupCommands([]parser.ZoneCommand{{Command: "G", Arg1: 3003}}, 3000); len(got) != 0 {
		t.Fatalf("initial -1 stale carry = %+v", got)
	}

	w := makeZeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeZeditTestSession(t, m, "Zeditbounds", game.LVL_IMPL)
	if err := cmdZedit(s, []string{"3000"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleZeditInput("e")
	_ = readMsgText(t, s)
	s.handleZeditInput("0")
	_ = readMsgText(t, s)
	s.handleZeditInput("d")
	_ = readMsgText(t, s)
	s.handleZeditInput("3002")
	_ = readMsgText(t, s)
	s.handleZeditInput("6")
	_ = readMsgText(t, s)
	state := s.zedit
	if state.zone.Commands[0].Arg2 != 6 {
		t.Fatalf("direction terminator was rejected: %+v", state.zone.Commands[0])
	}
	state.zone.Commands = []parser.ZoneCommand{{Command: "L"}}
	state.position = 0
	state.mode = zeditArg2
	s.handleZeditInput("1")
	if got := readMsgText(t, s); !strings.Contains(got, "0 - ...End Repeat\r\n") || state.mode != zeditMainMenu || state.zone.Commands[0].Arg3 != -1 {
		t.Fatalf("loop finish path = %q, state=%+v", got, state)
	}
	s.handleZeditInput("0")
	_ = readMsgText(t, s)
	s.handleZeditInput("q")
	_ = readMsgText(t, s)
	s.handleZeditInput("n")
}

func TestZeditNewZoneTemplatesAndIndexInsertion(t *testing.T) {
	w := makeZeditTestWorld(t)
	for _, ext := range []string{"zon", "wld", "mob", "obj", "shp"} {
		directory := filepath.Join(w.WorldPath, ext)
		if ext != "zon" {
			if err := os.Mkdir(directory, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(directory, "index"), []byte("30."+ext+"\n$\n"), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	m := newTestManager(t, w, nil)
	s := makeZeditTestSession(t, m, "Zeditimpl", LVL_HIGOD)
	if err := cmdZedit(s, []string{"new", "31"}); err != nil {
		t.Fatal(err)
	}
	if got, want := readMsgText(t, s), "Zone created.\r\n"; got != want {
		t.Fatalf("new-zone result = %q, want %q", got, want)
	}
	zone, ok := w.SnapshotZone(31)
	if !ok || zone.Name != "New Zone" || zone.TopRoom != 3199 || zone.Lifespan != 30 || zone.ResetMode != 2 {
		t.Fatalf("new in-memory zone = %+v, present=%v", zone, ok)
	}
	checks := map[string]string{
		"zon/31.zon": "#31\nNew Zone~\n3199 30 2\nS\n$\n",
		"wld/31.wld": "#3100\nThe Begining~\nNot much here.\n~\n31 0 0\nS\n$\n",
		"mob/31.mob": "$\n",
		"obj/31.obj": "$\n",
		"shp/31.shp": "$~\n",
	}
	for relative, want := range checks {
		data, err := os.ReadFile(filepath.Join(w.WorldPath, relative))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != want {
			t.Errorf("%s = %q, want %q", relative, data, want)
		}
	}
	for _, ext := range []string{"zon", "wld", "mob", "obj", "shp"} {
		data, err := os.ReadFile(filepath.Join(w.WorldPath, ext, "index"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "30."+ext+"\n31."+ext+"\n$\n" {
			t.Errorf("%s index = %q", ext, data)
		}
	}
}

func TestZeditNewZoneRefusals(t *testing.T) {
	for _, tc := range []struct {
		name   string
		number string
		want   string
	}{
		{name: "maximum", number: "327", want: "326 is the highest zone allowed.\r\n"},
		{name: "overlap", number: "30", want: "A zone already covers that area.\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := makeZeditTestWorld(t)
			m := newTestManager(t, w, nil)
			s := makeZeditTestSession(t, m, "Zeditimpl", LVL_HIGOD)
			if err := cmdZedit(s, []string{"new", tc.number}); err != nil {
				t.Fatal(err)
			}
			if got := readMsgText(t, s); got != tc.want {
				t.Fatalf("new-zone refusal = %q, want %q", got, tc.want)
			}
		})
	}
}
