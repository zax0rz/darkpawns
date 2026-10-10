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
