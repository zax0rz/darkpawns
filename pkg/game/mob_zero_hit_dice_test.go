package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestNewMobZeroHitDiceFollowsCDiceArithmetic pins C's read_mobile arithmetic
// for a prototype with no hit dice. C computes dice(hit, mana) + move
// unconditionally (src/db.c:1748-1751) and dice() returns 0 without drawing
// when either count is non-positive (src/utils.c:66-77), so such a mobile has
// no maximum hit points at all. The port used to invent a default of 100,
// which made every medit-built mobile read "is in excellent condition" instead
// of diag_char_to_char's bleeding line (src/act.informative.c:359-381).
func TestNewMobZeroHitDiceFollowsCDiceArithmetic(t *testing.T) {
	for _, tc := range []struct {
		name string
		hp   parser.DiceRoll
		want int
	}{
		{"no dice at all", parser.DiceRoll{}, 0},
		{"zero count", parser.DiceRoll{Sides: 6}, 0},
		{"zero sides", parser.DiceRoll{Num: 2}, 0},
		{"plus survives", parser.DiceRoll{Plus: 7}, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mob := NewMob(&parser.Mob{Keywords: "probe", ShortDesc: "a probe", HP: tc.hp}, 1)
			if got := mob.GetMaxHP(); got != tc.want {
				t.Fatalf("MaxHP = %d, want %d", got, tc.want)
			}
			if got := mob.GetHealth(); got != tc.want {
				t.Fatalf("Health = %d, want %d", got, tc.want)
			}
		})
	}
}
