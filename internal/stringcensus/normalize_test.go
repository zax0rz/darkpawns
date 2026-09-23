package stringcensus

import (
	"reflect"
	"testing"
)

func TestNormalizeSplitsAtLineBreaks(t *testing.T) {
	got := Normalize("You are hungry.\r\nYou are thirsty.\n\rYou are drunk.\nYou are sober.\rYou are ill.")
	want := []string{"You are hungry.", "You are thirsty.", "You are drunk.", "You are sober.", "You are ill."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Normalize = %#v, want %#v", got, want)
	}
}

func TestNormalizeSplitsAtFormatVerbs(t *testing.T) {
	got := Normalize("You strike %s and deal %d points of crushing damage with your axe.")
	want := []string{"You strike", "and deal", "points of crushing damage with your axe."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Normalize = %#v, want %#v", got, want)
	}
}

func TestNormalizeTreatsDoublePercentAsLiteral(t *testing.T) {
	got := Normalize("You have 100%% of your hit points left.")
	want := []string{"You have 100% of your hit points left."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Normalize = %#v, want %#v", got, want)
	}
}

func TestNormalizeKeepsBarePercent(t *testing.T) {
	got := Normalize("Your load is 50% of the maximum weight.")
	want := []string{"Your load is 50% of the maximum weight."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Normalize = %#v, want %#v", got, want)
	}
}

func TestNormalizeSplitsAtActCodes(t *testing.T) {
	got := Normalize("$n laughs at $N and waves $p proudly in the air.")
	want := []string{"laughs at", "and waves", "proudly in the air."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Normalize = %#v, want %#v", got, want)
	}
}

func TestNormalizeTreatsDoubleDollarAsLiteral(t *testing.T) {
	got := Normalize("The price is $$1000 gold pieces exactly.")
	want := []string{"The price is $1000 gold pieces exactly."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Normalize = %#v, want %#v", got, want)
	}
}

func TestNormalizeDropsColourEscapes(t *testing.T) {
	got := Normalize("\x1b[36mThe gate hums with a pale light.\x1b[0m")
	want := []string{"The gate hums with a pale light."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Normalize = %#v, want %#v", got, want)
	}
}

func TestNormalizeSplitsAroundColourEscapes(t *testing.T) {
	got := Normalize("The gate\x1b[36m shimmers\x1b[0m and then is still.")
	want := []string{"The gate", "shimmers", "and then is still."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Normalize = %#v, want %#v", got, want)
	}
}

func TestNormalizeCollapsesWhitespace(t *testing.T) {
	got := Normalize("Return to the temple and QUIT to leave the game and keep\n     your equipment.")
	want := []string{"Return to the temple and QUIT to leave the game and keep", "your equipment."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Normalize = %#v, want %#v", got, want)
	}
}

func TestNormalizeDropsShortSegments(t *testing.T) {
	// "Ok." and "No." are below the 8-character floor; the long segment survives.
	got := Normalize("Ok.\r\nNo.\r\nYou can not do that here.\r\n")
	want := []string{"You can not do that here."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Normalize = %#v, want %#v", got, want)
	}
}

func TestNormalizeDeduplicatesWithinLiteral(t *testing.T) {
	got := Normalize("The door is closed.\r\nThe door is closed.\r\n")
	want := []string{"The door is closed."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Normalize = %#v, want %#v", got, want)
	}
}

func TestNormalizeEmpty(t *testing.T) {
	if got := Normalize(""); len(got) != 0 {
		t.Fatalf("Normalize(\"\") = %#v, want empty", got)
	}
}

func TestConversionLen(t *testing.T) {
	cases := []struct {
		in      string
		n       int
		literal bool
	}{
		{"%s", 2, false},
		{"%-10s", 5, false},
		{"%ld", 3, false},
		{"%3d", 3, false},
		{"%.2f", 4, false},
		{"%+v", 3, false},
		{"%[1]s", 5, false},
		{"%%", 2, true},
		{"%", 0, false},
		{"% off", 0, false},
		{"%q", 2, false},
	}
	for _, tc := range cases {
		n, literal := conversionLen(tc.in)
		if n != tc.n || literal != tc.literal {
			t.Errorf("conversionLen(%q) = (%d, %v), want (%d, %v)", tc.in, n, literal, tc.n, tc.literal)
		}
	}
}
