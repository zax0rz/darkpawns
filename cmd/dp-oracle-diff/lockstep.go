package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

type lockstepConfig struct {
	scenarioName     string
	scenarioFile     string
	lockstepOut      string
	quiescence       time.Duration
	firstByteWait    time.Duration
	bootTimeout      time.Duration
	oracleBin        string
	seed             string
	drawParity       bool
	expectedDivPath  string
	expectedPinsPath string
}

// lockstepRequest is one JSONL line read from stdin in -lockstep mode.
type lockstepRequest struct {
	Type    string `json:"type"`
	Command string `json:"command,omitempty"`
	Pulses  int    `json:"pulses,omitempty"`
}

// lockstepResponse is one JSONL line written to stdout per request.
type lockstepResponse struct {
	Step             int                     `json:"step"`
	Type             string                  `json:"type"`
	Command          string                  `json:"command,omitempty"`
	OracleRaw        string                  `json:"oracle_raw"`
	OracleNormalized string                  `json:"oracle_normalized"`
	GoNormalized     string                  `json:"go_normalized"`
	Diff             string                  `json:"diff,omitempty"`
	Verdict          oraclediff.BlockVerdict `json:"verdict"`
	Fingerprint      string                  `json:"fingerprint,omitempty"`
	DrawParity       string                  `json:"draw_parity"`
	DrawStream       int                     `json:"draw_stream,omitempty"`
	DrawDetail       string                  `json:"draw_detail,omitempty"`
	OracleClosed     bool                    `json:"oracle_closed,omitempty"`
	GoClosed         bool                    `json:"go_closed,omitempty"`
	CEntries         int                     `json:"c_entries,omitempty"`
	GoEntries        int                     `json:"go_entries,omitempty"`
	OK               bool                    `json:"ok,omitempty"`
}

func executeLockstep(cfg lockstepConfig, in io.Reader, out io.Writer) error {
	scenario, rawScenario, err := loadScenario(cfg.scenarioName, cfg.scenarioFile, true)
	if err != nil {
		return err
	}
	h, err := bootHarness(scenario, harnessConfig{
		quiescence:    cfg.quiescence,
		firstByteWait: cfg.firstByteWait,
		bootTimeout:   cfg.bootTimeout,
		oracleBin:     cfg.oracleBin,
		seed:          cfg.seed,
		goTransport:   "telnet",
		drawParity:    cfg.drawParity,
	})
	if err != nil {
		return err
	}
	defer h.close()

	divPath := cfg.expectedDivPath
	if divPath == "" {
		divPath = filepath.Join(h.repoRoot, "cmd", "dp-oracle-diff", "expected_divergences.tsv")
	}
	pinsPath := cfg.expectedPinsPath
	if pinsPath == "" {
		pinsPath = filepath.Join(h.repoRoot, "cmd", "dp-oracle-diff", "expected_divergence_pins.tsv")
	}
	ledger, err := oraclediff.LoadExpectedLedger(divPath, pinsPath)
	if err != nil {
		return err
	}

	headText := stripProbeSection(rawScenario)
	var recordedSteps []string
	writeOut := func() error {
		if cfg.lockstepOut == "" {
			return nil
		}
		return writeScenarioAtomic(cfg.lockstepOut, renderLockstepScenario(headText, recordedSteps))
	}
	if err := writeOut(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)

	stepNum := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req lockstepRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			return fmt.Errorf("lockstep decode request: %w", err)
		}
		reqType := strings.ToLower(strings.TrimSpace(req.Type))
		if reqType == "" {
			if req.Pulses > 0 {
				reqType = "pulse"
			} else {
				reqType = "command"
			}
		}

		if reqType == "end" {
			if err := writeOut(); err != nil {
				return err
			}
			dpStatus, dpStream, dpDetail, cCount, goCount, dpErr := checkLockstepDraws(h)
			if dpErr != nil {
				return dpErr
			}
			return enc.Encode(lockstepResponse{
				Step:       stepNum,
				Type:       "end",
				Verdict:    oraclediff.VerdictSame,
				DrawParity: dpStatus,
				DrawStream: dpStream,
				DrawDetail: dpDetail,
				CEntries:   cCount,
				GoEntries:  goCount,
				OK:         true,
			})
		}

		var stepLine string
		switch reqType {
		case "command":
			stepLine = req.Command
		case "pulse":
			if req.Pulses <= 0 {
				return fmt.Errorf("lockstep pulse request requires positive pulses, got %d", req.Pulses)
			}
			stepLine = fmt.Sprintf("~dpclock pulse %d", req.Pulses)
		default:
			return fmt.Errorf("lockstep unknown request type %q", req.Type)
		}

		stepIdx := stepNum
		stepNum++

		var oracleBlocks, goBlocks []oraclediff.AudienceProbeBlock
		var oracleErr, goErr error
		done := make(chan struct{}, 2)
		if reqType == "pulse" && len(h.oracleAudience) == 0 && len(h.goAudience) == 0 {
			go func() {
				defer func() { done <- struct{}{} }()
				blk, err := oraclediff.RunPulseWithBarrier(h.oracleActor, req.Pulses, h.quiescence)
				if err != nil {
					oracleErr = err
					return
				}
				if h.drawParity {
					h.oracleDraws = append(h.oracleDraws, oraclediff.DrawBoundary{
						Label: fmt.Sprintf("probe step %d: %s", stepIdx+1, stepLine),
						Count: oraclediff.CountDrawLines(h.cDrawLog),
					})
				}
				oracleBlocks = []oraclediff.AudienceProbeBlock{blk}
			}()
			go func() {
				defer func() { done <- struct{}{} }()
				blk, err := oraclediff.RunPulseWithBarrier(h.goActor, req.Pulses, h.quiescence)
				if err != nil {
					goErr = err
					return
				}
				if h.drawParity {
					h.goDraws = append(h.goDraws, oraclediff.DrawBoundary{
						Label: fmt.Sprintf("probe step %d: %s", stepIdx+1, stepLine),
						Count: oraclediff.CountDrawLines(h.goDrawLog),
					})
				}
				goBlocks = []oraclediff.AudienceProbeBlock{blk}
			}()
		} else {
			go func() {
				defer func() { done <- struct{}{} }()
				oracleBlocks, oracleErr = oraclediff.RunSingleAudienceProbeStep(
					h.oracleActor, h.oracleAudience, stepLine, stepIdx, true, h.quiescence,
					oraclediff.ProbeOptions{
						AfterStep: func(i int, step string) {
							if h.drawParity {
								h.oracleDraws = append(h.oracleDraws, oraclediff.DrawBoundary{
									Label: fmt.Sprintf("probe step %d: %s", i+1, step),
									Count: oraclediff.CountDrawLines(h.cDrawLog),
								})
							}
						},
						Fault:     func(step string) error { return applyFaultStep(h.faults, step) },
						QuietMobs: scenario.QuietAllMobs,
					},
				)
			}()
			go func() {
				defer func() { done <- struct{}{} }()
				goBlocks, goErr = oraclediff.RunSingleAudienceProbeStep(
					h.goActor, h.goAudience, stepLine, stepIdx, true, h.quiescence,
					oraclediff.ProbeOptions{
						AfterStep: func(i int, step string) {
							if h.drawParity {
								h.goDraws = append(h.goDraws, oraclediff.DrawBoundary{
									Label: fmt.Sprintf("probe step %d: %s", i+1, step),
									Count: oraclediff.CountDrawLines(h.goDrawLog),
								})
							}
						},
						Fault:     func(step string) error { return applyFaultStep(h.faults, step) },
						QuietMobs: scenario.QuietAllMobs,
					},
				)
			}()
		}
		<-done
		<-done
		if oracleErr != nil {
			return fmt.Errorf("lockstep C oracle step %d (%q): %w\nserver log:\n%s", stepNum, stepLine, oracleErr, h.oracleEngine.log())
		}
		if goErr != nil {
			return fmt.Errorf("lockstep Go port step %d (%q): %w\nserver log:\n%s", stepNum, stepLine, goErr, h.goEngine.log())
		}

		recordedSteps = append(recordedSteps, stepLine)
		if reqType == "pulse" && len(h.oracleAudience) == 0 && len(h.goAudience) == 0 {
			recordedSteps = append(recordedSteps, "")
		}
		if err := writeOut(); err != nil {
			return err
		}

		var oracleBlock, goBlock oraclediff.AudienceProbeBlock
		if len(oracleBlocks) > 0 {
			oracleBlock = oracleBlocks[0]
		}
		hasGo := len(goBlocks) > 0
		if hasGo {
			goBlock = goBlocks[0]
		}
		cmp := oraclediff.CompareProbeBlock(scenario, oracleBlock, goBlock, hasGo, ledger)

		dpStatus, dpStream, dpDetail, cCount, goCount, dpErr := checkLockstepDraws(h)
		if dpErr != nil {
			return dpErr
		}

		resp := lockstepResponse{
			Step:             stepNum,
			Type:             reqType,
			Command:          stepLine,
			OracleRaw:        oracleBlock.Output,
			OracleNormalized: cmp.Diff.Oracle,
			GoNormalized:     cmp.Diff.Go,
			Diff:             cmp.Diff.Diff,
			Verdict:          cmp.Verdict,
			Fingerprint:      cmp.Fingerprint,
			DrawParity:       dpStatus,
			DrawStream:       dpStream,
			DrawDetail:       dpDetail,
			OracleClosed:     oracleBlock.Closed,
			GoClosed:         goBlock.Closed,
			CEntries:         cCount,
			GoEntries:        goCount,
			OK:               true,
		}
		if err := enc.Encode(resp); err != nil {
			return fmt.Errorf("lockstep encode response: %w", err)
		}
	}
	return scanner.Err()
}

func checkLockstepDraws(h *harnessSession) (status string, stream int, detail string, cCount, goCount int, err error) {
	if !h.drawParity {
		return "disabled", 0, "", 0, 0, nil
	}
	cDraws, err := oraclediff.ReadDrawLog(h.cDrawLog)
	if err != nil {
		return "", 0, "", 0, 0, fmt.Errorf("read C draw log: %w", err)
	}
	goDraws, err := oraclediff.ReadDrawLog(h.goDrawLog)
	if err != nil {
		return "", 0, "", 0, 0, fmt.Errorf("read Go draw log: %w", err)
	}
	if len(cDraws) == 0 {
		return "", 0, "", 0, 0, fmt.Errorf("draw-parity: the C oracle wrote no draw log (is DP_ORACLE_BIN built with the DP draw-log seam?)")
	}
	r := oraclediff.CompareDraws(cDraws, goDraws)
	if r.Match {
		return "match", r.Stream, "", r.CCount, r.GoCount, nil
	}
	call := func(draws []oraclediff.DrawCall, i int) string {
		if i < len(draws) {
			return draws[i].String()
		}
		return "(log ends)"
	}
	detail = fmt.Sprintf("stream=%d c=%s go=%s c-step=%s go-step=%s",
		r.Stream, call(cDraws, r.CEntry), call(goDraws, r.GoEntry),
		oraclediff.StepFor(h.oracleDraws, r.CEntry), oraclediff.StepFor(h.goDraws, r.GoEntry))
	return "mismatch", r.Stream, detail, r.CCount, r.GoCount, nil
}

// stripProbeSection removes any existing [probe] (or [probe:*]) section from a
// scenario text so -lockstep-out preserves comments, fixtures, setup, relogin,
// and warmup verbatim and appends the live-executed [probe] steps.
func stripProbeSection(raw string) string {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	var kept []string
	inProbe := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			lower := strings.ToLower(trimmed)
			if lower == "[probe]" || strings.HasPrefix(lower, "[probe:") {
				inProbe = true
				continue
			}
			inProbe = false
		}
		if !inProbe {
			kept = append(kept, line)
		}
	}
	return strings.TrimRight(strings.Join(kept, "\n"), "\n")
}

func renderLockstepScenario(head string, steps []string) string {
	var b strings.Builder
	b.WriteString(head)
	b.WriteString("\n\n[probe]\n")
	for _, step := range steps {
		if step == "" {
			b.WriteString("<ENTER>\n")
		} else {
			b.WriteString(step)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func writeScenarioAtomic(path, content string) error {
	clean := filepath.Clean(path)
	dir := filepath.Dir(clean)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create lockstep-out directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".lockstep-out-*.tmp")
	if err != nil {
		return fmt.Errorf("create lockstep-out temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write lockstep-out temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close lockstep-out temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil { // #nosec G302 -- scenario files are world-readable checked-in fixtures
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod lockstep-out temp file: %w", err)
	}
	if err := os.Rename(tmpPath, clean); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename lockstep-out file: %w", err)
	}
	return nil
}
