package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestPlayersRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["players"]
	if !ok {
		t.Fatal("players command has no C gate")
	}
	if gate.MinLevel != LVL_GRGOD || gate.MinPosition != combat.PosDead {
		t.Fatalf("players gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_GRGOD, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("players")
	if !ok {
		t.Fatal("players command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("players registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestCmdPlayersUsesCPlayerTableShapeWithoutDatabase(t *testing.T) {
	m := makeTestManager(t)
	var actor *Session
	for _, name := range []string{"Alpha", "Bravo", "Charlie", "Playersdepth"} {
		level := 1
		if name == "Playersdepth" {
			level = LVL_IMPL
		}
		s := makeCommandTestSession(t, m, name, level, 1001)
		registerInWorld(t, s)
		if name == "Playersdepth" {
			actor = s
		}
	}

	if err := cmdPlayers(actor, nil); err != nil {
		t.Fatalf("cmdPlayers: %v", err)
	}
	if got, want := readMsgText(t, actor), "A list of registered players:\r\nalpha               bravo               charlie             \r\nplayersdepth        \r\n"; got != want {
		t.Fatalf("players output = %q, want %q", got, want)
	}
}
