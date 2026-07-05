package combat

import (
	"testing"
)

// TestACReduction_GoldenAgainstCSource verifies that getMinusDam correctly applies the tiered
// AC-based damage reduction percentages to match the C implementation logic.
func TestACReduction_GoldenAgainstCSource(t *testing.T) {
	tests := []struct {
		ac   int
		want int
	}{
		{ac: 100, want: 100},
		{ac: 91, want: 100},
		{ac: 90, want: 98},
		{ac: 81, want: 98},
		{ac: 80, want: 96},
		{ac: 71, want: 96},
		{ac: 70, want: 94},
		{ac: 61, want: 94},
		{ac: 60, want: 92},
		{ac: 51, want: 92},
		{ac: 50, want: 90},
		{ac: 41, want: 90},
		{ac: 40, want: 88},
		{ac: 31, want: 88},
		{ac: 30, want: 86},
		{ac: 21, want: 86},
		{ac: 20, want: 84},
		{ac: 11, want: 84},
		{ac: 10, want: 80},
		{ac: 1, want: 80},
		{ac: 0, want: 78},
		{ac: -9, want: 78},
		{ac: -10, want: 76},
		{ac: -19, want: 76},
		{ac: -20, want: 74},
		{ac: -29, want: 74},
		{ac: -30, want: 72},
		{ac: -39, want: 72},
		{ac: -40, want: 70},
		{ac: -49, want: 70},
		{ac: -50, want: 68},
		{ac: -59, want: 68},
		{ac: -60, want: 66},
		{ac: -69, want: 66},
		{ac: -70, want: 64},
		{ac: -79, want: 64},
		{ac: -80, want: 62},
		{ac: -89, want: 62},
		{ac: -90, want: 60},
		{ac: -94, want: 60},
		{ac: -95, want: 58},
		{ac: -109, want: 58},
		{ac: -110, want: 56},
		{ac: -129, want: 56},
		{ac: -130, want: 54},
		{ac: -149, want: 54},
		{ac: -150, want: 52},
		{ac: -169, want: 52},
		{ac: -170, want: 50},
		{ac: -189, want: 50},
		{ac: -190, want: 48},
		{ac: -209, want: 48},
		{ac: -210, want: 46},
		{ac: -229, want: 46},
		{ac: -230, want: 44},
		{ac: -249, want: 44},
		{ac: -250, want: 43},
		{ac: -269, want: 43},
		{ac: -270, want: 40},
		{ac: -289, want: 40},
		{ac: -290, want: 38},
		{ac: -309, want: 38},
		{ac: -310, want: 36},
		{ac: -350, want: 36},
	}

	const baseDamage = 100

	for _, tt := range tests {
		got := getMinusDam(baseDamage, tt.ac)
		if got != tt.want {
			t.Errorf("getMinusDam(%d, %d) = %d; want %d", baseDamage, tt.ac, got, tt.want)
		}
	}
}
