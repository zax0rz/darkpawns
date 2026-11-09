package game

import "testing"

// TestCFNameTable pins C's fname() (src/handler.c:68-79): the leading run of
// alphabetic bytes, stopping at the first non-letter. Revert proof: the old Go
// fname() stopped at the first space, so "black-banded haute" returned
// "black-banded" and "D'haran breast plate armor" returned "D'haran"; every row
// below with a hyphen, apostrophe or inner non-letter fails against it.
//
// holder[30]: C writes the terminator at holder[30], one past the array, so a
// first letter-run of 30+ bytes overflows. A scan of lib/world found no object
// or door keyword whose first run reaches 30 letters, so the overflow is not
// emulated; the 30-letter row documents that the full run is returned.
func TestCFNameTable(t *testing.T) {
	thirty := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 30 letters
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain word", "sword", "sword"},
		{"trailing keyword list", "sword short blade", "sword"},
		{"hyphenated first keyword", "black-banded haute", "black"},
		{"apostrophe first keyword", "D'haran breast plate armor", "D"},
		{"hyphen unseen object", "chai-dao chai dao polearm", "chai"},
		{"apostrophe object", "titan's mit", "titan"},
		{"leading hyphen", "-black banded", ""},
		{"leading digit keyword", "40 oz 0 0 0 ball 0 0 0", ""},
		{"angle placeholder", "<NONE>", ""},
		{"only spaces", "   ", ""},
		{"empty", "", ""},
		{"30 letters (no overflow emulated)", thirty, thirty},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CFName(tc.in); got != tc.want {
				t.Fatalf("CFName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
