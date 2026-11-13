package session

import "testing"

// TestParseSignedIntPrefixWrapsLikeCAtoi pins the DP-1523 movement batch:
// C's atoi never fails (src/act.wizard.c:1742 for dc, :1513 for advance,
// new_cmds2.c:829 for dig), so a 20-digit token wraps negative through
// (int)strtol and stays present, while digit-leading junk parses. strconv.Atoi
// instead reported !ok, sending dc to its usage arm. Fails on revert (R5h).
func TestParseSignedIntPrefixWrapsLikeCAtoi(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int
		ok    bool
	}{
		{"overflow wraps negative", "99999999999999999999", -1, true},
		{"unsigned 32-bit wrap", "4294967301", 5, true},
		{"digit-leading junk", "1abc", 1, true},
		{"plain", "42", 42, true},
		{"negative literal", "-7", -7, true},
		{"no digits", "abc", 0, false},
		{"sign only", "-", 0, false},
		{"empty", "", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseSignedIntPrefix(tc.input)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("parseSignedIntPrefix(%q) = (%d, %v), want (%d, %v)",
					tc.input, got, ok, tc.want, tc.ok)
			}
		})
	}
}

// TestParseLeadingIntWrapsOntoValidVnum pins goto/at's parse
// (src/act.wizard.c:197-203): 4294975458 is 2^32 + 8162, and C's atoi lands
// it back on room 8162 instead of refusing the token.
func TestParseLeadingIntWrapsOntoValidVnum(t *testing.T) {
	got, ok := parseLeadingInt("4294975458")
	if !ok || got != 8162 {
		t.Fatalf("parseLeadingInt(4294975458) = (%d, %v), want (8162, true)", got, ok)
	}
	if got, ok := parseLeadingInt("12abc"); !ok || got != 12 {
		t.Fatalf("parseLeadingInt(12abc) = (%d, %v), want (12, true)", got, ok)
	}
}
