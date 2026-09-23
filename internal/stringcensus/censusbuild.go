package stringcensus

import (
	"path/filepath"
	"sort"
	"strings"
)

// buildReport turns raw candidates into the compared segment sets. It is split
// out of Run so tests can drive it with fixtures and no filesystem.
func buildReport(goSrc, cSrc, dataSrc []sourceFile, goCands, cCands []candidate, unresolved []unresolvedSite) *Report {
	goSegs := segmentsFrom(goCands)
	cSegs := segmentsFrom(cCands)

	cTexts := uniqueTexts(cSegs)
	goTexts := uniqueTexts(goSegs)
	cSet := newPatternSet(cTexts)
	goSet := newPatternSet(goTexts)

	r := &Report{
		GoFiles:        len(goSrc),
		CFiles:         len(cSrc),
		DataFiles:      len(dataSrc),
		GoCandidates:   len(goCands),
		CCandidates:    len(cCands),
		GoSegmentCount: len(goSegs),
		CSegmentCount:  len(cSegs),
		goSinkCounts:   map[string]int{},
		cSinkCounts:    map[string]int{},
		goPkgCounts:    map[string]int{},
		cFileCounts:    map[string]int{},
	}
	for _, c := range goCands {
		r.goSinkCounts[sinkBase(c.sink)]++
		r.goPkgCounts[packageOf(c.file)]++
	}
	for _, c := range cCands {
		r.cSinkCounts[cSinkBase(c.sink)]++
		r.cFileCounts[filepath.Base(c.file)]++
	}

	// Direction 1: a Go segment with no C source is go-only, a candidate for
	// invented text (R4). The brief's rule is mutual containment, case-sensitive.
	var goOnly []Segment
	for _, s := range goSegs {
		if cSet.containsSubstring(s.Text) || cSet.hasPatternInside(s.Text) {
			continue
		}
		goOnly = append(goOnly, s)
	}

	// World data is a legitimate source, so check the data before calling a
	// string invented. Matching is case-insensitive: Diku world text and Go
	// message literals do not share a case convention.
	if len(goOnly) > 0 {
		patterns := loweredUniqueTexts(goOnly)
		trie := buildPatternTrie(patterns)
		found := make([]bool, len(patterns))
		trie.visitMatches(dataCorpus(dataSrc), func(id int) { found[id] = true })
		kept := goOnly[:0]
		seen := map[string]bool{}
		for _, s := range goOnly {
			lower := strings.ToLower(s.Text)
			id := lowerID(patterns, lower)
			if id < 0 || !found[id] {
				kept = append(kept, s)
				continue
			}
			r.DataSourced++
			if !seen[lower] {
				seen[lower] = true
				r.DataSourcedSegments = append(r.DataSourcedSegments, s)
			}
		}
		goOnly = kept
	}
	r.GoOnlySegments = goOnly

	// Direction 2: a C segment no Go segment contains is c-missing, a port gap.
	for _, s := range cSegs {
		if goSet.hasPatternInside(s.Text) {
			continue
		}
		r.CMissingSegments = append(r.CMissingSegments, s)
	}

	for _, u := range unresolved {
		r.UnresolvedSites = append(r.UnresolvedSites, UnresolvedSite{
			File: u.file, Line: u.line, Sink: u.sink, Expr: u.expr,
		})
	}

	r.GoOnly = len(r.GoOnlySegments)
	r.CMissing = len(r.CMissingSegments)
	r.goOnlyByFile = countByFile(r.GoOnlySegments)
	r.cMissingByFile = countByFile(r.CMissingSegments)
	return r
}

// lowerID maps a lowered segment back to its trie id. patterns is sorted, so a
// binary search is exact and cheap.
func lowerID(patterns []string, lower string) int {
	i := sort.SearchStrings(patterns, lower)
	if i < len(patterns) && patterns[i] == lower {
		return i
	}
	return -1
}

// segmentsFrom normalizes every candidate and drops duplicate
// (segment, file, line, sink, function) rows.
func segmentsFrom(cands []candidate) []Segment {
	type key struct {
		text, file, sink, fn string
		line                 int
	}
	seen := map[key]bool{}
	out := make([]Segment, 0, len(cands))
	for _, c := range cands {
		for _, text := range Normalize(c.raw) {
			k := key{text: text, file: c.file, sink: c.sink, fn: c.fn, line: c.line}
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, Segment{Text: text, File: c.file, Line: c.line, Sink: c.sink, Fn: c.fn})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		if out[i].Sink != out[j].Sink {
			return out[i].Sink < out[j].Sink
		}
		return out[i].Text < out[j].Text
	})
	return out
}

// uniqueTexts returns the distinct segment texts, sorted.
func uniqueTexts(segs []Segment) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		if seen[s.Text] {
			continue
		}
		seen[s.Text] = true
		out = append(out, s.Text)
	}
	sort.Strings(out)
	return out
}

func loweredUniqueTexts(segs []Segment) []string {
	texts := uniqueTexts(segs)
	for i, t := range texts {
		texts[i] = strings.ToLower(t)
	}
	sort.Strings(texts)
	return texts
}

func countByFile(segs []Segment) map[string]int {
	m := map[string]int{}
	for _, s := range segs {
		m[s.File]++
	}
	return m
}

// sinkBase strips the receiver dot and any "(via x)" suffix from a report sink
// label, so per-sink counts aggregate by table entry.
func sinkBase(label string) string {
	label = strings.TrimPrefix(label, ".")
	if i := strings.IndexByte(label, '('); i > 0 {
		label = strings.TrimSpace(label[:i])
	}
	return label
}

// cSinkBase strips the "buffered:" tag so a sprintf literal counts under the
// call in the sink table, not under a separate pseudo-sink.
func cSinkBase(label string) string {
	return strings.TrimPrefix(label, "buffered:")
}

// packageOf reports the Go package directory a file belongs to.
func packageOf(rel string) string {
	return filepath.ToSlash(filepath.Dir(rel))
}
