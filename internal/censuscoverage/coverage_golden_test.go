package censuscoverage

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/internal/stringcensus"
)

// updateGolden regenerates testdata/golden with
// `go test ./internal/censuscoverage/ -update`.
var updateGolden = flag.Bool("update", false, "rewrite the coverage golden files")

// fixtureReport computes coverage for the fixture C tree against the fixture
// dump: two dumped scenarios, one with C blocks and one empty.
func fixtureReport(t *testing.T) *Report {
	t.Helper()
	sites, err := stringcensus.ExtractCSource(stringcensus.Options{
		Root:   "testdata",
		CDir:   "src",
		OutDir: "out",
	})
	if err != nil {
		t.Fatalf("ExtractCSource: %v", err)
	}
	scenarios, err := loadFixtureDump(t)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	return Compute(sites, scenarios)
}

// loadFixtureDump reads the fixture dump through an os.Root, the same way the
// CLI does.
func loadFixtureDump(t *testing.T) ([]Scenario, error) {
	t.Helper()
	root, err := OpenDump(filepath.Join("testdata", "dump"))
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = root.Close() })
	return LoadDir(root)
}

func verdictOf(sc *SiteCoverage) string {
	switch {
	case sc.Site.Unverifiable():
		return "unverifiable"
	case sc.Covered():
		return "yes"
	default:
		return "no"
	}
}

// assertSegmentVerdict finds the site that owns a segment and checks its
// verdict.
func assertSegmentVerdict(t *testing.T, r *Report, segment, want string) {
	t.Helper()
	for i := range r.Sites {
		sc := &r.Sites[i]
		for _, seg := range sc.Site.Segments {
			if seg != segment {
				continue
			}
			if got := verdictOf(sc); got != want {
				t.Errorf("segment %q verdict = %s, want %s (site %s:%d)", segment, got, want, sc.Site.File, sc.Site.Line)
			}
			return
		}
	}
	t.Errorf("segment %q not found in the report", segment)
}

// assertSiteVerdict checks one site by position, which is how an unverifiable
// literal (one with no segment at all) has to be asserted.
func assertSiteVerdict(t *testing.T, r *Report, file string, line int, want string) {
	t.Helper()
	for i := range r.Sites {
		sc := &r.Sites[i]
		if sc.Site.File == file && sc.Site.Line == line {
			if got := verdictOf(sc); got != want {
				t.Errorf("site %s:%d verdict = %s, want %s", file, line, got, want)
			}
			return
		}
	}
	t.Errorf("site %s:%d not found in the report", file, line)
}

// TestGoldenCoverage runs the whole coverage computation over the fixture dump
// and fixture C tree and compares both generated reports byte for byte.
func TestGoldenCoverage(t *testing.T) {
	report := fixtureReport(t)

	// The brief's three unit cases, asserted before the golden comparison so a
	// broken golden file cannot bless a broken matcher.
	assertSegmentVerdict(t, report, "gold pieces on hand.", "yes") // a %d in the middle is a wildcard
	assertSegmentVerdict(t, report, "has left the game.", "yes")   // an act code split the line
	assertSegmentVerdict(t, report, "You have to type quit--no less, to quit!", "no")

	// That last sentence is in the Go fixture tree (testdata/go/...): Go output
	// must never count as coverage of a C site.
	if !goFixtureMentions(t, "You have to type quit--no less, to quit!") {
		t.Fatal("fixture drift: the Go tree no longer contains the sentence this test is about")
	}

	// Both lines of a two-line literal must appear, in order, on later lines.
	assertSegmentVerdict(t, report, "You are hungry.", "yes")
	assertSegmentVerdict(t, report, "You are thirsty.", "yes")

	// A literal whose fixed text is all below the floor is unverifiable: not
	// covered, and not counted against the percentage either.
	assertSiteVerdict(t, report, "src/sample.c", 6, "unverifiable")

	compareGolden(t, report)
}

func goFixtureMentions(t *testing.T, text string) bool {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "go", "pkg", "game", "sample.go"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Contains(string(body), text)
}

func compareGolden(t *testing.T, report *Report) {
	t.Helper()
	goldenDir := filepath.Join("testdata", "golden")
	for name, got := range report.Render() {
		path := filepath.Join(goldenDir, name)
		if *updateGolden {
			if err := os.MkdirAll(goldenDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read golden %s: %v (run go test ./internal/censuscoverage/ -update)", name, err)
		}
		if string(want) != got {
			t.Errorf("%s does not match the golden file:\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
		}
	}
}
