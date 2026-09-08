package game

import "testing"

// Pin extra-flag bits to the C ITEM_* values in src/structs.h:468-496.
// World data and save files store C-numbered bits; drift here silently
// breaks alignment gates and rent filtering (see Linear DP-1258).
func TestExtraFlagBitsMatchCStructs(t *testing.T) {
	maskCases := []struct {
		name string
		got  int
		want int
	}{
		{"FlagNoRent", FlagNoRent, 1 << 2},            // ITEM_NORENT = 2
		{"FlagAntiGood", FlagAntiGood, 1 << 9},        // ITEM_ANTI_GOOD = 9
		{"FlagAntiEvil", FlagAntiEvil, 1 << 10},       // ITEM_ANTI_EVIL = 10
		{"FlagAntiNeutral", FlagAntiNeutral, 1 << 11}, // ITEM_ANTI_NEUTRAL = 11
	}
	for _, tc := range maskCases {
		if tc.got != tc.want {
			t.Errorf("%s mask mismatch: got bit %d, want bit %d (C ITEM_* position)", tc.name, bitPos(tc.got), bitPos(tc.want))
		}
	}

	// extraFlag* constants are raw bit positions (see item_helpers.go:140-141).
	posCases := []struct {
		name string
		got  int
		want int
	}{
		{"extraFlagNoDonate", extraFlagNoDonate, 3},    // ITEM_NODONATE = 3
		{"extraFlagInvisible", extraFlagInvisible, 5},  // ITEM_INVISIBLE = 5
		{"extraFlagNoDrop", extraFlagNoDrop, 7},        // ITEM_NODROP = 7
		{"extraFlagTakeName", extraFlagTakeName, 17},   // ITEM_TAKE_NAME = 17
		{"extraFlagTwoHanded", extraFlagTwoHanded, 28}, // ITEM_TWO_HANDED = 28
	}
	for _, tc := range posCases {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d (C ITEM_* bit position)", tc.name, tc.got, tc.want)
		}
	}
}

func bitPos(v int) int {
	n := 0
	for v&1 == 0 {
		v >>= 1
		n++
	}
	return n
}
