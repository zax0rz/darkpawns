package spells

import "testing"

func TestCastPositionRefusalMessageExactBytes(t *testing.T) {
	tests := []struct {
		name     string
		position int
		want     string
	}{
		{name: "dead", position: int(PosDead), want: "You can't do much of anything like this!\r\n"},
		{name: "mortally wounded", position: int(PosMortallyWounded), want: "You can't do much of anything like this!\r\n"},
		{name: "incapacitated", position: int(PosIncap), want: "You can't do much of anything like this!\r\n"},
		{name: "stunned", position: int(PosStunned), want: "You can't do much of anything like this!\r\n"},
		{name: "sleeping", position: int(PosSleeping), want: "You dream about great magical powers.\r\n"},
		{name: "resting", position: int(PosResting), want: "You cannot concentrate while resting.\r\n"},
		{name: "sitting", position: int(PosSitting), want: "You can't do this sitting!\r\n"},
		{name: "fighting", position: int(PosFighting), want: "Impossible!  You can't concentrate enough!\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CastPositionRefusalMessage(tt.position); got != tt.want {
				t.Errorf("CastPositionRefusalMessage(%d) = %q, want %q", tt.position, got, tt.want)
			}
		})
	}
}
