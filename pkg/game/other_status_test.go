package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestDoAFKToggleState(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	player := NewCharacter(1, "Afkstate", ClassWarrior, RaceHuman)
	player.SetRoom(1001)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	w.ExecAFK(player, "ignored argument")
	if !player.GetAFK() || player.GetFlags()&(1<<PrfAFK) == 0 {
		t.Fatal("enabling AFK did not set both the typed state and C preference bit")
	}
	if player.GetAFKMessage() != "" {
		t.Fatalf("AFK message = %q, want empty C-compatible message", player.GetAFKMessage())
	}

	w.ExecAFK(player, "ignored argument")
	if player.GetAFK() || player.GetFlags()&(1<<PrfAFK) != 0 {
		t.Fatal("disabling AFK did not clear both the typed state and C preference bit")
	}
}
