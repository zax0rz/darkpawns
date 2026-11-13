package common

import (
	"math"
	"strings"
)

// AtoiC parses s exactly like the C build's atoi, i.e. glibc
// atoi(s) == (int)strtol(s, NULL, 10) (src/act.wizard.c:1294 and siblings
// parse player numbers this way):
//
//   - leading isspace bytes (space, \t, \n, \v, \f, \r) are skipped;
//   - one optional '+' or '-' sign follows;
//   - the longest leading run of decimal digits is taken and scanning stops
//     at the first non-digit, so "5abc" is 5 and "12.9" is 12;
//   - no digits at all (empty, sign only, non-digit first) yield 0.
//
// Overflow does not fail: strtol clamps to LONG_MAX/LONG_MIN and the cast to
// C's 32-bit int keeps the low 32 bits. Proven against the host glibc the
// reference oracle links (DP-1523 probe, 2026-11-13): "3000000000" wraps to
// -1294967296 (pinned by give.gold-overflow-wrap, lines.atoi-overflow and
// wimpy.negative-dead), "99999999999999999999" saturates to LONG_MAX and
// truncates to -1 (C's "A NEGATIVE number??" arm), "-99999999999999999999"
// saturates to LONG_MIN and truncates to 0.
func AtoiC(s string) int {
	s = strings.TrimLeft(s, " \t\n\r\v\f")
	if s == "" {
		return 0
	}

	index := 0
	negative := false
	if s[0] == '+' || s[0] == '-' {
		negative = s[0] == '-'
		index++
	}
	start := index
	var value int64
	overflow := false
	for index < len(s) && s[index] >= '0' && s[index] <= '9' {
		digit := int64(s[index] - '0')
		if value > (math.MaxInt64-digit)/10 {
			overflow = true
		} else {
			value = value*10 + digit
		}
		index++
	}
	if index == start {
		return 0
	}
	if overflow {
		// strtol returns LONG_MAX/LONG_MIN; truncating to 32 bits gives
		// 0x7FFFFFFFFFFFFFFF -> -1 and 0x8000000000000000 -> 0.
		if negative {
			return 0
		}
		return -1
	}
	if negative {
		value = -value
	}
	return int(int32(value))
}
