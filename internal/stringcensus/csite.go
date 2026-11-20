package stringcensus

import (
	"path/filepath"
	"strings"
)

// CSite is one C literal that reaches a player: where it sits in the oracle and
// the fixed text it prints. It is the unit the coverage report reasons about,
// because it is the unit the C server actually prints.
type CSite struct {
	File string
	Line int
	Sink string
	// Fn is the enclosing C function.
	Fn string
	// Segments is the flat, deduplicated fixed text, in order (brief 07's
	// segment list for this literal).
	Segments []string
	// Fragments groups that text the way it prints: one entry per output line,
	// each an ordered list of segments, duplicates kept.
	Fragments [][]string
}

// Unverifiable reports whether the literal has no fixed text long enough to
// compare: every piece fell below MinSegmentLen, so no output can prove or
// disprove that it printed.
func (s CSite) Unverifiable() bool { return len(s.Segments) == 0 }

// LineContainsSegments reports whether one output line contains a fragment's
// segments in order, with arbitrary text between them. Whitespace collapses on
// both sides first, because the C output may justify text with runs of spaces
// that a normalized segment does not carry.
//
// A stored segment may itself contain a printf conversion that arrived through
// a %%-escape in the C source (a re-sprintf template such as say_spell's
// "'%%s'."): the normalizer kept the conversion as fixed text, so no transcript
// can carry it literally. Such a conversion matches arbitrary substituted text
// in order, exactly like the gaps between segments (DP-2892).
func LineContainsSegments(line string, segments []string) bool {
	line = collapseWhitespace(line)
	at := 0
	for _, seg := range segments {
		for _, piece := range segmentPieces(seg) {
			i := strings.Index(line[at:], piece)
			if i < 0 {
				return false
			}
			at += i + len(piece)
		}
	}
	return true
}

// segmentPieces splits a stored segment around printf conversions that the
// census stored as fixed text. Conversion syntax is parsed with the same
// conversionLen the normalizer uses, so a literal percent ("100%% reliable")
// stays one literal piece and only genuine conversions become wildcards. The
// first and last pieces may be empty when the segment starts or ends with a
// conversion; an empty piece matches the current position and never fails.
func segmentPieces(seg string) []string {
	if !strings.Contains(seg, "%") {
		return []string{seg}
	}
	var pieces []string
	cur := 0
	for i := 0; i < len(seg); {
		if seg[i] == '%' {
			if n, literal := conversionLen(seg[i:]); n > 0 && !literal {
				pieces = append(pieces, seg[cur:i])
				i += n
				cur = i
				continue
			}
		}
		i++
	}
	return append(pieces, seg[cur:])
}

// ExtractCSource reads the oracle C tree and returns every player-facing literal
// as a CSite, in the order extractCFiles produced it. It reuses the census's
// lexer, sink table and normalizer, so a coverage run and a census run can never
// disagree about what C prints.
func ExtractCSource(opts Options) ([]CSite, error) {
	opts = opts.withDefaults()
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, err
	}
	files, err := readSourceFiles(root, []string{opts.CDir}, isCSource)
	if err != nil {
		return nil, err
	}
	type key struct {
		file, sink, fn string
		line           int
		fragments      string
	}
	seen := map[key]bool{}
	var sites []CSite
	for _, c := range extractCFiles(files) {
		site := CSite{File: c.file, Line: c.line, Sink: c.sink, Fn: c.fn}
		var flat []string
		flatSeen := map[string]bool{}
		for _, fragment := range NormalizeLines(c.raw) {
			site.Fragments = append(site.Fragments, fragment)
			for _, seg := range fragment {
				if flatSeen[seg] {
					continue
				}
				flatSeen[seg] = true
				flat = append(flat, seg)
			}
		}
		site.Segments = flat
		k := key{file: c.file, sink: c.sink, fn: c.fn, line: c.line, fragments: strings.Join(flat, "\x1f")}
		if seen[k] {
			continue
		}
		seen[k] = true
		sites = append(sites, site)
	}
	return sites, nil
}
