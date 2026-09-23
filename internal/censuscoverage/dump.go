// Package censuscoverage turns a census dump into a coverage map: which C
// player-facing strings the scenarios that ran actually made C print.
//
// The dump comes from `cmd/dp-oracle-diff --dump-oracle <dir>`, one
// `<scenario>.txt` per scenario, holding exactly what `--show-oracle` prints:
// the normalized C blocks of that run. Coverage is therefore measured against
// the same normalization the differential judge used, which is the point — a
// green census only says the scenarios that ran matched, and this says which
// parts of the game they reached.
package censuscoverage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Block is one normalized C output burst with the probe label that produced it.
type Block struct {
	Label string
	Text  string
}

// Scenario is one dumped scenario: its normalized C blocks in probe order.
type Scenario struct {
	// Name is the scenario file name without its .txt extension.
	Name string
	// Blocks are the C output bursts, in order.
	Blocks []Block
}

// blockHeaderPrefix begins a block label line ("--- [look]"). It is the format
// `dp-oracle-diff --show-oracle` prints, so a dump is a verbatim copy of that
// output and can be read by a human exactly as the harness printed it.
const blockHeaderPrefix = "--- ["

// ParseDump parses one dump file body into a Scenario. Everything before the
// first block header is harness preamble and is dropped. A block runs until the
// next header; its text keeps its line structure, because a C literal's fixed
// text may span several printed lines.
func ParseDump(name, body string) Scenario {
	scenario := Scenario{Name: name}
	var (
		label   string
		lines   []string
		started bool
	)
	flush := func() {
		if !started {
			return
		}
		scenario.Blocks = append(scenario.Blocks, Block{
			Label: label,
			Text:  strings.Join(lines, "\n"),
		})
		lines = nil
	}
	for _, line := range strings.Split(body, "\n") {
		if newLabel, ok := parseBlockHeader(line); ok {
			flush()
			label = newLabel
			started = true
			continue
		}
		if started {
			lines = append(lines, line)
		}
	}
	flush()
	return scenario
}

// parseBlockHeader reads a "--- [label]" line. Labels may themselves contain
// brackets (multi-client blocks are labelled "look [oracle-name]"), so the
// label is everything between the first "[" and the last "]".
func parseBlockHeader(line string) (string, bool) {
	line = strings.TrimRight(line, "\r")
	if !strings.HasPrefix(line, blockHeaderPrefix) || !strings.HasSuffix(line, "]") {
		return "", false
	}
	label := strings.TrimSuffix(strings.TrimPrefix(line, blockHeaderPrefix), "]")
	return label, true
}

// OpenDump opens a dump directory as an os.Root. Every read through the Root is
// confined to that directory, and opening it proves the directory exists, so the
// tool errors out instead of reporting 0% coverage from a typo in the path.
func OpenDump(dir string) (*os.Root, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open dump directory %s (run a census with "+
			"ORACLE_REGRESSION_DUMP=%s first): %w", dir, dir, err)
	}
	return root, nil
}

// LoadDir reads every `<name>.txt` under an opened dump root. The directory is
// the artifact of a census run with ORACLE_REGRESSION_DUMP set; the tool never
// starts a census of its own.
func LoadDir(root *os.Root) ([]Scenario, error) {
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return nil, fmt.Errorf("read dump directory %s: %w", root.Name(), err)
	}
	var scenarios []Scenario
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".txt" {
			continue
		}
		body, err := root.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read dump file %s: %w", entry.Name(), err)
		}
		name := strings.TrimSuffix(entry.Name(), ".txt")
		scenarios = append(scenarios, ParseDump(name, string(body)))
	}
	sort.Slice(scenarios, func(i, j int) bool { return scenarios[i].Name < scenarios[j].Name })
	return scenarios, nil
}
