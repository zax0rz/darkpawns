package censuscoverage

import (
	"sort"

	"github.com/zax0rz/darkpawns/internal/stringcensus"
)

// SiteCoverage is one C site's verdict: the literal, the scenarios whose dumped
// C output printed it, and the probe labels that did.
type SiteCoverage struct {
	Site stringcensus.CSite
	// Scenarios are the scenario names that covered the site, sorted.
	Scenarios []string
	// Commands are the probe labels that covered the site, sorted and unique.
	Commands []string
}

// Covered reports whether any scenario printed the site's fixed text.
func (s SiteCoverage) Covered() bool { return len(s.Scenarios) > 0 }

// Report is the coverage of one C tree by one dump.
type Report struct {
	// Sites is every C site in source order, with its verdict.
	Sites []SiteCoverage
	// ScenarioCount and BlockCount describe the dump the verdicts come from, so
	// a partial dump is visible in the numbers themselves.
	ScenarioCount int
	BlockCount    int
	// EmptyScenarios are dump files with no C blocks at all.
	EmptyScenarios []string
}

// Compute matches every site against every block of every scenario.
func Compute(sites []stringcensus.CSite, scenarios []Scenario) *Report {
	report := &Report{ScenarioCount: len(scenarios)}
	covered := make([]map[string]bool, len(sites))
	commands := make([]map[string]bool, len(sites))
	for i := range sites {
		covered[i] = map[string]bool{}
		commands[i] = map[string]bool{}
	}

	matcher := stringcensus.NewSiteMatcher(sites)
	for _, scenario := range scenarios {
		report.BlockCount += len(scenario.Blocks)
		if len(scenario.Blocks) == 0 {
			report.EmptyScenarios = append(report.EmptyScenarios, scenario.Name)
			continue
		}
		for _, block := range scenario.Blocks {
			label := block.Label
			matcher.ScanBlock(block.Text, func(site, _ int) {
				covered[site][scenario.Name] = true
				commands[site][label] = true
			})
		}
	}

	report.Sites = make([]SiteCoverage, len(sites))
	for i, site := range sites {
		report.Sites[i] = SiteCoverage{
			Site:      site,
			Scenarios: sortedKeys(covered[i]),
			Commands:  sortedKeys(commands[i]),
		}
	}
	return report
}

// Totals summarises the run. Segments are the census's unit (brief 07's
// segments); sites are the C literals a player actually reads. Unverifiable
// sites are counted apart: their fixed text is too short to compare, so no dump
// can be evidence either way and they are excluded from the percentage rather
// than counted as misses.
type Totals struct {
	Sites             int
	SitesCovered      int
	UnverifiableSites int
	Segments          int
	SegmentsCovered   int
}

// Totals rolls the report up.
func (r *Report) Totals() Totals {
	var t Totals
	for i := range r.Sites {
		sc := &r.Sites[i]
		t.Sites++
		if sc.Covered() {
			t.SitesCovered++
		}
		if sc.Site.Unverifiable() {
			t.UnverifiableSites++
		}
		t.Segments += len(sc.Site.Segments)
		if sc.Covered() {
			t.SegmentsCovered += len(sc.Site.Segments)
		}
	}
	return t
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
