package stringcensus

import (
	"os"
	"path/filepath"
	"testing"
)

func seg(text, file, sink string) Segment {
	return Segment{Text: text, File: file, Line: 1, Sink: sink}
}

func reportWithGoOnly(segs ...Segment) *Report {
	return &Report{
		GoOnlySegments: segs,
		GoOnly:         len(segs),
		goOnlyByFile:   countByFile(segs),
	}
}

func TestBaselineCheckReportsNewSegments(t *testing.T) {
	base := &Baseline{Segments: map[string]BaselineEntry{
		"a known segment": {Segment: "a known segment", Reason: Unreviewed},
	}}
	added, stale := base.Check(reportWithGoOnly(
		seg("a known segment", "pkg/game/a.go", ".Send"),
		seg("a new segment here", "pkg/game/b.go", ".Send"),
	))
	if len(added) != 1 || added[0] != "a new segment here" {
		t.Fatalf("added = %v, want the new segment", added)
	}
	if len(stale) != 0 {
		t.Fatalf("stale = %v, want none", stale)
	}
}

func TestBaselineCheckReportsStaleEntries(t *testing.T) {
	base := &Baseline{Segments: map[string]BaselineEntry{
		"a fixed segment": {Segment: "a fixed segment"},
	}}
	added, stale := base.Check(reportWithGoOnly())
	if len(added) != 0 {
		t.Fatalf("added = %v, want none", added)
	}
	if len(stale) != 1 || stale[0] != "a fixed segment" {
		t.Fatalf("stale = %v, want the fixed segment", stale)
	}
}

func TestBaselineMergePreservesReasons(t *testing.T) {
	base := &Baseline{Segments: map[string]BaselineEntry{
		"a reviewed segment": {Segment: "a reviewed segment", Reason: "data-driven", FirstSeen: "2026-09-01"},
	}}
	merged := base.Merge(reportWithGoOnly(
		seg("a reviewed segment", "pkg/game/a.go", ".Send"),
		seg("a brand new segment", "pkg/game/c.go", ".SendMessage"),
	), "2026-09-23")

	if got := merged.Segments["a reviewed segment"]; got.Reason != "data-driven" || got.FirstSeen != "2026-09-01" {
		t.Fatalf("reviewed entry = %+v, want reason and firstSeen preserved", got)
	}
	if got := merged.Segments["a brand new segment"]; got.Reason != Unreviewed || got.FirstSeen != "2026-09-23" {
		t.Fatalf("new entry = %+v, want unreviewed stamped today", got)
	}
	if merged.Generated != "2026-09-23" {
		t.Fatalf("generated = %q", merged.Generated)
	}
}

func TestBaselineMergePreservesTriage(t *testing.T) {
	base := &Baseline{Segments: map[string]BaselineEntry{
		"a triaged segment": {
			Segment:   "a triaged segment",
			Reason:    ReasonBugParaphrase,
			Evidence:  `src/act.other.c:1363 "You will no longer loot corpses.\r\n"`,
			FirstSeen: "2026-09-23",
		},
	}}
	merged := base.Merge(reportWithGoOnly(
		seg("a triaged segment", "pkg/game/a.go", ".Send"),
		seg("a fresh segment", "pkg/game/d.go", ".Send"),
	), "2026-09-24")

	// A regeneration must not throw away the triage: reason and evidence survive
	// for a segment that is still go-only.
	if got := merged.Segments["a triaged segment"]; got.Reason != ReasonBugParaphrase || got.Evidence == "" {
		t.Fatalf("triaged entry = %+v, want reason and evidence preserved", got)
	}
	if got := merged.Segments["a fresh segment"]; got.Reason != Unreviewed || got.Evidence != "" {
		t.Fatalf("fresh entry = %+v, want unreviewed with no evidence", got)
	}
}

func TestBaselineSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, BaselineFile)
	want := (&Baseline{Generated: "2026-09-23", Segments: map[string]BaselineEntry{
		"a segment": {Segment: "a segment", File: "pkg/game/a.go", Line: 7, Sink: ".Send", Reason: Unreviewed, FirstSeen: "2026-09-23"},
	}})
	if err := want.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := LoadBaseline(path)
	if err != nil {
		t.Fatal(err)
	}
	entry := got.Segments["a segment"]
	if entry.Line != 7 || entry.Reason != Unreviewed {
		t.Fatalf("round trip = %+v", entry)
	}
	if n := got.UnreviewedCount(); n != 1 {
		t.Fatalf("unreviewed = %d, want 1", n)
	}
}

func TestLoadBaselineMissingFileIsEmpty(t *testing.T) {
	base, err := LoadBaseline(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("missing baseline should not error: %v", err)
	}
	if len(base.Segments) != 0 {
		t.Fatalf("segments = %v, want empty", base.Segments)
	}
}

func TestLoadBaselineRejectsMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), BaselineFile)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBaseline(path); err == nil {
		t.Fatal("malformed baseline should be reported, not ignored")
	}
}
