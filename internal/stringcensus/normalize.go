// Package stringcensus implements the two-directional player-facing string
// census that cmd/dp-string-census exposes.
//
// The census exists because `make oracle-regression` can only compare strings
// some scenario happens to make both servers print. Two surfaces proved that
// blind spot on 2026-09-23: Go printed `[ GHOST SHIP ] ...` and accepted
// `autoexit`, neither of which has any C source (violations of R4 the harness
// never triggered). A static comparison of every player-facing string in both
// trees flags that class in seconds.
//
// The census never changes game code. It reads the Go tree with go/ast, lexes
// the read-only C oracle under src/, normalizes both sides the same way, and
// compares normalized fixed-text segments in both directions.
package stringcensus

import (
	"strings"
	"unicode/utf8"
)

// MinSegmentLen is the shortest normalized segment the census compares. Shorter
// fragments ("Ok.", "Yes") collide across unrelated strings and would drown the
// report in noise; the brief fixes the floor at 8 characters.
const MinSegmentLen = 8

// gap is the in-band separator used internally to mark a position where a
// normalized segment ends: a line break, a colour escape, a format verb, or an
// act code. It cannot appear in normalized text, so splitting on it is exact.
const gap = byte(0x00)

// actCodes is the union of the $-codes C's perform_act (src/comm.c:2408) and
// Go's performAct (pkg/game/act.go:356) substitute. Go implements more codes
// than C (t/T/r/R/q/Q are Go additions), so the union is the safe splitter for
// both trees.
const actCodes = "nNeEmMsSpPoOaAtTrRqQFT$"

// Normalize turns one unescaped string literal into the fixed-text segments the
// census compares.
//
// Rules, applied to a literal that has already been unescaped:
//
//  1. ANSI/colour escapes become gaps. C writes colour through CC*(ch, lvl)
//     macros, which are runtime expressions rather than literal concatenation,
//     so they are never part of a literal to begin with; Go embeds raw "\x1b["
//     sequences in literals. Both are transport presentation, not game text.
//  2. CR, LF and CRLF/LFCR become gaps: a line break ends a segment.
//  3. printf conversions (%s, %-10s, %ld, %v) and act codes ($n, $N, $o, ...)
//     become gaps: they are variable text, not fixed text.
//  4. "%%" and "$$" are literal '%' and '$' (both parsers do the same).
//  5. Whitespace inside a segment collapses to single spaces and is trimmed.
//  6. A segment shorter than MinSegmentLen runes is dropped.
//
// The result is ordered and deduplicated.
func Normalize(raw string) []string {
	return splitSegments(blankGaps(raw))
}

// blankGaps replaces every position that ends a segment (colour escape, line
// break) with the gap marker, so splitSegments has a single delimiter to work
// with.
func blankGaps(raw string) string {
	if !strings.ContainsAny(raw, "\x1b\r\n") {
		return raw
	}
	var b strings.Builder
	b.Grow(len(raw))
	for i := 0; i < len(raw); {
		switch c := raw[i]; c {
		case 0x1b:
			if n := ansiLen(raw[i:]); n > 0 {
				b.WriteByte(gap)
				i += n
				continue
			}
			b.WriteByte(gap)
			i++
		case '\r', '\n':
			b.WriteByte(gap)
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// ansiLen returns the byte length of the ANSI escape sequence at the start of
// s (which begins with ESC), or 0 when s holds a bare ESC with nothing valid
// after it. CSI sequences are matched with the same grammar the oracle harness
// uses (internal/oraclediff/normalize.go), so colour bytes are treated
// identically by the census and the differential run.
func ansiLen(s string) int {
	if len(s) < 2 || s[0] != 0x1b {
		return 0
	}
	if s[1] != '[' {
		// Two-byte escape (ESC + final byte).
		return 2
	}
	for i := 2; i < len(s); i++ {
		if s[i] >= '@' && s[i] <= '~' {
			return i + 1
		}
	}
	return 0
}

// splitSegments walks the gap-blanked literal and emits one segment per run of
// fixed text between gaps, format conversions and act codes.
func splitSegments(s string) []string {
	var (
		out  []string
		seen = make(map[string]struct{})
		cur  strings.Builder
	)
	flush := func() {
		seg := collapseWhitespace(cur.String())
		cur.Reset()
		if utf8.RuneCountInString(seg) < MinSegmentLen {
			return
		}
		if _, dup := seen[seg]; dup {
			return
		}
		seen[seg] = struct{}{}
		out = append(out, seg)
	}
	for i := 0; i < len(s); {
		switch c := s[i]; {
		case c == gap:
			flush()
			i++
		case c == '%':
			n, literal := conversionLen(s[i:])
			if n == 0 {
				cur.WriteByte(c)
				i++
				continue
			}
			if literal {
				cur.WriteByte('%')
			} else {
				flush()
			}
			i += n
		case c == '$' && i+1 < len(s) && strings.IndexByte(actCodes, s[i+1]) >= 0:
			if s[i+1] == '$' {
				cur.WriteByte('$')
			} else {
				flush()
			}
			i += 2
		default:
			cur.WriteByte(c)
			i++
		}
	}
	flush()
	return out
}

// conversionLen reports the byte length of the printf conversion at the start
// of s (which begins with '%'), and whether it is a literal percent ("%%").
// It returns 0 when s does not begin a conversion, in which case the '%' is
// ordinary text ("50% off").
//
// C and Go differ in which conversions exist (%ld and %v do not overlap), but
// the grammar is the same, so both trees share this parser: flags, width,
// precision, length modifiers, conversion byte.
//
// Two deliberate narrowings, both because player prose matters more here than
// the full printf grammar:
//
//   - the space flag is not recognised, so "50% of your load" stays one
//     segment instead of splitting at a legal-but-unused "% o";
//   - a length modifier that is also a Go conversion (%q, %t) counts as the
//     conversion when nothing follows it.
func conversionLen(s string) (int, bool) {
	if len(s) == 0 || s[0] != '%' {
		return 0, false
	}
	i := 1
	if i < len(s) && s[i] == '%' {
		return 2, true
	}
	// Explicit argument index, as in Go's "%[1]s".
	if i < len(s) && s[i] == '[' {
		j := i + 1
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		if j < len(s) && s[j] == ']' {
			i = j + 1
		}
	}
	for i < len(s) && strings.IndexByte("-+#0'", s[i]) >= 0 {
		i++
	}
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i < len(s) && s[i] == '.' {
		j := i + 1
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		i = j
	}
	mods := i
	for i < len(s) && strings.IndexByte("hlLqjzt", s[i]) >= 0 {
		i++
	}
	if i < len(s) && isLetter(s[i]) {
		return i + 1, false
	}
	if mods < len(s) && strings.IndexByte("qt", s[mods]) >= 0 {
		return mods + 1, false
	}
	return 0, false
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// collapseWhitespace trims a segment and collapses internal whitespace runs to
// single spaces, so a literal wrapped across source lines compares equal to the
// same sentence on one line in the other tree.
func collapseWhitespace(s string) string {
	if !strings.ContainsAny(s, " \t\v\f") {
		return strings.TrimSpace(s)
	}
	return strings.Join(strings.Fields(s), " ")
}
