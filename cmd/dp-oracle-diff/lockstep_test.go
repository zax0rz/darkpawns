package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestLockstepPulseBarrierReplayMatchesCensusIncludingIdleToVoid(t *testing.T) {
	oracleBin := os.Getenv("DP_ORACLE_BIN")
	if oracleBin == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("no home dir for DP_ORACLE_BIN")
		}
		oracleBin = filepath.Join(home, "darkpawns-c-oracle", "bin", "circle")
	}
	if _, err := os.Stat(oracleBin); err != nil {
		t.Skipf("C oracle binary not present at %s: %v", oracleBin, err)
	}

	tmp := t.TempDir()
	headPath := filepath.Join(tmp, "idle-head.txt")
	outPath := filepath.Join(tmp, "idle-out.txt")
	headContent := `# Lockstep IDLE_TO_VOID barrier replay equivalence test
[fixture]
quiet-mobs

[setup:oracle]
Idletest
Y
oraclepass
oraclepass
Y
M
H
W
K
Y
<ENTER>
1

[setup:port]
Idletest
Y
oraclepass
oraclepass
Y
M
H
W
K
Y
<ENTER>
1

[probe]
look
`
	if err := os.WriteFile(headPath, []byte(headContent), 0o600); err != nil {
		t.Fatalf("write head scenario: %v", err)
	}

	// 9 mud hours (9 * 630 = 5670 pulses) exceeds IDLE_TO_VOID (8, src/utils.h:131).
	input := strings.Join([]string{
		`{"type":"command","command":"look"}`,
		`{"type":"pulse","pulses":5670}`,
		`{"type":"command","command":"look"}`,
		`{"type":"end"}`,
	}, "\n") + "\n"
	var outBuf bytes.Buffer
	err := executeLockstep(lockstepConfig{
		scenarioFile:  headPath,
		lockstepOut:   outPath,
		quiescence:    400 * time.Millisecond,
		firstByteWait: 2 * time.Second,
		bootTimeout:   30 * time.Second,
		oracleBin:     oracleBin,
		seed:          "1",
		drawParity:    true,
	}, strings.NewReader(input), &outBuf)
	if err != nil {
		t.Fatalf("executeLockstep: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 JSONL responses, got %d:\n%s", len(lines), outBuf.String())
	}
	var pulseResp lockstepResponse
	for i, l := range lines {
		var resp lockstepResponse
		if err := json.Unmarshal([]byte(l), &resp); err != nil {
			t.Fatalf("unmarshal response %d: %v", i, err)
		}
		if resp.Verdict != oraclediff.VerdictSame {
			t.Fatalf("step %d (%s) verdict=%q diff:\n%s", resp.Step, resp.Command, resp.Verdict, resp.Diff)
		}
		if resp.DrawParity != "match" {
			t.Fatalf("step %d (%s) draw_parity=%q detail=%s", resp.Step, resp.Command, resp.DrawParity, resp.DrawDetail)
		}
		if resp.Type == "pulse" {
			pulseResp = resp
		}
	}
	if pulseResp.BarrierStep != "<ENTER>" {
		t.Fatalf("pulse response BarrierStep = %q, want <ENTER>", pulseResp.BarrierStep)
	}
	if !strings.Contains(pulseResp.OracleRaw, "pulled into a void") {
		t.Fatalf("expected IDLE_TO_VOID void message in 5670-pulse output, got:\n%s", pulseResp.OracleRaw)
	}

	// Verify the written scenario records "<ENTER>" as a real probe step.
	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read lockstep-out: %v", err)
	}
	sc, err := oraclediff.ParseScenario("idle-out", bytes.NewReader(written))
	if err != nil {
		t.Fatalf("ParseScenario(idle-out): %v\n%s", err, string(written))
	}
	wantProbe := []string{"look", "~dpclock pulse 5670", "", "look"}
	if len(sc.Probe) != len(wantProbe) {
		t.Fatalf("written probe = %v, want %v\nfile:\n%s", sc.Probe, wantProbe, string(written))
	}
	for i := range wantProbe {
		if sc.Probe[i] != wantProbe[i] {
			t.Fatalf("written probe[%d] = %q, want %q", i, sc.Probe[i], wantProbe[i])
		}
	}

	// Replay the written scenario through batch execute() with draw parity to
	// prove lockstep and batch census execute the exact same timeline.
	if err := execute("", outPath, 400*time.Millisecond, 2*time.Second, 30*time.Second, oracleBin, "1", false, "", false, "telnet", true, true); err != nil {
		t.Fatalf("batch execute replay of lockstep-out failed: %v", err)
	}
}
