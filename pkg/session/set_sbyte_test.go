package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestSetHitrollDamrollWrapToSByte — do_set applies RANGE(-20, 200) to the
// parsed int and then assigns it to points.hitroll/points.damroll, which C
// declares as sbyte (src/act.wizard.c:2836-2840; src/structs.h:916-917). The
// assignment truncates, so 128..200 wrap into -128..-56 while the
// acknowledgement still prints the clamped value. The port stored the wide
// value, so every downstream to-hit and damage roll diverged.
func TestSetHitrollDamrollWrapToSByte(t *testing.T) {
	// The clamp itself is C's RANGE on the int: the ack prints it unchanged.
	if got := clampSetValue("damroll", 200, nil, nil); got != 200 {
		t.Fatalf("clampSetValue(damroll, 200) = %d, want 200 (C's RANGE, printed by the ack)", got)
	}

	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Sbyte set", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 18305, Keywords: "cleaner", ShortDesc: "a Big Burly Cleaner", Level: 13, THAC0: 7}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)

	mob, err := w.SpawnMob(18305, 1001)
	if err != nil {
		t.Fatal(err)
	}
	target := setTarget{mob: mob}

	cases := []struct {
		field string
		in    int
		want  int
	}{
		{"damroll", 200, -56},
		{"damroll", 128, -128},
		{"damroll", 127, 127},
		{"damroll", -20, -20},
		{"hitroll", 200, -56},
	}
	for _, tc := range cases {
		if _, _, early := applySetField(nil, target, setField{name: tc.field, typ: setNumber}, "", tc.in, false); early {
			t.Fatalf("applySetField(%s, %d) returned early", tc.field, tc.in)
		}
		got := mob.GetDamroll()
		if tc.field == "hitroll" {
			got = mob.GetHitroll()
		}
		if got != tc.want {
			t.Errorf("after set %s %d: stored value = %d, want %d (C's sbyte storage)", tc.field, tc.in, got, tc.want)
		}
	}
}
