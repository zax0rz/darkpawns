package stringcensus

import (
	"bytes"
	"index/suffixarray"
	"sort"
	"strings"
)

// patternSet answers the two containment questions the matcher asks of one
// corpus of normalized segments:
//
//   - does some pattern contain this text?
//   - does this text contain some pattern?
//
// The first is a suffix-array lookup over the patterns joined by a separator
// that cannot occur inside a segment. The second walks a trie of the patterns
// from every position of the text, which costs O(len(text)) with the short
// common prefixes English sentences produce. Neither needs a full automaton.
type patternSet struct {
	sa   *suffixarray.Index
	trie *patternTrie
}

func newPatternSet(patterns []string) *patternSet {
	sorted := append([]string(nil), patterns...)
	sort.Strings(sorted)
	joined := []byte(strings.Join(sorted, "\x00"))
	return &patternSet{
		sa:   suffixarray.New(joined),
		trie: buildPatternTrie(sorted),
	}
}

// containsSubstring reports whether some pattern contains s.
func (p *patternSet) containsSubstring(s string) bool {
	if s == "" {
		return false
	}
	return len(p.sa.Lookup([]byte(s), 1)) > 0
}

// hasPatternInside reports whether some pattern is a substring of s.
func (p *patternSet) hasPatternInside(s string) bool {
	id, ok := p.trie.firstMatch(s)
	return ok && id >= 0
}

// patternTrie is a byte trie over the pattern corpus. Nodes are stored flat;
// children are maps because the alphabet of normalized prose is small but
// sparse.
type patternTrie struct {
	children []map[byte]int
	terminal []int
}

func buildPatternTrie(patterns []string) *patternTrie {
	t := &patternTrie{
		children: []map[byte]int{nil},
		terminal: []int{-1},
	}
	for id, p := range patterns {
		node := int(0)
		for i := 0; i < len(p); i++ {
			c := p[i]
			next, ok := t.children[node][c]
			if !ok {
				next = int(len(t.children))
				t.children[node] = withChild(t.children[node], c, next)
				t.children = append(t.children, nil)
				t.terminal = append(t.terminal, -1)
			}
			node = next
		}
		t.terminal[node] = int(id)
	}
	return t
}

func withChild(m map[byte]int, c byte, node int) map[byte]int {
	if m == nil {
		m = make(map[byte]int, 4)
	}
	m[c] = node
	return m
}

// firstMatch returns the id of any pattern that occurs inside s.
func (t *patternTrie) firstMatch(s string) (int, bool) {
	for i := 0; i < len(s); i++ {
		node := int(0)
		for j := i; j < len(s); j++ {
			next, ok := t.children[node][s[j]]
			if !ok {
				break
			}
			node = next
			if id := t.terminal[node]; id >= 0 {
				return id, true
			}
		}
	}
	return -1, false
}

// visitMatches reports every pattern id occurring inside s. The caller dedupes;
// a repeated line in a 7 MB world tree would otherwise report the same id
// thousands of times.
func (t *patternTrie) visitMatches(s string, visit func(id int)) {
	for i := 0; i < len(s); i++ {
		node := int(0)
		for j := i; j < len(s); j++ {
			next, ok := t.children[node][s[j]]
			if !ok {
				break
			}
			node = next
			if id := t.terminal[node]; id >= 0 {
				visit(id)
			}
		}
	}
}

// dataCorpus builds the searchable world-data text: every file under the data
// directories, with whitespace runs collapsed to single spaces so a sentence
// wrapped across source lines still matches, and lower-cased because Diku world
// text is written in a different case convention than Go message literals.
//
// Normalized segments never contain tabs or repeated spaces (see Normalize), so
// scanning the collapsed corpus also covers the raw one: a segment that occurs
// in the raw text occurs here.
func dataCorpus(files []sourceFile) string {
	var b bytes.Buffer
	for _, f := range files {
		b.WriteString(f.data)
		b.WriteByte(' ')
	}
	return strings.ToLower(strings.Join(strings.Fields(b.String()), " "))
}
