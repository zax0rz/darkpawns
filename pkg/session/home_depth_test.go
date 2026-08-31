package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCmdHomeNumericSetsLoadRoomWithoutMoving(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Homekeeper", game.LVL_IMMORT, 1001)
	registerInWorld(t, s)

	if err := cmdHome(s, []string{"1002", "trailing", "words"}); err != nil {
		t.Fatalf("cmdHome: %v", err)
	}
	if got := s.player.GetRoom(); got != 1001 {
		t.Fatalf("home setter moved player to room %d, want current room 1001", got)
	}
	if got := s.player.GetLoadRoom(); got != 1002 {
		t.Fatalf("load room = %d, want 1002", got)
	}
	if s.player.GetFlags()&(1<<uint(game.PlrLoadroom)) == 0 {
		t.Fatal("home setter did not set PLR_LOADROOM")
	}
	if got := readSessionText(t, s); got != "Home room set to 1002.\n" {
		t.Fatalf("home setter output = %q, want C line ending before telnet framing", got)
	}
}
