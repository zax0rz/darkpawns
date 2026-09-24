package stringcensus

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// BaselineFile is the checked-in ratchet baseline, next to the generated
// reports. It records the go-only segments that existed when the census was
// installed, each with a reason a reviewer must eventually settle. It mirrors
// the website voice-lint baseline (website-astro/voice-lint-baseline.json): old
// debt is visible, new debt fails.
const BaselineFile = "go-only-baseline.json"

// Unreviewed is the reason a freshly recorded segment starts with.
const Unreviewed = "unreviewed"

// Reason vocabulary (brief 09). Every baseline row carries exactly one, plus an
// Evidence string a reviewer can check in a minute. The vocabulary exists so
// "why is this string allowed?" has an answer that outlives the triage PR.
const (
	// ReasonBugInvented: C prints nothing here, or something else (R4).
	ReasonBugInvented = "bug:invented"
	// ReasonBugParaphrase: C prints a similar but different string (R1).
	ReasonBugParaphrase = "bug:paraphrase"
	// ReasonCensusComposed: C prints the same bytes, built from pieces or a
	// table the census cannot see.
	ReasonCensusComposed = "census:composed"
	// ReasonData: the text comes from world data or a script.
	ReasonData = "data"
	// ReasonSurfaceNoC: a surface C does not have (web client, webOLC, admin
	// API, GMCP, TLS, Go server lifecycle). No telnet player can reach it.
	ReasonSurfaceNoC = "surface:no-c"
	// ReasonUnsure: the triager could not settle it.
	ReasonUnsure = "unsure"
)

// Reasons lists the vocabulary for validation.
func Reasons() []string {
	return []string{
		Unreviewed,
		ReasonBugInvented,
		ReasonBugParaphrase,
		ReasonCensusComposed,
		ReasonData,
		ReasonSurfaceNoC,
		ReasonUnsure,
	}
}

// Baseline is the ratchet's key set.
type Baseline struct {
	Description string                   `json:"description"`
	Generated   string                   `json:"generated,omitempty"`
	Segments    map[string]BaselineEntry `json:"segments"`
}

// BaselineEntry is one known go-only segment.
type BaselineEntry struct {
	Segment string `json:"segment"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Sink    string `json:"sink"`
	// Reason is one of the vocabulary above. It starts as Unreviewed; a triage
	// pass replaces it with the classification and the evidence.
	Reason string `json:"reason"`
	// Evidence is the checkable proof for Reason: for bug:*/census:composed a C
	// citation and the C text, for data the lib/ path, for surface:no-c which
	// surface. Empty while Reason is Unreviewed.
	Evidence string `json:"evidence,omitempty"`
	// FirstSeen is the date the segment entered the baseline.
	FirstSeen string `json:"firstSeen"`
}

// LoadBaseline reads a baseline file. A missing file is not an error: the
// census can still report, it just has nothing to ratchet against.
func LoadBaseline(path string) (*Baseline, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path comes from the census output directory
	if os.IsNotExist(err) {
		return &Baseline{Segments: map[string]BaselineEntry{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if b.Segments == nil {
		b.Segments = map[string]BaselineEntry{}
	}
	return &b, nil
}

// Check compares a report against the baseline. It returns the go-only segments
// that are not in the baseline (the ratchet failure) and the baseline entries
// no longer produced (stale entries, reported but not fatal, since a fix should
// not have to update the baseline to go green).
func (b *Baseline) Check(r *Report) (added []string, stale []string) {
	current := map[string]bool{}
	for _, text := range r.GoOnlyTexts() {
		current[text] = true
		if _, ok := b.Segments[text]; !ok {
			added = append(added, text)
		}
	}
	for text := range b.Segments {
		if !current[text] {
			stale = append(stale, text)
		}
	}
	sort.Strings(added)
	sort.Strings(stale)
	return added, stale
}

// Merge returns a new baseline covering the report, preserving the reason and
// firstSeen of every segment that was already recorded and refreshing its
// file/line/sink. today is the ISO date new entries are stamped with.
func (b *Baseline) Merge(r *Report, today string) *Baseline {
	out := &Baseline{
		Description: "Known Go player-facing strings with no C source and no world-data source. " +
			"make string-census fails when a new one appears; each reason starts unreviewed and is " +
			"settled by a human (data-driven text, transport framing, deliberate post-C addition, or a bug).",
		Generated: today,
		Segments:  map[string]BaselineEntry{},
	}
	for _, s := range r.GoOnlySegments {
		entry := BaselineEntry{Segment: s.Text, File: s.File, Line: s.Line, Sink: s.Sink}
		if prev, ok := b.Segments[s.Text]; ok {
			entry.Reason = prev.Reason
			entry.Evidence = prev.Evidence
			entry.FirstSeen = prev.FirstSeen
		}
		if entry.Reason == "" {
			entry.Reason = Unreviewed
		}
		if entry.FirstSeen == "" {
			entry.FirstSeen = today
		}
		out.Segments[s.Text] = entry
	}
	return out
}

// Save writes the baseline with sorted keys and a trailing newline, so a
// regenerated file diffs cleanly.
func (b *Baseline) Save(path string) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// UnreviewedCount counts entries still carrying the starting reason, for the
// census output.
func (b *Baseline) UnreviewedCount() int {
	n := 0
	for _, e := range b.Segments {
		if e.Reason == Unreviewed {
			n++
		}
	}
	return n
}
