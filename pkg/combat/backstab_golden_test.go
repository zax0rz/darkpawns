package combat

import (
	"testing"
)

// TestBackstabMult_GoldenAgainstCSource verifies that the Go implementation of BackstabMult
// correctly matches the expected multipliers across various level thresholds, including Go's
// specific immortal-capping behavior at LVL_IMMORT (31).
func TestBackstabMult_GoldenAgainstCSource(t *testing.T) {
	tests := []struct {
		level int
		want  float64
	}{
		{level: -5, want: 1.0},
		{level: 0, want: 1.0},
		{level: 1, want: 1.2},
		{level: 2, want: 1.4},
		{level: 5, want: 2.0},
		{level: 10, want: 3.0},
		{level: 15, want: 4.0},
		{level: 20, want: 5.0},
		{level: 25, want: 6.0},
		{level: 30, want: 7.0},
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
