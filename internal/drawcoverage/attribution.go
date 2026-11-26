package drawcoverage

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// AttributedCall is one resolved top-level draw event from a single capture.
type AttributedCall struct {
	Site      string
	Primitive string
}

// Capture is the verified draw-site attribution of one (seed, scenario) pair.
type Capture struct {
	Seed     int
	Scenario string
	Calls    []AttributedCall
}

// Exclusion records a (seed, scenario) pair dropped from the reach count
// because its attributed draw stream failed parity or alignment checks.
type Exclusion struct {
	Seed     int
	Scenario string
	Reason   string
}

// LoadCensusRun reads every (seed, scenario) capture from attrRunDir, checks
// byte-for-byte c-draws.log parity against refRunDir (when non-empty), checks
// 1:1 line and primitive alignment between c-draws.log and c-draw-sites.log,
// and resolves each return address through inventory.
func LoadCensusRun(attrRunDir, refRunDir string, inventory []DrawSite) ([]Capture, []Exclusion, error) {
	byRet := make(map[uint64]DrawSite, len(inventory))
	for _, s := range inventory {
		byRet[s.RetVaddr] = s
	}

	attrPairs, err := discoverCaptures(attrRunDir)
	if err != nil {
		return nil, nil, fmt.Errorf("discover attributed captures in %s: %w", attrRunDir, err)
	}
	var refPairs map[pairKey]string
	if refRunDir != "" {
		refPairs, err = discoverCaptures(refRunDir)
		if err != nil {
			return nil, nil, fmt.Errorf("discover reference captures in %s: %w", refRunDir, err)
		}
	}

	keys := make([]pairKey, 0, len(attrPairs))
	for k := range attrPairs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].seed != keys[j].seed {
			return keys[i].seed < keys[j].seed
		}
		return keys[i].scenario < keys[j].scenario
	})

	var captures []Capture
	var exclusions []Exclusion
	for _, k := range keys {
		attrDir := attrPairs[k]
		attrDraws, err := os.ReadFile(filepath.Join(attrDir, "c-draws.log")) // #nosec G304 -- trusted local census run directory
		if err != nil {
			exclusions = append(exclusions, Exclusion{Seed: k.seed, Scenario: k.scenario, Reason: "missing_attr_draws"})
			continue
		}
		if refPairs != nil {
			refDir, ok := refPairs[k]
			if !ok {
				exclusions = append(exclusions, Exclusion{Seed: k.seed, Scenario: k.scenario, Reason: "missing_ref_capture"})
				continue
			}
			refDraws, err := os.ReadFile(filepath.Join(refDir, "c-draws.log")) // #nosec G304 -- trusted local census run directory
			if err != nil {
				exclusions = append(exclusions, Exclusion{Seed: k.seed, Scenario: k.scenario, Reason: "missing_ref_draws"})
				continue
			}
			if !bytes.Equal(attrDraws, refDraws) {
				exclusions = append(exclusions, Exclusion{Seed: k.seed, Scenario: k.scenario, Reason: "draw_stream_mismatch"})
				continue
			}
		}

		sitesPath := filepath.Join(attrDir, "c-draw-sites.log")
		calls, reason := parseAndVerifyCapture(attrDraws, sitesPath, byRet)
		if reason != "" {
			exclusions = append(exclusions, Exclusion{Seed: k.seed, Scenario: k.scenario, Reason: reason})
			continue
		}
		captures = append(captures, Capture{
			Seed:     k.seed,
			Scenario: k.scenario,
			Calls:    calls,
		})
	}
	return captures, exclusions, nil
}

type pairKey struct {
	seed     int
	scenario string
}

// discoverCaptures finds the final capture directory for every (seed, scenario)
// in a combined or single-seed census run. Recheck logs override initial logs
// when present, and higher capture attempt numbers override earlier ones.
func discoverCaptures(runDir string) (map[pairKey]string, error) {
	out := map[pairKey]string{}
	entries, err := os.ReadDir(runDir)
	if err != nil {
		return nil, err
	}
	var seedDirs []struct {
		seed int
		dir  string
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if after, ok := strings.CutPrefix(e.Name(), "seed"); ok {
			if n, err := strconv.Atoi(after); err == nil && n > 0 {
				seedDirs = append(seedDirs, struct {
					seed int
					dir  string
				}{seed: n, dir: filepath.Join(runDir, e.Name())})
			}
		}
	}
	// Support single-seed runs where initial-logs lives directly under runDir.
	if len(seedDirs) == 0 {
		seedDirs = append(seedDirs, struct {
			seed int
			dir  string
		}{seed: 1, dir: runDir})
	}

	for _, sd := range seedDirs {
		// Scan initial-logs first, then recheck-logs so a clean recheck supersedes
		// an earlier infra attempt.
		for _, sub := range []string{"initial-logs", "recheck-logs"} {
			logRoot := filepath.Join(sd.dir, sub)
			attempts, err := os.ReadDir(logRoot)
			if err != nil {
				continue
			}
			for _, att := range attempts {
				if !att.IsDir() || !strings.HasPrefix(att.Name(), "attempt.") {
					continue
				}
				attDir := filepath.Join(logRoot, att.Name())
				caps, err := os.ReadDir(attDir)
				if err != nil {
					continue
				}
				bestAttempt := map[string]int{}
				for _, c := range caps {
					if !c.IsDir() {
						continue
					}
					scenario, attemptNum, ok := splitCaptureName(c.Name())
					if !ok {
						continue
					}
					capDir := filepath.Join(attDir, c.Name())
					if _, err := os.Stat(filepath.Join(capDir, "c-draws.log")); err != nil {
						continue
					}
					if attemptNum >= bestAttempt[scenario] {
						bestAttempt[scenario] = attemptNum
						out[pairKey{seed: sd.seed, scenario: scenario}] = capDir
					}
				}
			}
		}
	}
	return out, nil
}

func splitCaptureName(name string) (scenario string, attempt int, ok bool) {
	idx := strings.LastIndex(name, ".capture")
	if idx <= 0 {
		return "", 0, false
	}
	n, err := strconv.Atoi(name[idx+len(".capture"):])
	if err != nil || n < 1 {
		return "", 0, false
	}
	return name[:idx], n, true
}

func parseAndVerifyCapture(drawBytes []byte, sitesPath string, byRet map[uint64]DrawSite) ([]AttributedCall, string) {
	// #nosec G304 -- trusted local census evidence path
	f, err := os.Open(sitesPath)
	if err != nil {
		return nil, "missing_sites_log"
	}
	defer func() { _ = f.Close() }()

	drawSc := bufio.NewScanner(bytes.NewReader(drawBytes))
	siteSc := bufio.NewScanner(f)
	var calls []AttributedCall

	for {
		hasDraw := drawSc.Scan()
		hasSite := siteSc.Scan()
		if !hasDraw && !hasSite {
			break
		}
		if hasDraw != hasSite {
			return nil, "line_count_mismatch"
		}
		drawPrim, ok := drawLinePrimitive(drawSc.Text())
		if !ok {
			return nil, "malformed_draw_line"
		}
		retStr, sitePrim, ok := strings.Cut(siteSc.Text(), "\t")
		if !ok {
			return nil, "malformed_site_line"
		}
		if drawPrim != sitePrim {
			return nil, "primitive_mismatch"
		}
		retVaddr, err := strconv.ParseUint(strings.TrimPrefix(retStr, "0x"), 16, 64)
		if err != nil {
			return nil, "malformed_ret_vaddr"
		}
		site, ok := byRet[retVaddr]
		if !ok {
			return nil, fmt.Sprintf("unknown_ret_vaddr:0x%x", retVaddr)
		}
		if site.Primitive != sitePrim {
			return nil, "inventory_primitive_mismatch"
		}
		calls = append(calls, AttributedCall{
			Site:      site.Site,
			Primitive: site.Primitive,
		})
	}
	if err := drawSc.Err(); err != nil {
		return nil, "draw_scan_error"
	}
	if err := siteSc.Err(); err != nil {
		return nil, "site_scan_error"
	}
	return calls, ""
}

// drawLinePrimitive extracts "number", "dice", or "uniform" from a c-draws.log
// line of the form "DRAW <idx> <n> Number(a,b) = v".
func drawLinePrimitive(line string) (string, bool) {
	fields := strings.Fields(line)
	if len(fields) < 4 || fields[0] != "DRAW" {
		return "", false
	}
	name, _, ok := strings.Cut(fields[3], "(")
	if !ok {
		return "", false
	}
	switch name {
	case "Number":
		return "number", true
	case "Dice":
		return "dice", true
	case "Uniform":
		return "uniform", true
	default:
		return "", false
	}
}
