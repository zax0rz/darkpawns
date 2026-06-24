package combat

import "testing"

// Tier-2 fidelity golden test (deterministic — no RNG, no game-code change).
//
// strAppGolden is the strength-apply table transcribed VERBATIM from the original C source
// (src/constants.c: `const struct str_app_type str_app[]`), columns {tohit, todam} only — the two
// fields the Go port keeps and the two that drive combat math. Index is the strength-apply index
// 0–30 (25 = STR 25, 26–30 = the 18/01-50 … 18/100 percentile band). getTHAC0 minus strApp[i].ToHit
// is the to-hit calc and strApp[i].ToDam feeds damage, so any divergence here is "numbers off".
var strAppGolden = [31][2]int{
	{-5, -4}, {-5, -4}, {-3, -2}, {-3, -1}, {-2, -1}, {-2, -1}, {-1, 0}, {-1, 0}, // 0–7
	{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, // 8–15
	{0, 1}, {1, 1}, {1, 2}, {3, 7}, {3, 8}, {4, 9}, {4, 10}, {5, 11}, // 16–23
	{6, 12}, {7, 14}, // 24–25
	{1, 3}, {2, 3}, {2, 4}, {2, 5}, {3, 6}, // 26–30 = 18/01-50, 18/51-75, 18/76-90, 18/91-99, 18/100
}

// TestStrApp_GoldenAgainstCSource asserts the Go strApp ToHit/ToDam table reproduces the C table
// exactly. A mismatch means strength-based to-hit/damage diverges from the original game.
func TestStrApp_GoldenAgainstCSource(t *testing.T) {
	if len(strApp) != len(strAppGolden) {
		t.Fatalf("strApp length = %d, want %d (per src/constants.c str_app[])", len(strApp), len(strAppGolden))
	}
	for i, want := range strAppGolden {
		if strApp[i].ToHit != want[0] || strApp[i].ToDam != want[1] {
			t.Errorf("strApp[%d] = {ToHit:%d ToDam:%d}, want {ToHit:%d ToDam:%d} (per src/constants.c)",
				i, strApp[i].ToHit, strApp[i].ToDam, want[0], want[1])
		}
	}
}
