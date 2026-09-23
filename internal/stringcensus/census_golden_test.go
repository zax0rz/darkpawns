package stringcensus

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden regenerates testdata/golden with `go test ./internal/stringcensus/ -update`.
var updateGolden = flag.Bool("update", false, "rewrite the census golden files")

// TestGoldenCensus runs the whole census over the fixture tree in
// testdata/census and compares every generated report byte for byte. The
// fixture is deliberately small: one Go file, one C file, one string-table C
// file and one world file, covering a C-sourced string, an invented string, a
// data-sourced string, a fmt.Sprintf, a concatenation, an unresolved local, a
// c-missing string and a C table entry.
func TestGoldenCensus(t *testing.T) {
	report, err := Run(Options{
		Root:     "testdata/census",
		GoDirs:   []string{"go/pkg/game"},
		CDir:     "src",
		DataDirs: []string{"lib/world"},
		OutDir:   "out",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The fixture's expectations, stated as assertions first so a broken golden
	// file cannot silently bless a broken census.
	assertGoOnly(t, report, "The gate hums with a pale light.")
	assertGoOnly(t, report, "You strike") // fmt.Sprintf fixed text
	assertGoOnly(t, report, "and deal a lot of damage.")
	assertCMissing(t, report, "has left the game.")
	assertCMissing(t, report, "You have to type quit--no less, to quit!")
	assertCMissing(t, report, "one-quarter full(waxing)")
	assertNotReported(t, report, "Goodbye, friend.. Come back soon!")
	assertNotReported(t, report, "Return to the temple and QUIT to leave the game and keep your equipment.")
	assertNotReported(t, report, "A cold wind blows through the ruined hall.")
	if len(report.UnresolvedSites) != 1 || report.UnresolvedSites[0].Expr != "body" {
		t.Fatalf("unresolved = %+v, want one row for body", report.UnresolvedSites)
	}

	goldenDir := filepath.Join("testdata", "golden")
	rendered := report.Render()
	for name, got := range rendered {
		goldenPath := filepath.Join(goldenDir, name)
		if *updateGolden {
			if err := os.MkdirAll(goldenDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("read golden %s: %v (run go test ./internal/stringcensus/ -update)", name, err)
		}
		if string(want) != got {
			t.Errorf("%s does not match the golden file:\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
		}
	}
}

func assertGoOnly(t *testing.T, r *Report, text string) {
	t.Helper()
	for _, s := range r.GoOnlySegments {
		if s.Text == text {
			return
		}
	}
	t.Errorf("go-only report is missing %q (go-only: %v)", text, textsOf(r.GoOnlySegments))
}

func assertCMissing(t *testing.T, r *Report, text string) {
	t.Helper()
	for _, s := range r.CMissingSegments {
		if s.Text == text {
			return
		}
	}
	t.Errorf("c-missing report is missing %q (c-missing: %v)", text, textsOf(r.CMissingSegments))
}

func assertNotReported(t *testing.T, r *Report, text string) {
	t.Helper()
	for _, s := range r.GoOnlySegments {
		if s.Text == text {
			t.Errorf("%q was reported go-only, but it has a C or data source", text)
		}
	}
	for _, s := range r.CMissingSegments {
		if s.Text == text {
			t.Errorf("%q was reported c-missing, but Go contains it", text)
		}
	}
}

func textsOf(segs []Segment) []string {
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		out = append(out, s.Text)
	}
	return out
}
