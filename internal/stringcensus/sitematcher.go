package stringcensus

import (
	"strings"
)

// SiteMatcher matches C sites against normalized output text.
//
// Scanning is line-driven, not site-driven: one pass over a dump's bytes costs
// a trie walk plus a lookup per segment that actually occurs, instead of
// sites × lines comparisons over a 900-scenario census. The per-line segment
// list is a cheap prefilter; a site only advances when its next expected
// fragment really matches the line.
type SiteMatcher struct {
	sites []CSite
	state []siteState
	// refs maps a fragment's first segment to the fragments that start with it.
	// It is a necessary condition for the fragment to match a line, so it gates
	// the expensive ordered check.
	refs map[string][]fragRef
	// texts are the distinct segment texts the trie is built over; trie ids
	// index this slice.
	texts []string
	trie  *patternTrie
	// hit and present are per-line scratch; visited marks results for the block
	// being scanned and is reset by the caller between blocks.
	hit     []bool
	present []int
}

type fragRef struct {
	site     int
	fragment int
}

type siteState struct {
	cursor int
	done   bool
}

// NewSiteMatcher builds a matcher over the sites of one C tree.
func NewSiteMatcher(sites []CSite) *SiteMatcher {
	m := &SiteMatcher{
		sites: sites,
		state: make([]siteState, len(sites)),
		refs:  map[string][]fragRef{},
	}
	seen := map[string]bool{}
	for i, site := range sites {
		for fi, fragment := range site.Fragments {
			if len(fragment) == 0 {
				continue
			}
			first := fragment[0]
			m.refs[first] = append(m.refs[first], fragRef{site: i, fragment: fi})
			if !seen[first] {
				seen[first] = true
				m.texts = append(m.texts, first)
			}
		}
	}
	m.trie = buildPatternTrie(m.texts)
	m.hit = make([]bool, len(m.texts))
	return m
}

// ScanBlock advances every site against one normalized output block. visit is
// called once per site the block completes, with the 1-based line of the site's
// last fragment.
func (m *SiteMatcher) ScanBlock(block string, visit func(site, line int)) {
	for i := range m.state {
		m.state[i] = siteState{}
	}
	for lineNo, line := range strings.Split(block, "\n") {
		line = collapseWhitespace(line)
		if line == "" {
			continue
		}
		m.trie.visitMatches(line, func(id int) {
			if !m.hit[id] {
				m.hit[id] = true
				m.present = append(m.present, id)
			}
		})
		for _, id := range m.present {
			for _, ref := range m.refs[m.texts[id]] {
				st := &m.state[ref.site]
				if st.done || ref.fragment != st.cursor {
					continue
				}
				site := &m.sites[ref.site]
				if !LineContainsSegments(line, site.Fragments[ref.fragment]) {
					continue
				}
				st.cursor++
				if st.cursor == len(site.Fragments) {
					st.done = true
					visit(ref.site, lineNo+1)
				}
			}
		}
		for _, id := range m.present {
			m.hit[id] = false
		}
		m.present = m.present[:0]
	}
}
