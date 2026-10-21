package oraclediff

import (
	"strings"
	"testing"
)

// TestCompileTimeStampNormalizesBootMinute pins the mask approved for
// SCMD_VERSION's immortal line: load_compile_time() stamps each engine's own
// process wall clock at boot (src/version.c:12-62), minute-granular, so two
// boots that straddle a minute differ only in that stamp.
func TestCompileTimeStampNormalizesBootMinute(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "adjacent minutes",
			a:    "Dark Pawns 2.3-\r\nCompile Time: October 21, 2026   09:27\r\n",
			b:    "Dark Pawns 2.3-\r\nCompile Time: October 21, 2026   09:28\r\n",
		},
		{
			name: "adjacent days",
			a:    "Compile Time: October 31, 2026   23:59\r\n",
			b:    "Compile Time: November 1, 2026   00:00\r\n",
		},
		{
			name: "single-digit day and hour",
			a:    "Compile Time: February 5, 2026   09:07\r\n",
			b:    "Compile Time: February 5, 2026   09:08\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, want := Normalize(tt.a), Normalize(tt.b)
			if got != want {
				t.Fatalf("boot-minute boundary left unnormalized: %q vs %q", got, want)
			}
			if want := "Dark Pawns 2.3-\nCompile Time: <COMPILE_TIME>\n"; tt.name == "adjacent minutes" && got != want {
				t.Fatalf("normalized bytes = %q, want %q", got, want)
			}
		})
	}
}

// TestCompileTimeStampKeepsFormatDivergences proves the mask is anchored to C's
// exact shape: anything else at that slot is a real player-visible divergence
// and must still fail the comparison.
func TestCompileTimeStampKeepsFormatDivergences(t *testing.T) {
	canonical := "Compile Time: October 21, 2026   09:27"
	divergent := []struct {
		name string
		line string
	}{
		{"abbreviated month", "Compile Time: Oct 21, 2026   09:27"},
		{"single space before the clock", "Compile Time: October 21, 2026 09:27"},
		{"seconds", "Compile Time: October 21, 2026   09:27:11"},
		{"missing clock", "Compile Time: October 21, 2026"},
		{"invented label", "Built with: go1.26.10"},
		{"stamp embedded in prose", "say Compile Time: October 21, 2026   09:27"},
		{"unknown stamp", "Compile Time: unknown"},
	}
	for _, d := range divergent {
		t.Run(d.name, func(t *testing.T) {
			if Normalize(d.line+"\r\n") == Normalize(canonical+"\r\n") {
				t.Fatalf("mask hid a divergence: %q normalized like %q", d.line, canonical)
			}
		})
	}
}

// TestCompileTimeStampAdmitsPaddedDay records the one thing the approved
// pattern (\d{1,2}) cannot tell apart: C formats the day with %d, so it never
// emits "01", and a port that did would be masked too. Every other shape
// difference — month abbreviation, separator widths, seconds, label — still
// fails, which is what the mask must not hide.
func TestCompileTimeStampAdmitsPaddedDay(t *testing.T) {
	if Normalize("Compile Time: October 01, 2026   09:27\r\n") != Normalize("Compile Time: October 1, 2026   09:27\r\n") {
		t.Fatal(`pattern \d{1,2} is expected to admit a zero-padded day; the approved pattern keeps it masked`)
	}
}

// TestCompileTimeStampLeavesSurroundingText pins that the mask touches only the
// stamp line, so the mortal line and neighbouring output stay comparable.
func TestCompileTimeStampLeavesSurroundingText(t *testing.T) {
	got := Normalize("Dark Pawns 2.3-\r\nCompile Time: October 21, 2026   09:27\r\n\r\nYou feel better.\r\n")
	want := "Dark Pawns 2.3-\nCompile Time: <COMPILE_TIME>\n\nYou feel better.\n"
	if got != want {
		t.Fatalf("normalized bytes = %q, want %q", got, want)
	}
	if !strings.Contains(got, "Dark Pawns 2.3-") {
		t.Fatal("mask swallowed the revision line")
	}
}
