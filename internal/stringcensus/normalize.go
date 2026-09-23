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
// The result is ordered and deduplicated; use NormalizeLines when the order of
// printed lines matters.
func Normalize(raw string) []string {
	var (
		out  []string
		seen = make(map[string]struct{})
	)
	for _, fragment := range NormalizeLines(raw) {
		for _, seg := range fragment {
			if _, dup := seen[seg]; dup {
				continue
			}
			seen[seg] = struct{}{}
			out = append(out, seg)
		}
	}
	return out
}

// Boundaries inside a marked literal. gapSegment ends a segment (a colour
// escape, a format conversion, an act code, a line break); gapLine ends a
// printed line. Neither byte can appear in real player text, and both are
// stripped from the input before marking, so a literal that somehow contains
// them cannot forge a boundary.
const (
	gapSegment = byte(0x00)
	gapLine    = byte(0x01)
)

// NormalizeLines splits one literal into the lines the server prints, each line
// an ordered list of the fixed-text segments that survive normalization.
// Fragment boundaries come only from line breaks; colour escapes, format
// conversions and act codes end a segment inside a line. Duplicates are kept,
// because a caller matching real output needs the order and the count.
//
// By the time a literal arrives here C's adjacent-literal concatenation and
// Go's "+" have already joined their pieces, so one fragment is exactly one
// printed line of fixed text.
func NormalizeLines(raw string) [][]string {
	var (
		fragments [][]string
		fragment  []string
		cur       strings.Builder
	)
	flushSegment := func() {
		if seg := collapseWhitespace(cur.String()); utf8.RuneCountInString(seg) >= MinSegmentLen {
			fragment = append(fragment, seg)
		}
		cur.Reset()
	}
	flushLine := func() {
		flushSegment()
		if len(fragment) > 0 {
			fragments = append(fragments, fragment)
			fragment = nil
		}
	}
	s := markBoundaries(raw)
	for i := 0; i < len(s); {
		switch c := s[i]; {
		case c == gapLine:
			flushLine()
			i++
		case c == gapSegment:
			flushSegment()
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
				flushSegment()
			}
			i += n
		case c == '$' && i+1 < len(s) && strings.IndexByte(actCodes, s[i+1]) >= 0:
			if s[i+1] == '$' {
				cur.WriteByte('$')
			} else {
				flushSegment()
			}
			i += 2
		default:
			cur.WriteByte(c)
			i++
		}
	}
	flushLine()
	return fragments
}

// markBoundaries replaces every position that ends a segment or a line with a
// marker, so the splitter has one delimiter per rule to work with.
func markBoundaries(raw string) string {
	if !strings.ContainsAny(raw, "\x1b\r\n\x00\x01") {
		return raw
	}
	var b strings.Builder
	b.Grow(len(raw))
	for i := 0; i < len(raw); {
		switch c := raw[i]; c {
		case gapSegment, gapLine:
			i++ // reserved markers never reach the splitter
		case 0x1b:
			if n := ansiLen(raw[i:]); n > 0 {
				b.WriteByte(gapSegment)
				i += n
				continue
			}
			b.WriteByte(gapSegment)
			i++
		case '\r', '\n':
			b.WriteByte(gapLine)
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
