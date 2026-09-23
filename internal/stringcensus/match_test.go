package stringcensus

import "testing"

func TestPatternSetContainment(t *testing.T) {
	set := newPatternSet([]string{
		"Goodbye, friend.. Come back soon!",
		"Sorry, but you cannot do that here!",
	})
	cases := []struct {
		text            string
		wantContains    bool
		wantPatternInIt bool
	}{
		{"Goodbye, friend.. Come back soon!", true, true},
		{"Come back soon!", true, false},
		{"friend.. Come back", true, false},
		{"He said: Goodbye, friend.. Come back soon! and left.", false, true},
		{"Something else entirely.", false, false},
	}
	for _, tc := range cases {
		if got := set.containsSubstring(tc.text); got != tc.wantContains {
			t.Errorf("containsSubstring(%q) = %v, want %v", tc.text, got, tc.wantContains)
		}
		if got := set.hasPatternInside(tc.text); got != tc.wantPatternInIt {
			t.Errorf("hasPatternInside(%q) = %v, want %v", tc.text, got, tc.wantPatternInIt)
		}
	}
}

func TestPatternSetEmpty(t *testing.T) {
	set := newPatternSet(nil)
	if set.containsSubstring("anything at all") {
		t.Error("empty set reported a containment")
	}
	if set.hasPatternInside("anything at all") {
		t.Error("empty set reported a pattern inside")
	}
}

func TestPatternSetVisitMatchesIsDeduplicable(t *testing.T) {
	set := newPatternSet([]string{"a repeated line", "another repeated line"})
	text := "a repeated line another repeated line a repeated line"
	seen := make([]bool, 2)
	set.trie.visitMatches(text, func(id int32) { seen[id] = true })
	if !seen[0] || !seen[1] {
		t.Fatalf("visitMatches saw %v, want both patterns", seen)
	}
}

func TestDataCorpusCollapsesAndLowers(t *testing.T) {
	files := []sourceFile{
		{rel: "lib/world/wld/1.wld", data: "The Gate   hums\nwith a PALE light.\n"},
	}
	got := dataCorpus(files)
	if got != "the gate hums with a pale light." {
		t.Fatalf("dataCorpus = %q", got)
	}
	if !newPatternSet([]string{"hums with a pale light."}).hasPatternInside(got) {
		t.Fatal("collapsed corpus should contain a wrapped sentence")
	}
}
