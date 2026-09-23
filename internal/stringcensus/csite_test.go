package stringcensus

import (
	"reflect"
	"testing"
)

func TestNormalizeLinesSplitsLinesIntoSegments(t *testing.T) {
	got := NormalizeLines("You strike %s and deal damage.\r\nYou have been defeated.\n")
	want := [][]string{
		{"You strike", "and deal damage."},
		{"You have been defeated."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeLines = %#v, want %#v", got, want)
	}
}

func TestNormalizeLinesKeepsDuplicatesAndOrder(t *testing.T) {
	got := NormalizeLines("The gate hums and the gate closes.\r\nThe gate hums and the gate closes.")
	want := [][]string{
		{"The gate hums and the gate closes."},
		{"The gate hums and the gate closes."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeLines = %#v, want %#v", got, want)
	}
	// Normalize is the flat, deduplicated view of the same text.
	if flat := Normalize("The gate hums and the gate closes.\r\nThe gate hums and the gate closes."); len(flat) != 1 {
		t.Fatalf("Normalize = %#v, want one segment", flat)
	}
}

func TestNormalizeLinesSkipsEmptyFragments(t *testing.T) {
	// "Ok." is below the floor, and an act code alone leaves nothing either.
	got := NormalizeLines("Ok.\r\n$n\r\nYou can not do that here.")
	want := [][]string{{"You can not do that here."}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeLines = %#v, want %#v", got, want)
	}
}

func TestNormalizeAndNormalizeLinesAgree(t *testing.T) {
	raw := "Return to the temple and QUIT to leave" +
		" the game and keep your equipment.\r\n$n has left the game."
	if flat, ordered := Normalize(raw), NormalizeLines(raw); len(flat) != 2 || len(ordered) != 2 {
		t.Fatalf("Normalize = %#v, NormalizeLines = %#v", flat, ordered)
	}
}

func siteFromRaw(raw string) CSite {
	site := CSite{File: "src/x.c", Line: 1, Sink: "send_to_char"}
	for _, fragment := range NormalizeLines(raw) {
		site.Fragments = append(site.Fragments, fragment)
		site.Segments = append(site.Segments, fragment...)
	}
	return site
}

func TestSiteMatchLineMatchesVerbsAsWildcards(t *testing.T) {
	site := siteFromRaw("Saving %s.\r\nYou have %d gold pieces on hand.\r\n")
	if site.Unverifiable() {
		t.Fatal("the second fragment is at or above the floor, so the site is verifiable")
	}
	block := "Saving Frodo.\nYou have 1234 gold pieces on hand.\n"
	if got := site.MatchLine(block); got != 2 {
		t.Fatalf("MatchLine = %d, want 2 (the line of the last fragment)", got)
	}
	if got := site.MatchLine("Saving Frodo.\nYou have nothing to speak of.\n"); got != 0 {
		t.Fatalf("MatchLine = %d, want 0 when the fixed text is absent", got)
	}
}

func TestSiteMatchLineMatchesActCodesAsWildcards(t *testing.T) {
	site := siteFromRaw("$n has left the game.\r\n")
	if got := site.MatchLine("Frodo has left the game.\n"); got != 1 {
		t.Fatalf("MatchLine = %d, want 1", got)
	}
	if got := site.MatchLine("Nobody here did anything.\n"); got != 0 {
		t.Fatalf("MatchLine = %d, want 0", got)
	}
}

func TestSiteMatchLineRequiresSegmentsInOrder(t *testing.T) {
	site := siteFromRaw("You have to type quit--no less, to quit!\r\n")
	if got := site.MatchLine("You have to type quit--no less, to quit!\n"); got != 1 {
		t.Fatalf("MatchLine = %d, want 1", got)
	}
	if got := site.MatchLine("to quit! You have to type quit--no less,\n"); got != 0 {
		t.Fatalf("MatchLine = %d, want 0: the words are out of order", got)
	}
}

func TestSiteMatchLineRequiresLaterLinesForLaterFragments(t *testing.T) {
	site := siteFromRaw("You are hungry.\r\nYou are thirsty.\r\n")
	if got := site.MatchLine("You are hungry.\nYou are thirsty.\n"); got != 2 {
		t.Fatalf("MatchLine = %d, want 2", got)
	}
	// Both fragments on one line cannot happen: the literal prints two lines.
	if got := site.MatchLine("You are hungry. You are thirsty.\n"); got != 0 {
		t.Fatalf("MatchLine = %d, want 0", got)
	}
}

func TestSiteMatchLineCollapsesOutputWhitespace(t *testing.T) {
	site := siteFromRaw("No way!  You're fighting for your life!\r\n")
	if got := site.MatchLine("No way!  You're fighting for your life!\n"); got != 1 {
		t.Fatalf("MatchLine = %d, want 1", got)
	}
}

func TestSiteUnverifiableWhenEverySegmentIsShort(t *testing.T) {
	site := siteFromRaw("Ok.\r\nNo.\r\n")
	if !site.Unverifiable() {
		t.Fatal("a literal with no segment at or above the floor is unverifiable")
	}
	if got := site.MatchLine("Ok.\nNo.\n"); got != 0 {
		t.Fatalf("MatchLine = %d, want 0 for an unverifiable site", got)
	}
}

func TestLineContainsSegments(t *testing.T) {
	cases := []struct {
		line string
		segs []string
		want bool
	}{
		{"You strike the rat and deal a lot of damage.", []string{"You strike", "and deal a lot of damage."}, true},
		{"You strike the rat and deal a lot of damage.", []string{"You strike", "and deal nothing."}, false},
		{"and deal a lot of damage. You strike", []string{"You strike", "and deal a lot of damage."}, false},
		{"Saving Frodo.", []string{"Saving"}, true},
		{"", []string{"Saving"}, false},
		{"anything at all", nil, true},
	}
	for _, tc := range cases {
		if got := LineContainsSegments(tc.line, tc.segs); got != tc.want {
			t.Errorf("LineContainsSegments(%q, %v) = %v, want %v", tc.line, tc.segs, got, tc.want)
		}
	}
}

func TestExtractCSourceReadsTheFixtureOracle(t *testing.T) {
	sites, err := ExtractCSource(Options{Root: "testdata/census", CDir: "src"})
	if err != nil {
		t.Fatalf("ExtractCSource: %v", err)
	}
	byText := map[string]CSite{}
	for _, s := range sites {
		for _, seg := range s.Segments {
			byText[seg] = s
		}
	}
	goodbye, ok := byText["Goodbye, friend.. Come back soon!"]
	if !ok {
		t.Fatalf("fixture site not found; got %d sites", len(sites))
	}
	if goodbye.Fn != "do_quit" || goodbye.Sink != "send_to_char" {
		t.Fatalf("site = %+v, want do_quit / send_to_char", goodbye)
	}
	if ln := goodbye.MatchLine("Frodo waves.\nGoodbye, friend.. Come back soon!\n"); ln != 2 {
		t.Fatalf("MatchLine = %d, want 2", ln)
	}
}
