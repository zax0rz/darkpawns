package stringcensus

import (
	"sort"
	"testing"
)

func TestSiteMatcherScansBlocksPerLine(t *testing.T) {
	sites := []CSite{
		siteFromRaw("$n has left the game.\r\n"),
		siteFromRaw("You are hungry.\r\nYou are thirsty.\r\n"),
		siteFromRaw("You have been poisoned by the spider's bite.\r\n"),
	}
	m := NewSiteMatcher(sites)

	type hit struct {
		site, line int
	}
	scan := func(block string) []hit {
		var hits []hit
		m.ScanBlock(block, func(site, line int) { hits = append(hits, hit{site, line}) })
		sort.Slice(hits, func(i, j int) bool { return hits[i].site < hits[j].site })
		return hits
	}

	got := scan("Frodo has left the game.\n")
	if len(got) != 1 || got[0] != (hit{site: 0, line: 1}) {
		t.Fatalf("first block hits = %+v, want only site 0 on line 1", got)
	}

	got = scan("You are hungry.\nYou are thirsty.\n")
	if len(got) != 1 || got[0] != (hit{site: 1, line: 2}) {
		t.Fatalf("second block hits = %+v, want only site 1 on line 2", got)
	}

	// State resets per block: a site covered in one block is not carried over,
	// and the never-printed site is still never reported.
	if got := scan("Nothing in this block matches any of them.\n"); len(got) != 0 {
		t.Fatalf("third block hits = %+v, want none", got)
	}

	// The same block scanned twice reports again, which is what lets the caller
	// see a site covered by more than one scenario.
	if got := scan("Frodo has left the game.\n"); len(got) != 1 {
		t.Fatalf("repeated block hits = %+v, want the site again", got)
	}
}

func TestSiteMatcherIgnoresUnverifiableSites(t *testing.T) {
	m := NewSiteMatcher([]CSite{siteFromRaw("Ok.\r\nNo.\r\n")})
	called := false
	m.ScanBlock("Ok.\nNo.\n", func(int, int) { called = true })
	if called {
		t.Fatal("a site with no segment at or above the floor must never be reported covered")
	}
}

func TestSiteMatcherMatchesFragmentsInOrder(t *testing.T) {
	m := NewSiteMatcher([]CSite{siteFromRaw("You are hungry.\r\nYou are thirsty.\r\n")})
	var lines []int
	m.ScanBlock("You are thirsty.\nYou are hungry.\n", func(_, line int) { lines = append(lines, line) })
	if len(lines) != 0 {
		t.Fatalf("hits = %v, want none: the lines are in the wrong order", lines)
	}
}
