// Command dp-census-coverage maps how much of the C game's player-facing text
// the census scenarios actually reached.
//
// It reads a census dump directory written by
//
//	dp-oracle-diff --dump-oracle <dir>      (ORACLE_REGRESSION_DUMP=<dir> make oracle-regression)
//
// and writes docs/fidelity/strings/coverage.tsv plus a summary. It never starts
// a census of its own: coverage is a claim about the scenarios that ran, so the
// dump has to come from a real run.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/internal/censuscoverage"
	"github.com/zax0rz/darkpawns/internal/stringcensus"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "dp-census-coverage: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root := flag.String("root", ".", "repository root")
	cDir := flag.String("c-dir", "", "C oracle dir (default "+stringcensus.DefaultCDir+")")
	dump := flag.String("dump", "", "census dump directory (default $ORACLE_REGRESSION_DUMP)")
	outDir := flag.String("out-dir", "", "output dir relative to root (default "+stringcensus.DefaultOutDir+")")
	top := flag.Int("top", 15, "how many C files to list, least covered first")
	flag.Parse()

	dumpDir := *dump
	if dumpDir == "" {
		dumpDir = os.Getenv("ORACLE_REGRESSION_DUMP")
	}
	if dumpDir == "" {
		return fmt.Errorf("no dump directory: pass --dump <dir> or set ORACLE_REGRESSION_DUMP, " +
			"e.g. ORACLE_REGRESSION_DUMP=/tmp/dp-dump make oracle-regression")
	}
	// Read the dump through an os.Root: every read stays inside the directory,
	// and a bad path fails here instead of reporting 0% coverage.
	dumpRoot, err := censuscoverage.OpenDump(dumpDir)
	if err != nil {
		return err
	}
	defer func() { _ = dumpRoot.Close() }()

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	relOut := *outDir
	if relOut == "" {
		relOut = stringcensus.DefaultOutDir
	}
	absOut := filepath.Join(absRoot, filepath.FromSlash(relOut))
	if filepath.IsAbs(relOut) {
		absOut = filepath.Clean(relOut)
	}

	sites, err := stringcensus.ExtractCSource(stringcensus.Options{Root: absRoot, CDir: *cDir})
	if err != nil {
		return err
	}
	scenarios, err := censuscoverage.LoadDir(dumpRoot)
	if err != nil {
		return err
	}
	report := censuscoverage.Compute(sites, scenarios)
	printReport(report, *top)
	if err := report.WriteOutputs(absOut); err != nil {
		return err
	}
	fmt.Printf("wrote %s and %s\n", censuscoverage.CoverageFile, censuscoverage.SummaryFile)
	return nil
}

// printReport prints the headline numbers and the least-covered C files: the
// actionable view is what the scenarios never reached, not the total.
func printReport(r *censuscoverage.Report, top int) {
	t := r.Totals()
	fmt.Printf("census coverage: %d scenarios, %d blocks in the dump\n", r.ScenarioCount, r.BlockCount)
	fmt.Printf("c sites:    %d of %d covered (%.1f%%)\n", t.SitesCovered, t.Sites, percentOf(t.SitesCovered, t.Sites))
	fmt.Printf("c segments: %d of %d covered (%.1f%%), %d unverifiable sites\n",
		t.SegmentsCovered, t.Segments, percentOf(t.SegmentsCovered, t.Segments), t.UnverifiableSites)
	if len(r.EmptyScenarios) > 0 {
		fmt.Printf("note: %d dumped scenarios have no C blocks: %s\n",
			len(r.EmptyScenarios), strings.Join(r.EmptyScenarios, ", "))
	}

	rolls := r.ByFile()
	sort.Slice(rolls, func(i, j int) bool {
		pi, pj := percentOf(rolls[i].SegmentsCovered, rolls[i].Segments), percentOf(rolls[j].SegmentsCovered, rolls[j].Segments)
		if pi != pj {
			return pi < pj
		}
		if rolls[i].Segments != rolls[j].Segments {
			return rolls[i].Segments > rolls[j].Segments
		}
		return rolls[i].Key < rolls[j].Key
	})
	if len(rolls) == 0 {
		return
	}
	fmt.Printf("least covered C files (of %d):\n", len(rolls))
	for i, roll := range rolls {
		if i >= top {
			break
		}
		fmt.Printf("  %6.1f%%  %4d/%-4d segments  %s\n",
			percentOf(roll.SegmentsCovered, roll.Segments), roll.SegmentsCovered, roll.Segments, roll.Key)
	}
}

// percentOf renders a share as a percentage; an empty denominator is 0.
func percentOf(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) * 100 / float64(total)
}
