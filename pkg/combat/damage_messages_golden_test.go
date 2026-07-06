package combat

import (
	"testing"
)

// TestDamageMessages_Golden verifies that damage values correctly map to the expected
// damage message tiers and thresholds defined in the Go implementation.
func TestDamageMessages_Golden(t *testing.T) {
	tests := []struct {
		damage         int
		expectedMinDam int
	}{
		{damage: 0, expectedMinDam: 0},
		{damage: 1, expectedMinDam: 1},
		{damage: 2, expectedMinDam: 1},
		{damage: 3, expectedMinDam: 3},
		{damage: 4, expectedMinDam: 3},
		{damage: 5, expectedMinDam: 5},
		{damage: 6, expectedMinDam: 5},
		{damage: 7, expectedMinDam: 7},
		{damage: 10, expectedMinDam: 7},
		{damage: 11, expectedMinDam: 11},
		{damage: 17, expectedMinDam: 11},
		{damage: 18, expectedMinDam: 18},
		{damage: 25, expectedMinDam: 18},
		{damage: 26, expectedMinDam: 26},
		{damage: 35, expectedMinDam: 26},
		{damage: 36, expectedMinDam: 36},
		{damage: 47, expectedMinDam: 36},
		{damage: 48, expectedMinDam: 48},
		{damage: 59, expectedMinDam: 48},
		{damage: 60, expectedMinDam: 60},
		{damage: 79, expectedMinDam: 60},
		{damage: 80, expectedMinDam: 80},
		{damage: 100, expectedMinDam: 80},
		{damage: 101, expectedMinDam: 101},
		{damage: 9999, expectedMinDam: 101},
		{damage: 10000, expectedMinDam: 10000},
		{damage: 15000, expectedMinDam: 10000},
	}

	for _, tt := range tests {
		var matchedTier *damMessageTier
		for i := len(damMessageTiers) - 1; i >= 0; i-- {
			if tt.damage >= damMessageTiers[i].MinDamage {
				matchedTier = &damMessageTiers[i]
				break
			}
		}

		if matchedTier == nil {
			t.Fatalf("No matching tier found for damage %d", tt.damage)
		}

		if matchedTier.MinDamage != tt.expectedMinDam {
			t.Errorf("Damage %d mapped to tier with MinDamage %d; want %d",
				tt.damage, matchedTier.MinDamage, tt.expectedMinDam)
		}
	}
}
