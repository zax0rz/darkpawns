package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestCOrderListOutput_BuildRoomMobs verifies R5h for Session.buildRoomMobs:
// three mobs entered in known order must appear in C's world[room].people
// newest-first order, 50/50 times.
func TestCOrderListOutput_BuildRoomMobs(t *testing.T) {
	const roomVNum = 9950
	for iter := range 50 {
		w, err := game.NewWorld(&parser.World{
			Rooms: []parser.Room{{VNum: roomVNum, Name: "Test Room", Zone: 1}},
			Mobs: []parser.Mob{
				{VNum: 9951, Keywords: "guard", ShortDesc: "Guard Alpha", DefaultPos: 8, Position: 8},
				{VNum: 9952, Keywords: "guard", ShortDesc: "Guard Beta", DefaultPos: 8, Position: 8},
				{VNum: 9953, Keywords: "guard", ShortDesc: "Guard Gamma", DefaultPos: 8, Position: 8},
			},
		})
		if err != nil {
			t.Fatalf("iter %d: NewWorld: %v", iter, err)
		}
		t.Cleanup(w.StopAITicker)

		if _, err := w.SpawnMob(9951, roomVNum); err != nil {
			t.Fatalf("iter %d: spawn 9951: %v", iter, err)
		}
		if _, err := w.SpawnMob(9952, roomVNum); err != nil {
			t.Fatalf("iter %d: spawn 9952: %v", iter, err)
		}
		if _, err := w.SpawnMob(9953, roomVNum); err != nil {
			t.Fatalf("iter %d: spawn 9953: %v", iter, err)
		}

		p := game.NewPlayer(1, "Agent", roomVNum)
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("iter %d: AddPlayer: %v", iter, err)
		}
		mgr := &Manager{world: w}
		s := &Session{manager: mgr, player: p}

		got := s.buildRoomMobs()
		if len(got) != 3 {
			t.Fatalf("iter %d: got %d mobs, want 3", iter, len(got))
		}
		want := []string{"Guard Gamma", "Guard Beta", "Guard Alpha"}
		for i := range want {
			if got[i].Name != want[i] {
				t.Fatalf("iter %d: buildRoomMobs[%d] = %q, want %q", iter, i, got[i].Name, want[i])
			}
		}
	}
}

// TestCOrderListOutput_SendStatRoom verifies R5h for sendStatRoom
// (src/act.wizard.c:452-465): mixed players and mobs entered in known order
// must appear in C's world[room].people newest-first order, 50/50 times.
func TestCOrderListOutput_SendStatRoom(t *testing.T) {
	const roomVNum = 9960
	for iter := range 50 {
		w, err := game.NewWorld(&parser.World{
			Rooms: []parser.Room{{VNum: roomVNum, Name: "Stat Room", Zone: 1}},
			Mobs: []parser.Mob{
				{VNum: 9961, Keywords: "alpha", ShortDesc: "Mob Alpha", DefaultPos: 8, Position: 8},
				{VNum: 9962, Keywords: "beta", ShortDesc: "Mob Beta", DefaultPos: 8, Position: 8},
			},
		})
		if err != nil {
			t.Fatalf("iter %d: NewWorld: %v", iter, err)
		}
		t.Cleanup(w.StopAITicker)

		if _, err := w.SpawnMob(9961, roomVNum); err != nil {
			t.Fatalf("iter %d: spawn 9961: %v", iter, err)
		}
		mgr := newTestManager(t, w, nil)
		s := makeTestSession(t, mgr, "Immortal", roomVNum, true)
		if err := w.AddPlayer(s.player); err != nil {
			t.Fatalf("iter %d: AddPlayer: %v", iter, err)
		}
		if _, err := w.SpawnMob(9962, roomVNum); err != nil {
			t.Fatalf("iter %d: spawn 9962: %v", iter, err)
		}

		s.sendStatRoom()
		var out strings.Builder
		for len(s.send) > 0 {
			out.WriteString(readMsgText(t, s))
		}
		text := out.String()
		idxBeta := strings.Index(text, "Mob Beta(MOB)")
		idxImm := strings.Index(text, "Immortal(PC)")
		idxAlpha := strings.Index(text, "Mob Alpha(MOB)")
		if idxBeta < 0 || idxImm < 0 || idxAlpha < 0 || !(idxBeta < idxImm && idxImm < idxAlpha) {
			t.Fatalf("iter %d: expected Mob Beta(MOB) < Immortal(PC) < Mob Alpha(MOB) in:\n%s", iter, text)
		}
	}
}
