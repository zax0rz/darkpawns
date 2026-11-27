package oraclediff

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// BlockVerdict is the single-block verdict produced by the shared census and
// lockstep comparator.
type BlockVerdict string

const (
	VerdictSame     BlockVerdict = "same"
	VerdictExpected BlockVerdict = "expected"
	VerdictDiverged BlockVerdict = "diverged"
)

// ExpectedLedger holds the census expected-divergence ledger
// (cmd/dp-oracle-diff/expected_divergences.tsv) and its per-block shape pins
// (cmd/dp-oracle-diff/expected_divergence_pins.tsv).
type ExpectedLedger struct {
	baselined   map[string]bool
	runVarying  map[string]bool
	pinnedShape map[string]map[string]map[string]bool // scenario -> label -> sha256 -> true
}

// LoadExpectedLedger loads the ledger and pin TSVs from disk. Missing files
// return an empty ledger (no expected divergences).
func LoadExpectedLedger(divergencesPath, pinsPath string) (*ExpectedLedger, error) {
	var divReader, pinsReader io.Reader
	if divergencesPath != "" {
		f, err := os.Open(filepath.Clean(divergencesPath)) // #nosec G304 -- trusted harness ledger path
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("open expected divergences %s: %w", divergencesPath, err)
			}
		} else {
			defer func() { _ = f.Close() }()
			divReader = f
		}
	}
	if pinsPath != "" {
		f, err := os.Open(filepath.Clean(pinsPath)) // #nosec G304 -- trusted harness pin path
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("open expected divergence pins %s: %w", pinsPath, err)
			}
		} else {
			defer func() { _ = f.Close() }()
			pinsReader = f
		}
	}
	return ParseExpectedLedger(divReader, pinsReader)
}

// ParseExpectedLedger parses the expected_divergences.tsv and
// expected_divergence_pins.tsv streams using the exact same column contract as
// scripts/oracle_regression_worker.sh.
func ParseExpectedLedger(divergences, pins io.Reader) (*ExpectedLedger, error) {
	l := &ExpectedLedger{
		baselined:   make(map[string]bool),
		runVarying:  make(map[string]bool),
		pinnedShape: make(map[string]map[string]map[string]bool),
	}
	if divergences != nil {
		sc := bufio.NewScanner(divergences)
		lineNo := 0
		for sc.Scan() {
			lineNo++
			line := strings.TrimRight(sc.Text(), "\r\n")
			if line == "" || (lineNo == 1 && strings.HasPrefix(line, "scenario\t")) {
				continue
			}
			cols := strings.Split(line, "\t")
			if len(cols) < 1 || cols[0] == "" {
				continue
			}
			scenario := cols[0]
			l.baselined[scenario] = true
			if len(cols) >= 5 && cols[4] == "run-varying" {
				l.runVarying[scenario] = true
			}
		}
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("read expected divergences: %w", err)
		}
	}
	if pins != nil {
		sc := bufio.NewScanner(pins)
		lineNo := 0
		for sc.Scan() {
			lineNo++
			line := strings.TrimRight(sc.Text(), "\r\n")
			if line == "" || (lineNo == 1 && strings.HasPrefix(line, "scenario\t")) {
				continue
			}
			cols := strings.Split(line, "\t")
			if len(cols) < 3 || cols[0] == "" || cols[1] == "" || len(cols[2]) != 64 {
				continue
			}
			scenario, label, sha := cols[0], cols[1], cols[2]
			if l.pinnedShape[scenario] == nil {
				l.pinnedShape[scenario] = make(map[string]map[string]bool)
			}
			if l.pinnedShape[scenario][label] == nil {
				l.pinnedShape[scenario][label] = make(map[string]bool)
			}
			l.pinnedShape[scenario][label][sha] = true
		}
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("read expected divergence pins: %w", err)
		}
	}
	return l, nil
}

// DiffFingerprint returns the canonical sha256 hex digest of a unified diff
// (the same digest printed on dp-oracle-diff's divergence-fingerprint lines).
func DiffFingerprint(diff string) string {
	if diff == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(diff))
	return hex.EncodeToString(sum[:])
}

// ClassifyBlock classifies one compared block against the expected-divergence
// ledger and pins.
func (l *ExpectedLedger) ClassifyBlock(scenarioName, label, diff string) BlockVerdict {
	if diff == "" {
		return VerdictSame
	}
	if l == nil || !l.baselined[scenarioName] {
		return VerdictDiverged
	}
	if l.runVarying[scenarioName] {
		return VerdictExpected
	}
	fp := DiffFingerprint(diff)
	if labels := l.pinnedShape[scenarioName]; labels != nil {
		if shapes := labels[label]; shapes != nil && shapes[fp] {
			return VerdictExpected
		}
	}
	return VerdictDiverged
}

// BlockComparison is the complete result of comparing one oracle block against
// its Go port counterpart.
type BlockComparison struct {
	Diff        BlockDiff
	Verdict     BlockVerdict
	Fingerprint string
	Hollow      bool
}

// NormalizerFor returns the tier-1 normalizer selected by a scenario's
// fixtures. Both census and lockstep call this function.
func NormalizerFor(scenario Scenario) func(string) string {
	normalize := Normalize
	if scenario.KeepANSI {
		normalize = NormalizeKeepANSI
	}
	if scenario.KeepPrompts {
		normalize = NormalizeKeepPrompts
	}
	return normalize
}

// CompareProbeBlock compares one oracle probe block against its Go counterpart
// using the exact census normalization, close-marker handling, two-sided
// prompt-frame reconciliation, unified diff, hollow-step detection, and
// expected-divergence ledger classification.
func CompareProbeBlock(scenario Scenario, oracleResult, goResult AudienceProbeBlock, hasGo bool, ledger *ExpectedLedger) BlockComparison {
	normalize := NormalizerFor(scenario)
	oracleRaw := oracleResult.Output
	var goRaw string
	goClosed := false
	if hasGo {
		goRaw = goResult.Output
		goClosed = goResult.Closed
	}
	if scenario.CompareClose {
		oracleRaw = WithCloseMarker(oracleRaw, oracleResult.Closed)
		goRaw = WithCloseMarker(goRaw, goClosed)
	}
	oracleBlock := normalize(oracleRaw)
	var goBlock string
	if hasGo {
		goBlock = normalize(goRaw)
	}
	oracleBefore, goBefore := oracleBlock, goBlock
	hollow := false
	if !scenario.KeepANSI && !scenario.KeepPrompts {
		oracleBlock, goBlock = ReconcilePromptFrames(oracleBlock, goBlock)
		if hasGo && IsHollowPromptStep(oracleResult.Command, oracleResult.Audience, oracleRaw, goRaw, oracleBefore, goBefore, oracleBlock, goBlock) {
			hollow = true
		}
	}
	label := oracleResult.Command
	if len(scenario.Peers) > 0 {
		label = fmt.Sprintf("%s [%s]", oracleResult.Command, oracleResult.Audience)
	}
	diffText := UnifiedDiff("c-oracle", "go-port", oracleBlock, goBlock)
	return BlockComparison{
		Diff: BlockDiff{
			Command: label,
			Oracle:  oracleBlock,
			Go:      goBlock,
			Diff:    diffText,
		},
		Verdict:     ledger.ClassifyBlock(scenario.Name, label, diffText),
		Fingerprint: DiffFingerprint(diffText),
		Hollow:      hollow,
	}
}

// WithCloseMarker appends the canonical closed-transport marker to a captured
// block when the server closed that connection during the block's read.
func WithCloseMarker(raw string, closed bool) string {
	if !closed {
		return raw
	}
	if raw != "" && !strings.HasSuffix(raw, "\n") {
		raw += "\n"
	}
	return raw + CloseMarker + "\n"
}

func isActorProbeBlock(command, audience string) bool {
	if !strings.HasPrefix(command, "send:") {
		return audience == "actor"
	}
	directed := strings.TrimPrefix(command, "send:")
	target, _, _ := strings.Cut(directed, " ")
	return target != "" && audience == target
}

func isPulseControlStep(command string) bool {
	line := command
	if directed, ok := strings.CutPrefix(command, "send:"); ok {
		_, after, ok := strings.Cut(directed, " ")
		if !ok {
			return false
		}
		line = after
	}
	return strings.HasPrefix(strings.TrimSpace(line), pulseControl)
}

func isEmptyNormalizedBlock(s string) bool {
	return s == "" || s == "\n"
}

// IsHollowPromptStep reports whether an actor probe step had one engine emit
// nothing while the other emitted only framing/prompt bytes that Normalize or
// ReconcilePromptFrames stripped, leaving both post-reconcile blocks empty.
func IsHollowPromptStep(command, audience, oracleRaw, goRaw, oracleBefore, goBefore, oracleAfter, goAfter string) bool {
	if !isActorProbeBlock(command, audience) || isPulseControlStep(command) {
		return false
	}
	if !isEmptyNormalizedBlock(oracleAfter) || !isEmptyNormalizedBlock(goAfter) {
		return false
	}
	return isEmptyNormalizedBlock(oracleBefore) != isEmptyNormalizedBlock(goBefore) ||
		(oracleRaw == "") != (goRaw == "")
}
