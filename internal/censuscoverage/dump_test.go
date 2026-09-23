package censuscoverage

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDumpReadsBlocks(t *testing.T) {
	body := "normalized C oracle blocks:\n" +
		"--- [look]\n" +
		"The temple is quiet.\n" +
		"Frodo has left the game.\n" +
		"\n" +
		"--- [quit]\n" +
		"Goodbye, friend.. Come back soon!\n"
	scenario := ParseDump("quit-probe", body)
	if scenario.Name != "quit-probe" {
		t.Fatalf("name = %q", scenario.Name)
	}
	if len(scenario.Blocks) != 2 {
		t.Fatalf("blocks = %+v, want 2", scenario.Blocks)
	}
	if scenario.Blocks[0].Label != "look" || scenario.Blocks[1].Label != "quit" {
		t.Fatalf("labels = %q, %q", scenario.Blocks[0].Label, scenario.Blocks[1].Label)
	}
	if !strings.Contains(scenario.Blocks[0].Text, "Frodo has left the game.") {
		t.Fatalf("first block text = %q", scenario.Blocks[0].Text)
	}
	if strings.Contains(scenario.Blocks[0].Text, "Goodbye") {
		t.Fatalf("block boundary leaked into the first block: %q", scenario.Blocks[0].Text)
	}
}

func TestParseDumpKeepsBracketsInLabels(t *testing.T) {
	scenario := ParseDump("relogin", "--- [look [audience]]\nThe temple is quiet.\n")
	if len(scenario.Blocks) != 1 || scenario.Blocks[0].Label != "look [audience]" {
		t.Fatalf("blocks = %+v", scenario.Blocks)
	}
}

func TestParseDumpIgnoresPreamble(t *testing.T) {
	scenario := ParseDump("preamble", "Dark Pawns Tier-1 differential report\nresult: no normalized divergence\nlook\n")
	if len(scenario.Blocks) != 0 {
		t.Fatalf("blocks = %+v, want none without a block header", scenario.Blocks)
	}
}

func TestLoadDirFindsEveryScenario(t *testing.T) {
	root, err := OpenDump(filepath.Join("testdata", "dump"))
	if err != nil {
		t.Fatalf("OpenDump: %v", err)
	}
	t.Cleanup(func() { _ = root.Close() })
	scenarios, err := LoadDir(root)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	names := make([]string, 0, len(scenarios))
	for _, s := range scenarios {
		names = append(names, s.Name)
	}
	if len(names) != 2 || names[0] != "empty-probe" || names[1] != "quit-probe" {
		t.Fatalf("scenarios = %v, want empty-probe then quit-probe", names)
	}
}

func TestOpenDumpReportsAMissingDirectory(t *testing.T) {
	if _, err := OpenDump(filepath.Join("testdata", "no-such-dump")); err == nil {
		t.Fatal("a missing dump directory must be an error, not an empty green report")
	}
}

// TestEmptyScenarioIsSurfaced covers the "dumped but produced no C blocks" case:
// it must show up in the report rather than look like full coverage of nothing.
func TestEmptyScenarioIsSurfaced(t *testing.T) {
	report := fixtureReport(t)
	if len(report.EmptyScenarios) != 1 || report.EmptyScenarios[0] != "empty-probe" {
		t.Fatalf("empty scenarios = %v, want [empty-probe]", report.EmptyScenarios)
	}
	if report.ScenarioCount != 2 || report.BlockCount != 2 {
		t.Fatalf("scenario/block counts = %d/%d, want 2/2", report.ScenarioCount, report.BlockCount)
	}
}
