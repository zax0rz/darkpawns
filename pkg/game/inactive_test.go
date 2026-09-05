package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestDoInactiveToggleState(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	player := NewCharacter(1, "Inactivestate", ClassWarrior, RaceHuman)
	player.SetRoom(1001)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	w.ExecInactive(player, "ignored argument")
	if player.GetFlags()&(1<<PrfInactive) == 0 {
		t.Fatal("enabling inactive did not set the C preference bit")
	}

	w.ExecInactive(player, "ignored argument")
	if player.GetFlags()&(1<<PrfInactive) != 0 {
		t.Fatal("disabling inactive did not clear the C preference bit")
	}
}
