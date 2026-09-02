package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestRaceSayRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := cmdRegistry.Lookup("rsay")
	if !ok {
		t.Fatal("rsay command is not registered")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("rsay gate = (%d, %d), want (0, %d)", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}
}

func TestCmdRaceSayRunsCPrechecksForEmptyArguments(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Rsaystupid", 1, 1001)
	s.player.Stats.Int = 0
	s.player.Stats.Wis = 10
	registerInWorld(t, s)

	if err := cmdRaceSay(s, nil); err != nil {
		t.Fatalf("cmdRaceSay: %v", err)
	}
	if got, want := readMsgText(t, s), "You are too stupid to communicate with language!\r\n"; got != want {
		t.Fatalf("empty precheck output = %q, want %q", got, want)
	}

	s.player.Stats.Int = 10
	s.player.SetPlrFlag(game.PlrNoshout, true)
	if err := cmdRaceSay(s, nil); err != nil {
		t.Fatalf("cmdRaceSay noshout: %v", err)
	}
	if got, want := readMsgText(t, s), "You cannot race-say!\r\n"; got != want {
		t.Fatalf("empty noshout output = %q, want %q", got, want)
	}
}
