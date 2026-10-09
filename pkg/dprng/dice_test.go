package dprng

import "testing"

// Moved from pkg/spells (TestDice_Basic, TestDice_MultipleDice,
// TestDice_ZeroOrNegative) when the one-line spells.Dice wrapper was deleted
// as dead code: these were the only assertions on Dice itself.
func TestDice_Basic(t *testing.T) {
	for i := 0; i < 50; i++ {
		if got := Dice(1, 6); got < 1 || got > 6 {
			t.Errorf("Dice(1,6) = %d, want 1-6", got)
		}
	}
}

func TestDice_MultipleDice(t *testing.T) {
	for i := 0; i < 50; i++ {
		if got := Dice(3, 6); got < 3 || got > 18 {
			t.Errorf("Dice(3,6) = %d, want 3-18", got)
		}
	}
}

func TestDice_ZeroOrNegative(t *testing.T) {
	tests := []struct {
		num, sides int
	}{
		{0, 6},
		{2, 0},
		{-1, 6},
		{2, -1},
		{0, 0},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := Dice(tt.num, tt.sides); got != 0 {
				t.Errorf("Dice(%d,%d) = %d, want 0", tt.num, tt.sides, got)
			}
		})
	}
}
