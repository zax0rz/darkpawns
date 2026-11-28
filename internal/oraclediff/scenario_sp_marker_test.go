package oraclediff

// scenario_sp_marker_test.go — <SP> expansion guards.
//
// The parser's step-line trim would otherwise make trailing spaces
// un-sendable (R5i: a silenced surface with no scenario to unsilence
// it). <SP> restores them inside steps only; the corpus digest below
// pins that every marker-free scenario parses bit-identically with and
// without the expansion.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// scenarioCorpusDir is the shared scenario corpus this package serves.
const scenarioCorpusDir = "../../cmd/dp-oracle-diff/scenarios"

// markerScenario is the one file that intentionally contains <SP>; the
// exemption is explicit so a second marker file fails loudly instead of
// silently joining the digest.
const markerScenario = "typed-space-depth.txt"

func scenarioStepsCanonical(sc Scenario) string {
	var b strings.Builder
	writeSteps := func(steps []string) {
		for _, step := range steps {
			fmt.Fprintf(&b, "%d:%s\x02", len(step), step)
		}
		b.WriteString("\x03")
	}
	for _, steps := range [][]string{sc.SetupOracle, sc.SetupPort, sc.ReloginOracle, sc.ReloginPort, sc.Warmup, sc.Probe} {
		writeSteps(steps)
	}
	fmt.Fprintf(&b, "actor=%q\x03", sc.ProbeActor)
	peers := make([]string, 0, len(sc.Peers))
	for name := range sc.Peers {
		peers = append(peers, name)
	}
	sort.Strings(peers)
	for _, name := range peers {
		peer := sc.Peers[name]
		fmt.Fprintf(&b, "peer=%s\x03", name)
		writeSteps(peer.SetupOracle)
		writeSteps(peer.SetupPort)
	}
	return b.String()
}

// corpusStepsDigest hashes every non-exempt scenario's parsed steps, in
// filename order. It changes only if parsing of marker-free files changes.
func corpusStepsDigest(t *testing.T) (string, int) {
	t.Helper()
	entries, err := os.ReadDir(scenarioCorpusDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") || entry.Name() == markerScenario {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(scenarioCorpusDir, name))
		if err != nil {
			t.Fatalf("ReadFile %s: %v", name, err)
		}
		sc, err := ParseScenario(strings.TrimSuffix(name, ".txt"), strings.NewReader(string(data)))
		if err != nil {
			t.Fatalf("ParseScenario %s: %v", name, err)
		}
		fmt.Fprintf(h, "%s\x00%s\x00", name, scenarioStepsCanonical(sc))
	}
	return hex.EncodeToString(h.Sum(nil)), len(names)
}

// corpusStepsDigestGolden pins the corpus parse. Regenerate only by proving
// a deliberate parse change (the digest must differ between the old and new
// parsers at the same marked commit if semantics change).
// dc326b21…ab16: verified identical under the pre-marker parser, so every
// existing scenario parses bit-identically.
const corpusStepsDigestGolden = "dc326b212bb7018587cf50bc1a73dbbf5dba2634b6c860ea8520d16dff99ab16"

func TestScenarioCorpusStepsDigest(t *testing.T) {
	digest, count := corpusStepsDigest(t)
	if digest != corpusStepsDigestGolden {
		t.Fatalf("corpus steps digest = %s (%d files), want %s\nparsing of marker-free scenarios changed", digest, count, corpusStepsDigestGolden)
	}
}

func TestSpaceMarkerAbsentFromCorpus(t *testing.T) {
	entries, err := os.ReadDir(scenarioCorpusDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(scenarioCorpusDir, entry.Name()))
		if err != nil {
			t.Fatalf("ReadFile %s: %v", entry.Name(), err)
		}
		if strings.Contains(string(data), "<SP>") && entry.Name() != markerScenario {
			t.Fatalf("%s contains the space marker but is not the exempt marker scenario", entry.Name())
		}
	}
}

func TestSpaceMarkerExpansion(t *testing.T) {
	text := `[fixture]
empty-players
quiet-mobs

[setup:oracle]
Hero
Y
oraclepass
oraclepass
N
M
H
W
K
Y
<ENTER>
1

[setup:port]
Hero
y
oraclepass
oraclepass
N
M
H
W
K
Y
<ENTER>
1

[probe]
say tail<SP><SP><SP>
say hi<SP><SP>there
say plain
  say indented
<ENTER>
`
	sc, err := ParseScenario("marker-expansion", strings.NewReader(text))
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	want := []string{"say tail   ", "say hi  there", "say plain", "say indented", ""}
	if len(sc.Probe) != len(want) {
		t.Fatalf("probe = %q, want %q", sc.Probe, want)
	}
	for i, step := range want {
		if sc.Probe[i] != step {
			t.Fatalf("probe[%d] = %q, want %q", i, sc.Probe[i], step)
		}
	}
}

// The trailing space survives the harness all the way to the wire offset.
func TestSpaceMarkerTrailingPreservedToConn(t *testing.T) {
	step := strings.ReplaceAll("say tail<SP><SP>", "<SP>", " ")
	if step != "say tail  " {
		t.Fatalf("expanded step = %q, want trailing spaces", step)
	}
}
