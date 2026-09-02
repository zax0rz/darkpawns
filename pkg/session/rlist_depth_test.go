package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestRlistRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["rlist"]
	if !ok {
		t.Fatal("rlist command has no C gate")
	}
	if gate.MinLevel != LVL_IMMORT || gate.MinPosition != combat.PosDead {
		t.Fatalf("rlist gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_IMMORT, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("rlist")
	if !ok {
		t.Fatal("rlist command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("rlist registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestCmdRlistUsesNumericZoneAndCOutput(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Rlistadmin", 1001, true)
	s.player.Level = LVL_IMMORT

	if err := cmdRlist(s, []string{"1", "ignored", "words"}); err != nil {
		t.Fatalf("cmdRlist: %v", err)
	}
	want := "  1. [ 1001] Room A\r\n  2. [ 1002] Room B\r\n"
	if got := readSessionText(t, s); got != want {
		t.Fatalf("rlist output = %q, want %q", got, want)
	}
}

func TestCmdRlistUsesCDecimalPrefixAndNoZoneMessage(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Rlistadmin", 1001, true)
	s.player.Level = LVL_IMMORT

	if err := cmdRlist(s, []string{"999abc"}); err != nil {
		t.Fatalf("cmdRlist decimal prefix: %v", err)
	}
	if got, want := readSessionText(t, s), "The desired zone does not exist.\r\n"; got != want {
		t.Fatalf("decimal-prefix output = %q, want %q", got, want)
	}

	if err := cmdRlist(s, nil); err != nil {
		t.Fatalf("cmdRlist no argument: %v", err)
	}
	if got, want := readSessionText(t, s), "The desired zone does not exist.\r\n"; got != want {
		t.Fatalf("no-argument output = %q, want %q", got, want)
	}
}

func TestCmdRlistTruncatesAtCBufferLimit(t *testing.T) {
	rooms := make([]parser.Room, 200)
	for i := range rooms {
		rooms[i] = parser.Room{
			VNum: 1000 + i,
			Name: strings.Repeat("long room name ", 40),
			Zone: 1,
		}
	}
	w, err := game.NewWorld(&parser.World{Rooms: rooms})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	s := makeTestSession(t, m, "Rlistadmin", 1001, true)
	s.player.Level = LVL_IMMORT

	if err := cmdRlist(s, []string{"1"}); err != nil {
		t.Fatalf("cmdRlist overflow: %v", err)
	}
	if got, want := readSessionText(t, s), "Truncating room list due to size.\r\n"; got != want {
		t.Fatalf("overflow warning = %q, want %q", got, want)
	}
}
