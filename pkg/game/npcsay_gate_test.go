package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// npcSayGateWorld builds a room with one mob whose prototype INT/WIS are the
// arguments, plus a standing observer that receives the room line.
func npcSayGateWorld(t *testing.T, intel, wis int) (*World, *MobInstance, *strings.Builder) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Gate Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum: 9001, Keywords: "healer", ShortDesc: "the healer",
			Level: 1, Int: intel, Wis: wis,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	observer := NewPlayer(1, "Observer", 1001)
	observer.SetPosition(8) // POS_STANDING: SENDOK
	if err := w.AddPlayer(observer); err != nil {
		t.Fatal(err)
	}
	mob, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	out := &strings.Builder{}
	w.MessageSink = func(name string, msg []byte) {
		if name == "Observer" {
			out.Write(msg)
		}
	}
	return w, mob, out
}

// TestNPCSayStupidGate pins C's do_say refusal (src/act.comm.c:766-771):
// GET_WIS==0 or GET_INT==0 makes the speaker too stupid to use language, the
// refusal is TO_CHAR only, and a descriptor-less mobile therefore emits
// nothing. Revert proof: without the gate every row below spoke.
func TestNPCSayStupidGate(t *testing.T) {
	cases := []struct {
		name       string
		intel, wis int
		wantSaid   bool
	}{
		{"capable", 11, 11, true},
		{"zero int", 0, 11, false},
		{"zero wis", 11, 0, false},
		{"zero both", 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, mob, out := npcSayGateWorld(t, tc.intel, tc.wis)

			w.npcSay(mob, "My work here is done.")
			got := out.String()
			if tc.wantSaid && !strings.Contains(got, "states, 'My work here is done.'") {
				t.Fatalf("int=%d wis=%d did not speak: %q", tc.intel, tc.wis, got)
			}
			if !tc.wantSaid && got != "" {
				t.Fatalf("int=%d wis=%d spoke while too stupid: %q", tc.intel, tc.wis, got)
			}
		})
	}
}
