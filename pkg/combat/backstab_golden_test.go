package combat

import (
	"testing"
)

// TestBackstabMult_GoldenAgainstCSource verifies that the Go implementation of BackstabMult
// correctly matches the expected multipliers across various level thresholds, including Go's
// specific immortal-capping behavior at LVL_IMMORT (31).
//
// C's backstab_mult() returns int, so (level*.2)+1 is truncated on return
// (DP-1033): levels that aren't multiples of 5 round DOWN to the integer
// (e.g. level 14 → 3.8 → 3, level 19 → 4.8 → 4).
func TestBackstabMult_GoldenAgainstCSource(t *testing.T) {
	tests := []struct {
		level int
		want  float64
	}{
		{level: -5, want: 1.0},
		{level: 0, want: 1.0},
		{level: 1, want: 1.0},   // 1.2 → trunc 1
		{level: 2, want: 1.0},   // 1.4 → trunc 1
		{level: 4, want: 1.0},   // 1.8 → trunc 1
		{level: 5, want: 2.0},   // 2.0
		{level: 9, want: 2.0},   // 2.8 → trunc 2
		{level: 10, want: 3.0},  // 3.0
		{level: 14, want: 3.0},  // 3.8 → trunc 3 (DP-1033)
		{level: 15, want: 4.0},  // 4.0
		{level: 19, want: 4.0},  // 4.8 → trunc 4 (DP-1033)
		{level: 20, want: 5.0},  // 5.0
		{level: 25, want: 6.0},  // 6.0
		{level: 30, want: 7.0},  // 7.0
		{level: 31, want: 20.0}, // LVL_IMMORT (31) cap in Go
		{level: 50, want: 20.0},
		{level: 99, want: 20.0},
		{level: 100, want: 20.0},
		{level: 101, want: 20.0},
	}

	for _, tt := range tests {
		got := BackstabMult(tt.level)
		if got != tt.want {
			t.Errorf("BackstabMult(%d) = %f; want %f", tt.level, got, tt.want)
		}
	}
}
