package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

func TestStripProbeSectionAndRenderLockstepScenario(t *testing.T) {
	raw := `# comment
[fixture]
quiet-mobs

[setup:oracle]
Test
1

[setup:port]
Test
1

[probe]
look
score
`
	head := stripProbeSection(raw)
	if strings.Contains(head, "[probe]") || strings.Contains(head, "score") {
		t.Fatalf("stripProbeSection kept probe lines:\n%s", head)
	}
	rendered := renderLockstepScenario(head, []string{"~dpclock pulse 40", "look", ""})
	sc, err := oraclediff.ParseScenario("rendered", strings.NewReader(rendered))
	if err != nil {
		t.Fatalf("ParseScenario(rendered): %v\n%s", err, rendered)
	}
	if !sc.QuietAllMobs {
		t.Fatal("rendered scenario lost quiet-mobs fixture")
	}
	wantProbe := []string{"~dpclock pulse 40", "look", ""}
	if len(sc.Probe) != len(wantProbe) {
		t.Fatalf("probe len = %d, want %d (%v)", len(sc.Probe), len(wantProbe), sc.Probe)
	}
	for i := range wantProbe {
		if sc.Probe[i] != wantProbe[i] {
			t.Fatalf("probe[%d] = %q, want %q", i, sc.Probe[i], wantProbe[i])
		}
	}
}

func TestWriteScenarioAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "out.txt")
	if err := writeScenarioAtomic(path, "[probe]\nlook\n"); err != nil {
		t.Fatalf("writeScenarioAtomic: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "[probe]\nlook\n" {
		t.Fatalf("content = %q, want %q", string(got), "[probe]\nlook\n")
	}
}

func TestLockstepComparatorMatchesCensusComparator(t *testing.T) {
	// Verify that the shared comparator returns identical verdicts for
	// agreeing, differing, and ledger-EXPECTED blocks when called with the
	// same inputs from census and lockstep.
	scenario := oraclediff.Scenario{Name: "alias-cap"}
	oracleSame := oraclediff.AudienceProbeBlock{Command: "look", Audience: "actor", Output: "Room\r\n20H 100M 82V > "}
	goSame := oraclediff.AudienceProbeBlock{Command: "look", Audience: "actor", Output: "Room\n20H 100M 82V > "}

	censusSame := oraclediff.CompareProbeBlock(scenario, oracleSame, goSame, true, nil)
	lockstepSame := oraclediff.CompareProbeBlock(scenario, oracleSame, goSame, true, nil)
	if censusSame.Verdict != oraclediff.VerdictSame || lockstepSame.Verdict != censusSame.Verdict {
		t.Fatalf("same verdict mismatch: census=%q lockstep=%q", censusSame.Verdict, lockstepSame.Verdict)
	}

	goDiff := oraclediff.AudienceProbeBlock{Command: "look", Audience: "actor", Output: "Other\n20H 100M 82V > "}
	censusDiff := oraclediff.CompareProbeBlock(scenario, oracleSame, goDiff, true, nil)
	lockstepDiff := oraclediff.CompareProbeBlock(scenario, oracleSame, goDiff, true, nil)
	if censusDiff.Verdict != oraclediff.VerdictDiverged || lockstepDiff.Verdict != censusDiff.Verdict {
		t.Fatalf("diverged verdict mismatch: census=%q lockstep=%q", censusDiff.Verdict, lockstepDiff.Verdict)
	}

	divTSV := "scenario\tmanifest\tcase_id\tstatus\tstability\nalias-cap\talias.tsv\talias.cap\texcluded\t\n"
	pinsTSV := "scenario\tlabel\tsha256\tcitations\nalias-cap\tlook\t" + censusDiff.Fingerprint + "\talias.tsv:alias.cap:excluded\n"
	ledger, err := oraclediff.ParseExpectedLedger(strings.NewReader(divTSV), strings.NewReader(pinsTSV))
	if err != nil {
		t.Fatalf("ParseExpectedLedger: %v", err)
	}
	censusExp := oraclediff.CompareProbeBlock(scenario, oracleSame, goDiff, true, ledger)
	lockstepExp := oraclediff.CompareProbeBlock(scenario, oracleSame, goDiff, true, ledger)
	if censusExp.Verdict != oraclediff.VerdictExpected || lockstepExp.Verdict != censusExp.Verdict {
		t.Fatalf("expected verdict mismatch: census=%q lockstep=%q", censusExp.Verdict, lockstepExp.Verdict)
	}
}
