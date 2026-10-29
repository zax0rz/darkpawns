package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestHitrollDamrollStorageIsSByte — C stores points.hitroll and
// points.damroll as sbyte (src/structs.h:916-917), so a value outside
// -128..127 wraps: set damroll 200 stores -56. Both the `set` command's clamp
// and every reader (the file-derived 20 - THAC0 for mobs, the affect total for
// players) must honor that width.
func TestHitrollDamrollStorageIsSByte(t *testing.T) {
	cases := []struct{ in, want int }{
		{-20, -20},
		{127, 127},
		{128, -128},
		{200, -56},
	}
	for _, tc := range cases {
		if got := SByteValue(tc.in); got != tc.want {
			t.Errorf("SByteValue(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}

	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Sbyte proof", Zone: 1}},
		// 20 - (-200) = 220, which C's sbyte points.hitroll stores as -36.
		Mobs: []parser.Mob{{VNum: 4209, Keywords: "dragon", ShortDesc: "the dragon", Level: 30, THAC0: -200}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)

	mob, err := w.SpawnMob(4209, 1001)
	if err != nil {
		t.Fatal(err)
	}
	if got := mob.GetHitroll(); got != -36 {
		t.Fatalf("mob GetHitroll() = %d; want -36 (20 - THAC0 220 wraps in the sbyte field)", got)
	}

	mob.SetDamroll(200)
	if got := mob.GetDamroll(); got != -56 {
		t.Fatalf("mob GetDamroll() = %d; want -56 (sbyte storage)", got)
	}
}
