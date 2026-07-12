package combat

import (
	"testing"
)

// TestDamageMessages_Golden verifies that damage values correctly map to the
// expected damage message tiers. Boundaries match src/fight.c:981-992 (DP-1043):
//
//	dam==0 → 0, <=2 → 1, <=4 → 2, <=6 → 3, <=10 → 4, <=14 → 5,
//	<=19 → 6, <=23 → 7, <=33 → 8, <=43 → 9, <=53 → 10, else → 11
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
		{damage: 14, expectedMinDam: 11},
		{damage: 15, expectedMinDam: 15},
		{damage: 19, expectedMinDam: 15},
		{damage: 20, expectedMinDam: 20},
		{damage: 23, expectedMinDam: 20},
		{damage: 24, expectedMinDam: 24},
		{damage: 33, expectedMinDam: 24},
		{damage: 34, expectedMinDam: 34},
		{damage: 43, expectedMinDam: 34},
		{damage: 44, expectedMinDam: 44},
		{damage: 53, expectedMinDam: 44},
		{damage: 54, expectedMinDam: 54},
		{damage: 100, expectedMinDam: 54},
		{damage: 9999, expectedMinDam: 54},
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
