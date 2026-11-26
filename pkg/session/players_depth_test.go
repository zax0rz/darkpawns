package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCmdPlayersUsesCPlayerTableShapeWithoutDatabase(t *testing.T) {
	m := makeTestManager(t)
	var actor *Session
	for _, name := range []string{"Alpha", "Bravo", "Charlie", "Playersdepth"} {
		level := 1
		if name == "Playersdepth" {
			level = LVL_IMPL
		}
		// The real no-DB creation path builds characters at ID 0
		// (NewCharacterWithStats(0, ...) in char_creation.go) and AddPlayer
		// assigns distinct IDs in insertion order — the in-memory equivalent
		// of create_entry appending pfile records. The shared command-test
		// helper hardcodes ID 1, which would leave every player tied and the
		// creation-order sort a no-op, so build the real shape here.
		s := m.NewSession()
		s.player = game.NewPlayer(0, name, 1001)
		s.player.Level = level
		s.playerName = name
		s.authenticated = true
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
