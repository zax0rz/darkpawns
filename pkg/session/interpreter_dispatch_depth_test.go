package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestCommandPositionRefusalOutputBranches(t *testing.T) {
	tests := []struct {
		name     string
		position int
		want     string
	}{
		{name: "dead", position: combat.PosDead, want: "Lie still; you are DEAD!!! :-(\r\n"},
		{name: "incap", position: combat.PosIncap, want: "You are in a pretty bad shape, unable to do anything!\r\n"},
		{name: "mortally-wounded", position: combat.PosMortally, want: "You are in a pretty bad shape, unable to do anything!\r\n"},
		{name: "stunned", position: combat.PosStunned, want: "All you can do right now is think about the stars!\r\n"},
		{name: "sleeping", position: combat.PosSleeping, want: "In your dreams, or what?\r\n"},
		{name: "resting", position: combat.PosResting, want: "Nah... You feel too relaxed to do that..\r\n"},
		{name: "sitting", position: combat.PosSitting, want: "Maybe you should get on your feet first?\r\n"},
		{name: "fighting", position: combat.PosFighting, want: "No way!  You're fighting for your life!\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := makeTestManager(t)
			s := makeCommandTestSession(t, m, "PositionGate", 1, 1001)
			s.player.SetPosition(tt.position)

			if err := ExecuteCommand(s, "wiggle", nil); err != nil {
				t.Fatal(err)
			}
			if got := readMsgText(t, s); got != tt.want {
				t.Fatalf("wiggle at position %d = %q, want %q", tt.position, got, tt.want)
			}
		})
	}
}
