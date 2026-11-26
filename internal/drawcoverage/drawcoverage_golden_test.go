package drawcoverage

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateGolden = flag.Bool("update", false, "rewrite drawcoverage golden files")

func fixtureInventory() ([]DrawSite, []DrawSite) {
	top := []DrawSite{
		{Site: "src/db.c:2081", File: "src/db.c", Line: 2081, Function: "percent_load", Primitive: "uniform", CallVaddr: 0x26945, RetVaddr: 0x2694a},
		{Site: "src/fight.c:100", File: "src/fight.c", Line: 100, Function: "hit", Primitive: "number", CallVaddr: 0x41000, RetVaddr: 0x41005},
		{Site: "src/magic.c:55", File: "src/magic.c", Line: 55, Function: "mag_damage", Primitive: "dice", CallVaddr: 0x52000, RetVaddr: 0x52005},
		{Site: "src/spec_procs.c:42", File: "src/spec_procs.c", Line: 42, Function: "mayor", Primitive: "number", CallVaddr: 0x63000, RetVaddr: 0x63005},
		{Site: "src/weather.c:137#1", File: "src/weather.c", Line: 137, Function: "weather_change", Primitive: "dice", CallVaddr: 0xad2c5, RetVaddr: 0xad2ca},
		{Site: "src/weather.c:137#2", File: "src/weather.c", Line: 137, Function: "weather_change", Primitive: "dice", CallVaddr: 0xad2da, RetVaddr: 0xad2df},
	}
	internal := []DrawSite{
		{Site: "src/utils.c:90", File: "src/utils.c", Line: 90, Function: "number", Primitive: "uniform", CallVaddr: 0xaa0ab, RetVaddr: 0xaa0b0},
		{Site: "src/utils.c:109", File: "src/utils.c", Line: 109, Function: "dice", Primitive: "number", CallVaddr: 0xaa18f, RetVaddr: 0xaa194},
		{Site: "src/utils.c:121", File: "src/utils.c", Line: 121, Function: "uniform", Primitive: "prng_uniform", CallVaddr: 0xaa22f, RetVaddr: 0xaa234},
	}
	return top, internal
}

func fixtureCaptures() ([]Capture, []Exclusion) {
	captures := []Capture{
		{
			Seed:     1,
			Scenario: "combat-hit",
			Calls: []AttributedCall{
				{Site: "src/db.c:2081", Primitive: "uniform"},
				{Site: "src/fight.c:100", Primitive: "number"},
				{Site: "src/fight.c:100", Primitive: "number"},
			},
		},
		{
			Seed:     1,
			Scenario: "weather-tick",
			Calls: []AttributedCall{
				{Site: "src/db.c:2081", Primitive: "uniform"},
				{Site: "src/weather.c:137#1", Primitive: "dice"},
				{Site: "src/weather.c:137#2", Primitive: "dice"},
			},
		},
		{
			Seed:     2,
			Scenario: "magic-cast",
			Calls: []AttributedCall{
				{Site: "src/db.c:2081", Primitive: "uniform"},
				{Site: "src/magic.c:55", Primitive: "dice"},
			},
		},
	}
	exclusions := []Exclusion{
		{Seed: 1, Scenario: "timing-drift", Reason: "draw_stream_mismatch"},
	}
	return captures, exclusions
}

func TestGoldenDrawCoverage(t *testing.T) {
	top, internal := fixtureInventory()
	captures, exclusions := fixtureCaptures()
	report := Compute(top, internal, captures, exclusions)

	totals := report.Totals()
	if totals.Sites != 6 {
		t.Fatalf("totals.Sites = %d, want 6", totals.Sites)
	}
	if totals.ReachedSeed1 != 4 {
		t.Errorf("totals.ReachedSeed1 = %d, want 4", totals.ReachedSeed1)
	}
	if totals.ReachedAll != 5 {
		t.Errorf("totals.ReachedAll = %d, want 5", totals.ReachedAll)
	}
	if totals.TotalDraws != 8 || totals.Seed1Draws != 6 {
		t.Errorf("draws = (%d all, %d seed1), want (8, 6)", totals.TotalDraws, totals.Seed1Draws)
	}

	assertSiteReached := func(site string, wantAll, wantSeed1 bool, wantCount int64) {
		t.Helper()
		for i := range report.Sites {
			sc := &report.Sites[i]
			if sc.Site.Site != site {
				continue
			}
			if sc.Reached() != wantAll || sc.ReachedSeed1() != wantSeed1 || sc.DrawCount != wantCount {
				t.Errorf("site %s = (reached=%v, seed1=%v, count=%d), want (%v, %v, %d)",
					site, sc.Reached(), sc.ReachedSeed1(), sc.DrawCount, wantAll, wantSeed1, wantCount)
			}
			return
		}
		t.Errorf("site %s not found in report", site)
	}

	assertSiteReached("src/db.c:2081", true, true, 3)
	assertSiteReached("src/fight.c:100", true, true, 2)
	assertSiteReached("src/magic.c:55", true, false, 1)
	assertSiteReached("src/spec_procs.c:42", false, false, 0)
	assertSiteReached("src/weather.c:137#1", true, true, 1)
	assertSiteReached("src/weather.c:137#2", true, true, 1)

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
			t.Fatalf("read golden %s: %v (run go test ./internal/drawcoverage/ -update)", name, err)
		}
		if string(want) != got {
			t.Errorf("%s mismatch:\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
		}
	}
}

func TestLoadCensusRunParityAndExclusions(t *testing.T) {
	top, _ := fixtureInventory()
	tmp := t.TempDir()
	attrDir := filepath.Join(tmp, "attr")
	refDir := filepath.Join(tmp, "ref")

	writeCap := func(base string, seed string, scenario string, draws, sites string) {
		t.Helper()
		dir := filepath.Join(base, seed, "initial-logs", "attempt.combined", scenario+".capture1")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "c-draws.log"), []byte(draws), 0o644); err != nil {
			t.Fatal(err)
		}
		if sites != "" {
			if err := os.WriteFile(filepath.Join(dir, "c-draw-sites.log"), []byte(sites), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	goodDraws := "DRAW 0 1 Uniform(0,0) = 500000\nDRAW 1 1 Number(1,10) = 5\n"
	goodSites := "0x2694a\tuniform\n0x41005\tnumber\n"

	// 1. Exact parity -> included.
	writeCap(attrDir, "seed1", "clean-scenario", goodDraws, goodSites)
	writeCap(refDir, "seed1", "clean-scenario", goodDraws, "")

	// 2. Reference c-draws.log differs -> excluded as draw_stream_mismatch.
	writeCap(attrDir, "seed1", "drift-scenario", goodDraws, goodSites)
	writeCap(refDir, "seed1", "drift-scenario", "DRAW 0 1 Uniform(0,0) = 500000\nDRAW 1 1 Number(1,10) = 6\n", "")

	// 3. Primitive mismatch between c-draws.log and c-draw-sites.log -> excluded.
	writeCap(attrDir, "seed1", "prim-mismatch", goodDraws, "0x2694a\tuniform\n0x52005\tdice\n")
	writeCap(refDir, "seed1", "prim-mismatch", goodDraws, "")

	// 4. Line count mismatch -> excluded.
	writeCap(attrDir, "seed1", "len-mismatch", goodDraws, "0x2694a\tuniform\n")
	writeCap(refDir, "seed1", "len-mismatch", goodDraws, "")

	captures, exclusions, err := LoadCensusRun(attrDir, refDir, top)
	if err != nil {
		t.Fatalf("LoadCensusRun: %v", err)
	}
	if len(captures) != 1 || captures[0].Scenario != "clean-scenario" || len(captures[0].Calls) != 2 {
		t.Fatalf("captures = %+v, want 1 clean-scenario with 2 calls", captures)
	}
	wantReasons := map[string]string{
		"drift-scenario": "draw_stream_mismatch",
		"prim-mismatch":  "primitive_mismatch",
		"len-mismatch":   "line_count_mismatch",
	}
	if len(exclusions) != len(wantReasons) {
		t.Fatalf("exclusions = %+v, want %d", exclusions, len(wantReasons))
	}
	for _, ex := range exclusions {
		if wantReasons[ex.Scenario] != ex.Reason {
			t.Errorf("exclusion %s reason = %q, want %q", ex.Scenario, ex.Reason, wantReasons[ex.Scenario])
		}
	}
}

func TestExtractInventoryAgainstReferenceBinary(t *testing.T) {
	const bin = "/home/zach/darkpawns-c-oracle/bin/circle"
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("reference oracle binary not present at %s", bin)
	}
	inv, err := ExtractInventory(bin)
	if err != nil {
		t.Fatalf("ExtractInventory: %v", err)
	}
	if len(inv.TopLevel) != 361 || len(inv.Internal) != 3 {
		t.Fatalf("got %d top-level + %d internal, want 361 + 3", len(inv.TopLevel), len(inv.Internal))
	}
	found := map[string]DrawSite{}
	for _, s := range inv.TopLevel {
		found[s.Site] = s
	}
	for _, want := range []struct {
		site, fn, prim string
	}{
		{"src/db.c:2081", "percent_load", "uniform"},
		{"src/graph.c:355", "hunt_victim", "number"},
		{"src/weather.c:137#1", "weather_change", "dice"},
		{"src/weather.c:137#2", "weather_change", "dice"},
		{"src/weather.c:137#3", "weather_change", "dice"},
	} {
		got, ok := found[want.site]
		if !ok {
			t.Errorf("missing expected site %s", want.site)
			continue
		}
		if got.Function != want.fn || got.Primitive != want.prim {
			t.Errorf("site %s = (%s, %s), want (%s, %s)", want.site, got.Function, got.Primitive, want.fn, want.prim)
		}
	}
}
