package censuscoverage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/internal/stringcensus"
)

// Render returns the generated reports keyed by file name.
func (r *Report) Render() map[string]string {
	return map[string]string{
		CoverageFile: r.coverageTSV(),
		SummaryFile:  r.summary(),
	}
}

// WriteOutputs writes the coverage reports into outDir (an absolute path).
func (r *Report) WriteOutputs(outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for name, body := range r.Render() {
		if err := os.WriteFile(filepath.Join(outDir, name), []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// coverageTSV is the per-segment detail: every C segment with the verdict of the
// literal it belongs to and the scenarios that printed it.
func (r *Report) coverageTSV() string {
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("# Coverage of the C tree by one census dump (dp-oracle-diff --dump-oracle).\n")
	b.WriteString("# columns: segment\tfile:line\tsink\tfunction\tcovered\tscenarios\tcommands\n")
	b.WriteString("# covered is yes|no|unverifiable; unverifiable means the segment's fixed text is\n")
	b.WriteString("# shorter than the census floor, so no dump can prove or disprove it printed.\n")
	for i := range r.Sites {
		sc := &r.Sites[i]
		verdict := "no"
		switch {
		case sc.Site.Unverifiable():
			verdict = "unverifiable"
		case sc.Covered():
			verdict = "yes"
		}
		for _, seg := range siteSegments(sc) {
			fmt.Fprintf(&b, "%s\t%s:%d\t%s\t%s\t%s\t%s\t%s\n",
				sanitize(seg), sc.Site.File, sc.Site.Line, sc.Site.Sink, emptyAsNone(sc.Site.Fn),
				verdict, emptyAsNone(strings.Join(sc.Scenarios, ",")), emptyAsNone(strings.Join(sc.Commands, ",")))
		}
	}
	return b.String()
}

// siteSegments is the site's segments, or a single placeholder for an
// unverifiable literal so every C literal appears in the report at least once.
func siteSegments(sc *SiteCoverage) []string {
	if sc.Site.Unverifiable() {
		return []string{"(no fixed text at or above the floor)"}
	}
	return sc.Site.Segments
}

func (r *Report) summary() string {
	t := r.Totals()
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("# Deterministic: no dates, no paths outside the repository.\n")
	fmt.Fprintf(&b, "dump scenarios\t%d\n", r.ScenarioCount)
	fmt.Fprintf(&b, "dump blocks\t%d\n", r.BlockCount)
	fmt.Fprintf(&b, "dump scenarios with no c blocks\t%d\t%s\n", len(r.EmptyScenarios), strings.Join(r.EmptyScenarios, ","))
	fmt.Fprintf(&b, "c sites\t%d of %d covered (%s)\n", t.SitesCovered, t.Sites, percent(t.SitesCovered, t.Sites))
	fmt.Fprintf(&b, "c segments\t%d of %d covered (%s)\n", t.SegmentsCovered, t.Segments, percent(t.SegmentsCovered, t.Segments))
	fmt.Fprintf(&b, "unverifiable sites\t%d (fixed text shorter than the %d-character floor; excluded from the percentage)\n", t.UnverifiableSites, stringcensus.MinSegmentLen)

	b.WriteString("\n[per c file]\n")
	b.WriteString("# columns: file\tsites covered/total\tsegments covered/total (percent)\tunverifiable sites\n")
	for _, roll := range r.ByFile() {
		writeRollup(&b, roll.Key, roll)
	}
	b.WriteString("\n[per c function]\n")
	b.WriteString("# columns: file:function\tsites covered/total\tsegments covered/total (percent)\tunverifiable sites\n")
	for _, roll := range r.ByFunction() {
		writeRollup(&b, roll.Key, roll)
	}
	b.WriteString("\n[never seen segments]\n")
	b.WriteString("# columns: file:line\tsink\tfunction\tsegment\n")
	for _, roll := range r.ByFile() {
		for _, miss := range roll.NeverSeen {
			fmt.Fprintf(&b, "%s:%d\t%s\t%s\t%s\n", miss.File, miss.Line, miss.Sink, emptyAsNone(miss.Fn), sanitize(miss.Text))
		}
	}
	return b.String()
}

func writeRollup(b *strings.Builder, key string, roll Rollup) {
	fmt.Fprintf(b, "%s\t%d/%d\t%d/%d (%s)\t%d\n",
		key, roll.SitesCovered, roll.Sites, roll.SegmentsCovered, roll.Segments,
		percent(roll.SegmentsCovered, roll.Segments), roll.Unverifiable)
}

// percent renders a share, or "n/a" when there is nothing to divide.
func percent(part, total int) string {
	if total == 0 {
		return "n/a"
	}
	return strconv.FormatFloat(float64(part)*100/float64(total), 'f', 1, 64) + "%"
}

func emptyAsNone(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// sanitize keeps a segment on one TSV line.
func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	return strings.ReplaceAll(s, "\n", " ")
}
