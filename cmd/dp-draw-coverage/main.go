// Command dp-draw-coverage extracts every C RNG draw-primitive call site from
// the compiled reference binary, joins it with parity-checked draw attributions
// from a census run, and writes draw-inventory.tsv, draw-coverage.tsv, and
// draw-coverage-summary.txt.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zax0rz/darkpawns/internal/drawcoverage"
)

const (
	defaultOracleBin = "/home/zach/darkpawns-c-oracle/bin/circle"
	defaultOutDir    = "docs/fidelity/draws"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "dp-draw-coverage: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root := flag.String("root", ".", "repository root")
	oracleBin := flag.String("oracle-bin", defaultOracleBin, "compiled C reference oracle binary")
	runDir := flag.String("run", "", "attributed census run directory (required)")
	refRunDir := flag.String("ref-run", "", "reference census run directory for byte-for-byte c-draws.log parity check")
	outDir := flag.String("out-dir", defaultOutDir, "output directory (relative to --root unless absolute)")
	top := flag.Int("top", 15, "how many least-reached C files to print")
	flag.Parse()

	if *runDir == "" {
		return fmt.Errorf("--run <attributed-census-dir> is required")
	}

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	absOut := *outDir
	if !filepath.IsAbs(absOut) {
		absOut = filepath.Join(absRoot, filepath.FromSlash(*outDir))
	}

	inv, err := drawcoverage.ExtractInventory(*oracleBin)
	if err != nil {
		return err
	}
	captures, exclusions, err := drawcoverage.LoadCensusRun(*runDir, *refRunDir, inv.TopLevel)
	if err != nil {
		return err
	}
	report := drawcoverage.Compute(inv.TopLevel, inv.Internal, captures, exclusions)
	printSummary(report, *top)
	if err := report.WriteOutputs(absOut); err != nil {
		return err
	}
	fmt.Printf("wrote %s, %s, and %s to %s\n",
		drawcoverage.InventoryFile, drawcoverage.CoverageFile, drawcoverage.SummaryFile, absOut)
	return nil
}

func printSummary(r *drawcoverage.Report, top int) {
	t := r.Totals()
	fmt.Printf("draw coverage: %d pairs included (%d seed-1), %d excluded\n",
		r.PairsIncluded, r.Seed1Included, len(r.Exclusions))
	fmt.Printf("c draw sites (seed 1):    %d of %d reached (%.1f%%)\n",
		t.ReachedSeed1, t.Sites, pct(t.ReachedSeed1, t.Sites))
	fmt.Printf("c draw sites (all seeds): %d of %d reached (%.1f%%) (+%d multi-seed)\n",
		t.ReachedAll, t.Sites, pct(t.ReachedAll, t.Sites), t.ReachedAll-t.ReachedSeed1)
	fmt.Printf("top-level draws:          %d all seeds (%d seed 1)\n", t.TotalDraws, t.Seed1Draws)

	rolls := r.LeastReachedFiles()
	fmt.Printf("least reached C files (of %d):\n", len(rolls))
	for i, roll := range rolls {
		if i >= top {
			break
		}
		fmt.Printf("  %6.1f%%  %3d/%-3d sites  %s\n",
			pct(roll.ReachedAll, roll.Sites), roll.ReachedAll, roll.Sites, roll.File)
	}
}

func pct(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) * 100 / float64(total)
}
