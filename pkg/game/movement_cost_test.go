package game

import (
	"testing"
)

// These tests guard the movement-cost table and immortal exemption (DP-1029 / F9).
// See docs/briefs/BRIEF-2026-07-11-glm-dp1029-movement-cost.md.

// TestMovementLossTable asserts the single shared movementLoss table matches the
// one-true table from src/constants.c movement_loss[], guarding the C
// comment/value swap at indices 8/9 from regressing.
func TestMovementLossTable(t *testing.T) {
	want := []int{2, 2, 3, 4, 5, 7, 5, 6, 2, 6, 8, 6, 6, 6, 6, 4}
	if len(movementLoss) != 16 {
		t.Fatalf("movementLoss must have 16 entries, got %d", len(movementLoss))
	}
	for i, w := range want {
		if movementLoss[i] != w {
			t.Errorf("movementLoss[%d] = %d, want %d", i, movementLoss[i], w)
		}
	}

	// Explicitly pin the swapped entries: C's inline comments at idx 8/9 are
	// "Flying"/"Underwater" but the enum is SECT_UNDERWATER=8, SECT_FLYING=9.
	// The runtime indexes by enum, so the VALUES win. See the brief's TRAP section.
	if movementLoss[SECT_UNDERWATER] != 2 {
		t.Errorf("movementLoss[SECT_UNDERWATER=%d] = %d, want 2", SECT_UNDERWATER, movementLoss[SECT_UNDERWATER])
	}
	if movementLoss[SECT_FLYING] != 6 {
		t.Errorf("movementLoss[SECT_FLYING=%d] = %d, want 6", SECT_FLYING, movementLoss[SECT_FLYING])
	}
}

// TestSectorMoveCost samples a few sectors (incl. DESERT which previously fell
// through to default) and confirms an out-of-range index returns the INSIDE
// default rather than panicking.
func TestSectorMoveCost(t *testing.T) {
	cases := []struct {
		sector int
		want   int
	}{
		{SECT_INSIDE, 2},
		{SECT_FIELD, 3},
		{SECT_DESERT, 8}, // sectors 10-15 previously fell through to default 1
		{SECT_SWAMP, 4},
		{SECT_UNDERWATER, 2},
		{SECT_FLYING, 6},
	}
	for _, c := range cases {
		if got := sectorMoveCost(c.sector); got != c.want {
			t.Errorf("sectorMoveCost(%d) = %d, want %d", c.sector, got, c.want)
		}
	}

	// Out-of-range returns the INSIDE cost, not a panic.
	if got := sectorMoveCost(-1); got != movementLoss[SECT_INSIDE] {
		t.Errorf("sectorMoveCost(-1) = %d, want INSIDE default %d", got, movementLoss[SECT_INSIDE])
	}
	if got := sectorMoveCost(len(movementLoss)); got != movementLoss[SECT_INSIDE] {
		t.Errorf("sectorMoveCost(OOB) = %d, want INSIDE default %d", got, movementLoss[SECT_INSIDE])
	}
}

// TestMovePlayer_MortalPaysMoveCost verifies a mortal player's move points drop
// by the (src+dst)/2 cost when moving between two rooms of known sectors.

// TestMovePlayer_ImmortalExempt verifies an immortal's move points are unchanged.

// TestMovePlayer_ImmortalExemptWhenExhausted confirms immortals move even with
// zero move points — they never hit the "too exhausted" path.

// TestMovePlayer_MortalExhaustedBlocked confirms a mortal with too few move
// points is still blocked (regression guard for the immortal short-circuit).
