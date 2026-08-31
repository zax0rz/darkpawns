package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestGotoRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["goto"]
	if !ok {
		t.Fatal("goto command has no C gate")
	}
	if entry.MinLevel != LVL_IMMORT || entry.MinPosition != combat.PosSleeping {
		t.Fatalf("goto gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_IMMORT, combat.PosSleeping)
	}
}

func TestFindGotoRoomSkipsCFillWords(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "GotoWizard", 1001, true)
	s.player.Level = LVL_IMMORT

	room, ok := findGotoRoom(s, "the 1002 trailing words are ignored")
	if !ok || room != 1002 {
		t.Fatalf("findGotoRoom() = (%d, %t), want (1002, true)", room, ok)
	}
}

func TestGotoRoomAllowedHonorsCPrivateAndGodroomGates(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "GotoWizard", 1001, true)
	s.player.Level = LVL_IMMORT

	private := m.world.GetRoomInWorld(1002)
	private.Flags = []string{"512"}
	peer := game.NewPlayer(2, "PrivatePeer", 1002)
	peer2 := game.NewPlayer(3, "PrivatePeerTwo", 1002)
	if err := m.world.AddPlayer(peer); err != nil {
		t.Fatalf("add private peer: %v", err)
	}
	if err := m.world.AddPlayer(peer2); err != nil {
		t.Fatalf("add second private peer: %v", err)
	}
	if _, ok := gotoRoomAllowed(s, 1002); ok {
		t.Fatal("private room with an occupant should be rejected")
	}

	private.Flags = []string{"1024"}
	if _, ok := gotoRoomAllowed(s, 1002); ok {
		t.Fatal("god room should be rejected")
	}
}

func TestCmdAtPreservesNestedMovementLocation(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Wizard", 1001, true)
	s.player.Level = LVL_GRGOD

	if err := cmdAt(s, []string{"1002", "goto", "1001"}); err != nil {
		t.Fatalf("cmdAt failed: %v", err)
	}
	if got := s.player.GetRoom(); got != 1001 {
		t.Fatalf("nested movement ended in room %d, want 1001", got)
	}
}

func TestParseDigRoomNumberMatchesAtoiPrefix(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
		ok    bool
	}{
		{name: "decimal prefix", input: "1205-trailing", want: 1205, ok: true},
		{name: "plus sign", input: "+1205", want: 1205, ok: true},
		{name: "negative", input: "-1", want: -1, ok: true},
		{name: "nonnumeric", input: "room", want: 0, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseDigRoomNumber(tt.input)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parseDigRoomNumber(%q) = (%d, %t), want (%d, %t)", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParseDigArgumentsMatchesCFillWordParsing(t *testing.T) {
	direction, room := parseDigArguments([]string{"the", "N", "at", "1205", "trailing"})
	if direction != "n" || room != "1205" {
		t.Fatalf("parseDigArguments() = (%q, %q), want (%q, %q)", direction, room, "n", "1205")
	}
}

func TestDigDirectionsAcceptCSpellings(t *testing.T) {
	tests := map[string][2]string{
		"n": {"north", "south"},
		"E": {"east", "west"},
		"s": {"south", "north"},
		"W": {"west", "east"},
		"u": {"up", "down"},
		"D": {"down", "up"},
		"x": {"", ""},
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			gotDirection, gotReverse := digDirections(input)
			if gotDirection != want[0] || gotReverse != want[1] {
				t.Fatalf("digDirections(%q) = (%q, %q), want (%q, %q)", input, gotDirection, gotReverse, want[0], want[1])
			}
		})
	}
}
