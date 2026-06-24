package combat

import "testing"

// Tier-2 fidelity golden test (deterministic — no RNG, no game-code change).
//
// dexAppGolden is the dexterity-apply table transcribed VERBATIM from the original C source
// (src/constants.c: `struct dex_app_type dex_app[]`), all three columns — reaction, missile-attack,
// defensive — the three fields the Go port keeps and that drive AC/combat math. Index is the
// dexterity-apply index 0–25. Any divergence here means AC calculations or initiative diverge
// from the original game.
var dexAppGolden = [26][3]int{
	{-7, -7, 6}, // 0
	{-6, -6, 5}, // 1
	{-4, -4, 5}, // 2
	{-3, -3, 4}, // 3
	{-2, -2, 3}, // 4
	{-1, -1, 2}, // 5
	{0, 0, 1},   // 6
	{0, 0, 0},   // 7
	{0, 0, 0},   // 8
	{0, 0, 0},   // 9
	{0, 0, 0},   // 10
	{0, 0, 0},   // 11
	{0, 0, 0},   // 12
	{0, 0, 0},   // 13
	{0, 0, 0},   // 14
	{0, 0, -1},  // 15
	{1, 1, -2},  // 16
	{2, 2, -3},  // 17
	{2, 2, -4},  // 18
	{3, 3, -4},  // 19
	{3, 3, -4},  // 20
	{4, 4, -5},  // 21
	{4, 4, -5},  // 22
	{4, 4, -5},  // 23
	{5, 5, -6},  // 24
	{5, 5, -6},  // 25
}

// TestDexApp_GoldenAgainstCSource asserts the Go dexApp table reproduces the C table exactly.
// A mismatch means dexterity-based AC or attack modifiers diverge from the original game.
func TestDexApp_GoldenAgainstCSource(t *testing.T) {
	if len(dexApp) != len(dexAppGolden) {
		t.Fatalf("dexApp length = %d, want %d (per src/constants.c dex_app[])", len(dexApp), len(dexAppGolden))
	}
	for i, want := range dexAppGolden {
		if dexApp[i].Reaction != want[0] || dexApp[i].MissAtt != want[1] || dexApp[i].Defensive != want[2] {
			t.Errorf("dexApp[%d] = {Reaction:%d MissAtt:%d Defensive:%d}, want {Reaction:%d MissAtt:%d Defensive:%d} (per src/constants.c)",
				i, dexApp[i].Reaction, dexApp[i].MissAtt, dexApp[i].Defensive,
				want[0], want[1], want[2])
		}
	}
}
