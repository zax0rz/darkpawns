package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// src/limits.c:438-443 moves to world[3], then src/comm.c:2131-2133
// broadcasts link loss there. Room numbers need not equal their indexes.
func TestIdleForceRentUsesRoomIndexThree(t *testing.T) {
	for _, rooms := range [][]parser.Room{
		{{VNum: 0}, {VNum: 1}, {VNum: 3}, {VNum: 4}, {VNum: 1001}},
		{{VNum: 1001}, {VNum: 4}, {VNum: 3}, {VNum: 1}, {VNum: 0}},
		{{VNum: 0}, {VNum: 1}, {VNum: 3}, {VNum: 40}, {VNum: 1001}},
	} {
		w, err := NewWorld(&parser.World{Rooms: rooms})
		if err != nil {
			t.Fatal(err)
		}
		w.StopAITicker()
		actor := NewPlayer(1, "Idler", 1)
		actor.SetLevel(1)
		actor.IdleTimer = IDLE_DISCONNECT
		actor.WasInRoom = 1001
		destination := 4
		if rooms[3].VNum == 40 {
			destination = 40
		}
		correct := NewPlayer(2, "Correct", destination)
		wrong := NewPlayer(3, "Wrong", 3)
		for _, p := range []*Player{actor, correct, wrong} {
			if err := w.AddPlayer(p); err != nil {
				t.Fatal(err)
			}
		}
		output := map[string]string{}
		w.MessageSink = func(name string, msg []byte) { output[name] += string(msg) }
		w.CheckIdling(actor)
		if actor.GetRoom() != destination {
			t.Fatalf("idle destination=%d, want world[3] VNUM %d", actor.GetRoom(), destination)
		}
		if !strings.Contains(output["Correct"], "Idler has lost") {
			t.Fatalf("RNUM 3 peer missing link loss: %q", output["Correct"])
		}
		if strings.Contains(output["Wrong"], "has lost") {
			t.Fatalf("VNUM 3 peer received link loss: %q", output["Wrong"])
		}
	}
}
