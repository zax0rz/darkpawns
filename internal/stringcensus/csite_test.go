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

func TestLineContainsSegmentsResprintfTemplateWildcard(t *testing.T) {
	t.Parallel()
	// say_spell's re-sprintf template stores a literal %s; the rendered
	// transcript substitutes the spell's incantation for it (DP-2892).
	seg := "stares at you and utters the words, '%s'."
	if !LineContainsSegments("Chtwo stares at you and utters the words, 'vrogg'.", []string{seg}) {
		t.Fatalf("template segment should match substituted output")
	}
	if !LineContainsSegments("Chtwo stares at you and utters the words, 'marubhotfihuab'.", []string{seg}) {
		t.Fatalf("template segment should match any substituted word")
	}
	// The pieces must appear in order: missing the trailing quote-tail fails.
	if LineContainsSegments("Chtwo stares at you and utters the words, vrogg", []string{seg}) {
		t.Fatalf("line missing the quote tail must not match")
	}
	if LineContainsSegments("stares at you and utters nothing.", []string{seg}) {
		t.Fatalf("line missing the leading piece must not match")
	}
}

func TestLineContainsSegmentsLiteralPercentStaysLiteral(t *testing.T) {
	t.Parallel()
	// An ordinary %% escape that is not followed by a conversion stays fixed
	// text and matches literally, without wildcarding.
	seg := "100% reliable"
	if !LineContainsSegments("The seal reads: 100% reliable.", []string{seg}) {
		t.Fatalf("literal percent segment should match literally")
	}
	if LineContainsSegments("The seal reads: 100x reliable.", []string{seg}) {
		t.Fatalf("literal percent segment must not wildcard")
	}
	if LineContainsSegments("The seal reads: 100 reliable.", []string{seg}) {
		t.Fatalf("literal percent must be present, not skipped")
	}
}

func TestSegmentPieces(t *testing.T) {
	t.Parallel()
	if got := segmentPieces("plain segment"); len(got) != 1 || got[0] != "plain segment" {
		t.Fatalf("conversion-free segment stays one piece: %q", got)
	}
	got := segmentPieces("and utters the words, '%s'.")
	want := []string{"and utters the words, '", "'."}
	if len(got) != len(want) {
		t.Fatalf("template split = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("piece %d = %q, want %q", i, got[i], want[i])
		}
	}
	// A literal percent with no conversion letter stays whole.
	if got := segmentPieces("100% reliable"); len(got) != 1 {
		t.Fatalf("literal percent stays one piece: %q", got)
	}
	// A conversion at both ends produces empty outer pieces that never fail.
	got = segmentPieces("%s wears off.")
	if len(got) != 2 || got[0] != "" || got[1] != " wears off." {
		t.Fatalf("leading-conversion split = %q", got)
	}
}
