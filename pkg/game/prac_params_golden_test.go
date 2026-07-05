package game

import (
	"testing"
)

// pracParamsGolden is the prac_params matrix transcribed verbatim from src/class.c:261-267
var pracParamsGolden = [4][12]int{
	// learned_level:
	{95, 95, 85, 80, 95, 95, 85, 80, 85, 95, 80, 95},
	// max_per_prac:
	{100, 100, 25, 25, 100, 100, 25, 25, 25, 100, 25, 100},
	// min_per_prac:
	{25, 25, 0, 0, 25, 25, 0, 0, 0, 25, 0, 25},
	// prac_type:
	{PracTypeSpell, PracTypeSpell, PracTypeSkill, PracTypeSkill, PracTypeSpell, PracTypeBoth, PracTypeSkill, PracTypeBoth, PracTypeBoth, PracTypeBoth, PracTypeSkill, PracTypeBoth},
}

// TestPracParams_GoldenAgainstCSource verifies that the Go PracParams table exactly matches the C source matrix.
func TestPracParams_GoldenAgainstCSource(t *testing.T) {
	for row := 0; row < 4; row++ {
		for col := 0; col < 12; col++ {
			got := PracParams[row][col]
			want := pracParamsGolden[row][col]
			if got != want {
				t.Errorf("PracParams[%d][%d] = %d; want %d", row, col, got, want)
			}
		}
	}
}
