package stringcensus

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// committedBaselinePath is the ratchet baseline as checked in.
var committedBaselinePath = filepath.Join("..", "..", "docs", "fidelity", "strings", BaselineFile)

// TestCommittedBaselineUsesTheVocabulary guards the triage contract: every row
// carries a reason from the vocabulary and, unless it is still unreviewed, an
// evidence string a reviewer can check. A typo in a reason, or a reviewed row
// with no evidence, fails here instead of silently passing review.

// TestCommittedTriageReportCountsMatchTheBaseline keeps docs/fidelity/strings/
// triage.md honest: the counts in its header must match the baseline it
// describes.
func TestCommittedTriageReportCountsMatchTheBaseline(t *testing.T) {
	base, err := LoadBaseline(committedBaselinePath)
	if err != nil {
		t.Fatalf("load committed baseline: %v", err)
	}
	counts := map[string]int{}
	for _, entry := range base.Segments {
		counts[entry.Reason]++
	}
	body, err := os.ReadFile(filepath.Join("..", "..", "docs", "fidelity", "strings", "triage.md"))
	if err != nil {
		t.Fatalf("read triage report: %v", err)
	}
	report := string(body)
	for reason, n := range counts {
		want := "| `" + reason + "` | " + strconv.Itoa(n) + " |"
		if !strings.Contains(report, want) {
			t.Errorf("triage.md is missing the row %q (baseline has %d %s)", want, n, reason)
		}
	}
}
