// Command dp-string-census runs the two-directional player-facing string census
// described in docs/fidelity/strings/README.md.
//
//	go run ./cmd/dp-string-census            # report only, writes nothing
//	go run ./cmd/dp-string-census --update   # rewrite the reports and baseline
//	go run ./cmd/dp-string-census --check    # ratchet: fail on a new go-only string
//
// It never edits game code. The C tree under src/ and the Go tree are both
// read-only inputs (RULEBOOK R5).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/internal/stringcensus"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "dp-string-census: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root := flag.String("root", ".", "repository root")
	goDirs := flag.String("go-dirs", "", "comma-separated Go dirs (default "+strings.Join(stringcensus.DefaultGoDirs, ",")+")")
	cDir := flag.String("c-dir", "", "C oracle dir (default "+stringcensus.DefaultCDir+")")
	dataDirs := flag.String("data-dirs", "", "comma-separated world-data dirs (default "+strings.Join(stringcensus.DefaultDataDirs, ",")+")")
	outDir := flag.String("out-dir", "", "output dir relative to root (default "+stringcensus.DefaultOutDir+")")
	check := flag.Bool("check", false, "ratchet mode: fail when a go-only segment is not in the baseline")
	update := flag.Bool("update", false, "rewrite the generated reports and the baseline")
	top := flag.Int("top", 30, "how many go-only groups by file to print")
	flag.Parse()

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	relOut := *outDir
	if relOut == "" {
		relOut = stringcensus.DefaultOutDir
	}
	opts := stringcensus.Options{
		Root:     absRoot,
		GoDirs:   splitList(*goDirs),
		CDir:     *cDir,
		DataDirs: splitList(*dataDirs),
		OutDir:   relOut,
	}
	report, err := stringcensus.Run(opts)
	if err != nil {
		return err
	}
	absOut := filepath.Join(absRoot, filepath.FromSlash(relOut))
	if filepath.IsAbs(relOut) {
		absOut = filepath.Clean(relOut)
	}

	printReport(report, *top)

	switch {
	case *update:
		return updateAll(report, absOut)
	case *check:
		return checkRatchet(report, absOut)
	default:
		fmt.Println("report only: pass --update to write the reports or --check to run the ratchet")
		return nil
	}
}

// printReport writes the human summary: totals plus the largest go-only groups
// by file, which is what a review reads first.
func printReport(r *stringcensus.Report, top int) {
	fmt.Printf("string census: %s\n", r)
	fmt.Printf("scanned %d go files, %d c files, %d data files\n", r.GoFiles, r.CFiles, r.DataFiles)
	fmt.Printf("candidates: %d go (literal at a sink), %d c (literal in a sink call)\n", r.GoCandidates, r.CCandidates)
	groups := r.GoOnlyByFile()
	if len(groups) == 0 {
		return
	}
	fmt.Printf("largest go-only groups by file (of %d files):\n", len(groups))
	for i, g := range groups {
		if i >= top {
			break
		}
		fmt.Printf("  %6d  %s\n", g.Count, g.File)
	}
}

func updateAll(r *stringcensus.Report, absOut string) error {
	if err := r.WriteOutputs(absOut); err != nil {
		return err
	}
	baselinePath := filepath.Join(absOut, stringcensus.BaselineFile)
	base, err := stringcensus.LoadBaseline(baselinePath)
	if err != nil {
		return err
	}
	merged := base.Merge(r, today())
	if err := merged.Save(baselinePath); err != nil {
		return err
	}
	fmt.Printf("wrote %s and %s (%d go-only entries, %d unreviewed)\n",
		absOut, stringcensus.BaselineFile, len(merged.Segments), merged.UnreviewedCount())
	return nil
}

// checkRatchet is the CI gate. A new go-only segment fails: it is an R4
// candidate that no scenario had to trigger. Stale entries and stale reports
// are warnings, because fixing a string should not require regenerating the
// census to go green.
func checkRatchet(r *stringcensus.Report, absOut string) error {
	baselinePath := filepath.Join(absOut, stringcensus.BaselineFile)
	base, err := stringcensus.LoadBaseline(baselinePath)
	if err != nil {
		return err
	}
	added, stale := base.Check(r)
	if len(stale) > 0 {
		fmt.Printf("note: %d baseline entries are no longer produced; run make string-census-update to prune them\n", len(stale))
	}
	staleFiles, err := r.StaleOutputs(absOut)
	if err != nil {
		return err
	}
	if len(staleFiles) > 0 {
		sort.Strings(staleFiles)
		fmt.Printf("warning: generated reports are stale (%s); run make string-census-update\n", strings.Join(staleFiles, ", "))
	}
	if len(added) == 0 {
		fmt.Printf("string census ratchet: ok (%d known go-only segments, %d unreviewed)\n",
			len(base.Segments), base.UnreviewedCount())
		return nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "string census ratchet: %d new go-only segment(s) with no C or world-data source\n", len(added))
	for i, text := range added {
		if i == 20 {
			fmt.Fprintf(&b, "  ... and %d more (see %s)\n", len(added)-20, stringcensus.GoOnlyFile)
			break
		}
		fmt.Fprintf(&b, "  %q\n", text)
	}
	b.WriteString("No scenario made C print these, so the census could not vouch for them (R2/R4).\n")
	b.WriteString("Check the C source first; if the string is legitimate, add it to " + stringcensus.BaselineFile + " with a reason.\n")
	return fmt.Errorf("%s", b.String())
}

func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func today() string {
	return time.Now().Format("2006-01-02")
}
