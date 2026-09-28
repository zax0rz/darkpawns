package game

// stop_fighting_position_test.go — DP-1321's R5h unit proof for the
// post-combat position reset. C's stop_fighting removes the character from
// the combat list, clears FIGHTING, sets POS_STANDING, then calls
// update_pos, which re-derives the wounded band from hit points
// (src/fight.c:230-252 and 186-201). Without the reset a player stayed
// POS_FIGHTING after combat ended and every move answered "No way! You're
// fighting for your life!" — the fix is on main (Player.StopFighting); this
// test is what fails if it is reverted.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// TestStopFightingPositionBands covers update_pos's thresholds at the exact
// HP values C branches on (fight.c:189-200): above 0 stays standing (set by
// stop_fighting before update_pos returns early), 0 is stunned, -3 incap,
// -6 mortally wounded, -11 dead.
func TestStopFightingPositionBands(t *testing.T) {
	cases := []struct {
		hp   int
		want int
		name string
	}{
		{hp: 100, want: combat.PosStanding, name: "healthy-standing"},
		{hp: 1, want: combat.PosStanding, name: "one-hp-standing"},
		{hp: 0, want: combat.PosStunned, name: "zero-stunned"},
		{hp: -3, want: combat.PosIncap, name: "minus-three-incap"},
		{hp: -6, want: combat.PosMortally, name: "minus-six-mortally"},
		{hp: -11, want: combat.PosDead, name: "minus-eleven-dead"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPlayer(1, "Bandtest", 1001)
			p.Fighting = "wharf rat"
			p.Position = combat.PosFighting
			p.SetHealth(tc.hp)
			p.MaxHealth = 100

			p.StopFighting()

			if p.Fighting != "" {
				t.Fatalf("Fighting = %q, want cleared", p.Fighting)
			}
			if p.GetPosition() != tc.want {
				t.Fatalf("position after StopFighting at HP %d = %d, want %d", tc.hp, p.GetPosition(), tc.want)
			}
		})
	}
}
