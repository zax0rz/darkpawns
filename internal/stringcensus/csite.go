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

// MatchLine reports the 1-based line of block where the site's fixed text
// appears, or 0 when it does not appear.
//
// Segments of one fragment must sit on one line with arbitrary text between
// them — that text is where printf verbs and act codes were substituted. A
// later fragment must sit on a later line, because a fragment boundary came
// from a line break inside the C literal.
func (s CSite) MatchLine(block string) int {
	if s.Unverifiable() {
		return 0
	}
	lines := strings.Split(block, "\n")
	next := 0
	last := -1
	for _, fragment := range s.Fragments {
		if len(fragment) == 0 {
			continue
		}
		at := -1
		for i := next; i < len(lines); i++ {
			if LineContainsSegments(lines[i], fragment) {
				at = i
				break
			}
		}
		if at < 0 {
			return 0
		}
		last = at
		next = at + 1
	}
	if last < 0 {
		return 0
	}
	return last + 1
}

// LineContainsSegments reports whether one output line contains a fragment's
// segments in order, with arbitrary text between them. Whitespace collapses on
// both sides first, because the C output may justify text with runs of spaces
// that a normalized segment does not carry.
func LineContainsSegments(line string, segments []string) bool {
	line = collapseWhitespace(line)
	at := 0
	for _, seg := range segments {
		i := strings.Index(line[at:], seg)
		if i < 0 {
			return false
		}
		at += i + len(seg)
	}
	return true
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
