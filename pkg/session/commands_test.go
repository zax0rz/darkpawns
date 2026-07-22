package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// makeCommandTestSession creates a Session suitable for calling ExecuteCommand
// in tests. It uses Manager.NewSession so the limiter and context are initialized.
func makeCommandTestSession(t *testing.T, m *Manager, name string, level, roomVNum int) *Session {
	t.Helper()
	s := m.NewSession()
	s.player = game.NewPlayer(1, name, roomVNum)
	s.player.Level = level
	s.playerName = name
	s.authenticated = true
	return s
}

// TestGrabAndHoldKeepDistinctCGates verifies the two C rows stay distinct even
// though both words share the same handler. C gives grab level 0 and hold 1.
func TestGrabAndHoldKeepDistinctCGates(t *testing.T) {
	holdEntry, ok := cmdRegistry.Lookup("hold")
	if !ok {
		t.Fatal("'hold' command not found in registry")
	}

	grabEntry, ok := cmdRegistry.Lookup("grab")
	if !ok {
		t.Fatal("'grab' command not found in registry")
	}

	if grabEntry == holdEntry {
		t.Fatal("grab and hold share one entry, so their distinct C gates cannot be represented")
	}
	if grabEntry.MinLevel != 0 || holdEntry.MinLevel != 1 {
		t.Errorf("grab/hold levels = %d/%d, want 0/1", grabEntry.MinLevel, holdEntry.MinLevel)
	}
	if grabEntry.MinPosition != combat.PosResting || holdEntry.MinPosition != combat.PosResting {
		t.Errorf("grab/hold positions = %d/%d, want resting", grabEntry.MinPosition, holdEntry.MinPosition)
	}
}

func TestReekCommandRegistrations(t *testing.T) {
	tests := []struct {
		name        string
		minLevel    int
		minPosition int
	}{
		{name: "detect", minLevel: 0, minPosition: combat.PosStanding},
		{name: "mold", minLevel: LVL_IMMORT, minPosition: combat.PosResting},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, ok := cmdRegistry.Lookup(tt.name)
			if !ok {
				t.Fatalf("%q command not found in registry", tt.name)
			}
			if entry.Name != tt.name {
				t.Fatalf("%q resolved to primary command %q", tt.name, entry.Name)
			}
			if entry.MinLevel != tt.minLevel {
				t.Errorf("%q MinLevel = %d, want %d", tt.name, entry.MinLevel, tt.minLevel)
			}
			if entry.MinPosition != tt.minPosition {
				t.Errorf("%q MinPosition = %d, want %d", tt.name, entry.MinPosition, tt.minPosition)
			}
		})
	}
}

func TestR2CommandSurfaceRegistrationsMatchC(t *testing.T) {
	tests := []struct {
		name        string
		minLevel    int
		minPosition int
	}{
		{name: "grats", minLevel: 0, minPosition: combat.PosSleeping},
		{name: ".", minLevel: 0, minPosition: combat.PosSleeping},
		{name: ":", minLevel: 1, minPosition: combat.PosResting},
		{name: ";", minLevel: 0, minPosition: combat.PosDead},
		{name: "?", minLevel: 0, minPosition: combat.PosDead},
		{name: "'", minLevel: 0, minPosition: combat.PosResting},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, ok := cmdRegistry.Lookup(tt.name)
			if !ok {
				t.Fatalf("%q command not found in registry", tt.name)
			}
			if entry.Name != tt.name {
				t.Errorf("%q resolved to primary command %q", tt.name, entry.Name)
			}
			if entry.MinLevel != tt.minLevel {
				t.Errorf("%q MinLevel = %d, want %d", tt.name, entry.MinLevel, tt.minLevel)
			}
			if entry.MinPosition != tt.minPosition {
				t.Errorf("%q MinPosition = %d, want %d", tt.name, entry.MinPosition, tt.minPosition)
			}
		})
	}

	if _, ok := cmdRegistry.Lookup("gratz"); ok {
		t.Error("Go-only player command \"gratz\" remains registered; C only exposes \"grats\"")
	}
}

func TestSplitCommandInputMatchesCNonLetterRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCmd  string
		wantArgs []string
	}{
		{name: "apostrophe attached", input: "'hello", wantCmd: "'", wantArgs: []string{"hello"}},
		{name: "apostrophe separated", input: "' hello", wantCmd: "'", wantArgs: []string{"hello"}},
		{name: "emote attached", input: ":grins broadly", wantCmd: ":", wantArgs: []string{"grins", "broadly"}},
		{name: "reply attached", input: ".hi", wantCmd: ".", wantArgs: []string{"hi"}},
		{name: "wiznet attached", input: ";test", wantCmd: ";", wantArgs: []string{"test"}},
		{name: "help alone", input: "?", wantCmd: "?", wantArgs: nil},
		{name: "leading whitespace", input: "\t  'hello", wantCmd: "'", wantArgs: []string{"hello"}},
		{name: "letter command", input: "say hello there", wantCmd: "say", wantArgs: []string{"hello", "there"}},
		{name: "blank line", input: " \t ", wantCmd: "", wantArgs: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCmd, gotArgs := splitCommandInput(tt.input)
			if gotCmd != tt.wantCmd {
				t.Errorf("command = %q, want %q", gotCmd, tt.wantCmd)
			}
			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("args = %#v, want %#v", gotArgs, tt.wantArgs)
			}
			for i := range tt.wantArgs {
				if gotArgs[i] != tt.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q", i, gotArgs[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestExecuteCommandAcceptsAttachedAndSeparatedSayShorthand(t *testing.T) {
	for _, input := range []string{"'hello", "' hello"} {
		t.Run(input, func(t *testing.T) {
			m := makeTestManager(t)
			s := makeTestSession(t, m, "Alice", 1001, true)
			s.player.Stats.Int = 10
			s.player.Stats.Wis = 10
			registerInWorld(t, s)

			if err := ExecuteCommand(s, input, nil); err != nil {
				t.Fatalf("ExecuteCommand(%q): %v", input, err)
			}
			if got := readSessionText(t, s); !strings.Contains(got, "You say 'hello'") {
				t.Errorf("ExecuteCommand(%q) output = %q, want say self-echo", input, got)
			}
		})
	}
}

// TestMinLevelBlocksMortal verifies that a command with MinLevel=LVL_IMMORT
// is rejected for a level-1 player with the classic "Huh?!?" message.
func TestMinLevelBlocksMortal(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "mortal", 1, 1001)

	var called bool
	name := "dp954testblock"
	cmdRegistry.Register(name, wrapArgs(func(s *Session, args []string) error {
		called = true
		return nil
	}), "dp954 test block", LVL_IMMORT, 0)

	if err := ExecuteCommand(s, name, nil); err != nil {
		t.Fatalf("ExecuteCommand returned error: %v", err)
	}

	if called {
		t.Error("handler was called for a mortal player")
	}

	text := readMsgText(t, s)
	if text != "Huh?!?\r\n" {
		t.Errorf("got message %q, want %q", text, "Huh?!?\r\n")
	}
}

// TestMinLevelAllowsImmortal verifies that a command with MinLevel=LVL_IMMORT
// runs its handler for an immortal player.
func TestMinLevelAllowsImmortal(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "immortal", LVL_IMMORT, 1001)

	var called bool
	name := "dp954testallow"
	cmdRegistry.Register(name, wrapArgs(func(s *Session, args []string) error {
		called = true
		return nil
	}), "dp954 test allow", LVL_IMMORT, 0)

	if err := ExecuteCommand(s, name, nil); err != nil {
		t.Fatalf("ExecuteCommand returned error: %v", err)
	}

	if !called {
		t.Error("handler was not called for an immortal player")
	}
}

// TestMinLevelZeroAllowsAll verifies that a command with MinLevel=0 runs for
// any player, including level 1.
func TestMinLevelZeroAllowsAll(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "newbie", 1, 1001)

	var called bool
	name := "dp954testzero"
	cmdRegistry.Register(name, wrapArgs(func(s *Session, args []string) error {
		called = true
		return nil
	}), "dp954 test zero", 0, 0)

	if err := ExecuteCommand(s, name, nil); err != nil {
		t.Fatalf("ExecuteCommand returned error: %v", err)
	}

	if !called {
		t.Error("handler was not called for a level-1 player on a MinLevel=0 command")
	}
}

// TestCommandRegistry_QABatch1 verifies the DP-1059 / DP-1060 registration
// fixes (BRIEF-2026-07-12-qa-batch1-oneliners.md):
//   - "search" is the canonical command word for do_detect at level 0 / standing
//     (C: interpreter.c binds "search" → do_detect).
//   - "detect" remains registered as an alias word so existing callers still resolve.
//   - "mold" is gated to LVL_IMMORT / PosResting (C: interpreter.c LVL_IMMORT, POS_RESTING).
func TestCommandRegistry_QABatch1(t *testing.T) {
	search, ok := cmdRegistry.Lookup("search")
	if !ok {
		t.Fatal("'search' command not found in registry — expected canonical do_detect word (DP-1060)")
	}
	if search.Name != "search" {
		t.Errorf("'search' resolved to primary %q, want 'search'", search.Name)
	}
	if search.MinLevel != 0 {
		t.Errorf("'search' MinLevel = %d, want 0", search.MinLevel)
	}
	if search.MinPosition != combat.PosStanding {
		t.Errorf("'search' MinPosition = %d, want PosStanding", search.MinPosition)
	}

	detect, ok := cmdRegistry.Lookup("detect")
	if !ok {
		t.Fatal("'detect' alias not found in registry — must remain resolvable (DP-1060)")
	}
	if detect.MinLevel != 0 {
		t.Errorf("'detect' MinLevel = %d, want 0", detect.MinLevel)
	}
	if detect.MinPosition != combat.PosStanding {
		t.Errorf("'detect' MinPosition = %d, want PosStanding", detect.MinPosition)
	}

	mold, ok := cmdRegistry.Lookup("mold")
	if !ok {
		t.Fatal("'mold' command not found in registry")
	}
	if mold.MinLevel != LVL_IMMORT {
		t.Errorf("'mold' MinLevel = %d, want LVL_IMMORT (%d) — mortal object creation must be gated (DP-1059)", mold.MinLevel, LVL_IMMORT)
	}
	if mold.MinPosition != combat.PosResting {
		t.Errorf("'mold' MinPosition = %d, want PosResting", mold.MinPosition)
	}
}
