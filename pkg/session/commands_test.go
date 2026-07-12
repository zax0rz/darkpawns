package session

import (
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

// TestGrabAliasResolvesToHold verifies that the 'grab' command is registered
// as an alias for 'hold' (DP-639).
func TestGrabAliasResolvesToHold(t *testing.T) {
	holdEntry, ok := cmdRegistry.Lookup("hold")
	if !ok {
		t.Fatal("'hold' command not found in registry")
	}

	grabEntry, ok := cmdRegistry.Lookup("grab")
	if !ok {
		t.Fatal("'grab' alias not found in registry")
	}

	if grabEntry != holdEntry {
		t.Error("'grab' does not resolve to the same command entry as 'hold'")
	}

	if grabEntry.Name != "hold" {
		t.Errorf("grab entry primary name = %q, want 'hold'", grabEntry.Name)
	}
}

func TestReekCommandRegistrations(t *testing.T) {
	tests := []struct {
		name        string
		minLevel    int
		minPosition int
	}{
		{name: "detect", minLevel: 0, minPosition: combat.PosStanding},
		{name: "mold", minLevel: 0, minPosition: combat.PosStanding},
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
