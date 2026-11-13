package common

import "testing"

// TestAtoiCMatchesC pins AtoiC against literal values proven from the C build
// (glibc atoi == (int)strtol, probed on 2026-11-13 with the same libc the
// reference oracle links) and from oracle-green depth rows — never values
// computed by the helper itself (R5h):
//
//   - "3000000000" -> -1294967296: give.gold-overflow-wrap
//     (docs/fidelity/depth/give.tsv), lines.atoi-overflow (lines.tsv) and
//     wimpy.negative-dead (other.tsv);
//   - "99999999999999999999" -> -1: C answers "A NEGATIVE number??"
//     (src/act.wizard.c:1293-1295, 1393-1395); the 20-digit token saturates
//     strtol at LONG_MAX and the int cast keeps the low 32 bits;
//   - "-99999999999999999999" -> 0: strtol saturates at LONG_MIN, whose low
//     32 bits are 0.
func TestAtoiCMatchesC(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"letters", "abc", 0},
		{"sign only", "-", 0},
		{"plus only", "+", 0},
		{"zero", "0", 0},
		{"plain", "42", 42},
		{"leading whitespace", " \t\n\v\f\r 7", 7},
		{"trailing letters", "5abc", 5},
		{"signed trailing junk", "  -12xy", -12},
		{"plus sign", "  +42zz", 42},
		{"decimal point stops scan", "12.9", 12},
		{"hex prefix stops scan", "0x10", 0},
		{"int max", "2147483647", 2147483647},
		{"int max plus one", "2147483648", -2147483648},
		{"uint32 range wraps to zero", "4294967296", 0},
		{"give lines wimpy wrap", "3000000000", -1294967296},
		{"negative give wrap", "-3000000000", 1294967296},
		{"long max truncates", "9223372036854775807", -1},
		{"long max plus one saturates", "9223372036854775808", -1},
		{"load vstat saturation", "99999999999999999999", -1},
		{"long min truncates", "-9223372036854775808", 0},
		{"long min minus one saturates", "-9223372036854775809", 0},
		{"negative saturation", "-99999999999999999999", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := AtoiC(tc.input); got != tc.want {
				t.Errorf("AtoiC(%q) = %d, want C's %d", tc.input, got, tc.want)
			}
		})
	}
}
