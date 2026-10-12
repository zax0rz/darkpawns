package oraclediff

import (
	"strings"
	"testing"
)

// TestCanonicalizePromptRunsCollapsesRepaintFraming proves the per-side half
// of the refined comparator: everything from a framing run's first prompt to
// the run's end collapses to one token, while the blanks before that prompt
// survive (their count is the real R1 signal) and prompt-less blank runs pass
// through untouched as message bytes.
func TestCanonicalizePromptRunsCollapsesRepaintFraming(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "repaint with blank between prompts collapses, pre-prompt blank kept",
			in:   []string{"msg", "", "<PROMPT>", "", "<PROMPT>", "msg2"},
			want: []string{"msg", "", "<PROMPT>", "msg2"},
		},
		{
			name: "consecutive repaints with no blank collapse",
			in:   []string{"msg", "", "<PROMPT>", "<PROMPT>", "msg2"},
			want: []string{"msg", "", "<PROMPT>", "msg2"},
		},
		{
			name: "blank run without a prompt is message bytes",
			in:   []string{"msg", "", "", "msg2"},
			want: []string{"msg", "", "", "msg2"},
		},
		{
			name: "single prompt run already canonical",
			in:   []string{"msg", "", "", "<PROMPT>"},
			want: []string{"msg", "", "", "<PROMPT>"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := canonicalizePromptRuns(tc.in)
			if strings.Join(got, "\x00") != strings.Join(tc.want, "\x00") {
				t.Fatalf("canonicalizePromptRuns(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestReconcilePromptFrameEdgesStripsOneSidedFraming proves the two-sided
// half: a block-edge prompt frame that exists on one side only is transport
// timing and is stripped, in either direction, leading or trailing. These are
// the exact shapes of the 16 noise scenarios (pf063-step0-raw / classify.tsv).
func TestReconcilePromptFrameEdgesStripsOneSidedFraming(t *testing.T) {
	cases := []struct {
		name         string
		a, b         string
		wantA, wantB string
	}{
		{
			name:  "C-only trailing blank+prompt frame",
			a:     "msg\n\n<PROMPT>\n",
			b:     "msg\n",
			wantA: "msg\n",
			wantB: "msg\n",
		},
		{
			name:  "Go-only trailing blank+prompt frame",
			a:     "msg\n",
			b:     "msg\n\n<PROMPT>\n",
			wantA: "msg\n",
			wantB: "msg\n",
		},
		{
			name:  "Go-only leading bare prompt",
			a:     "msg\n",
			b:     "<PROMPT>\nmsg\n",
			wantA: "msg\n",
			wantB: "msg\n",
		},
		{
			name:  "Go-only leading bare prompt after message burst",
			a:     "msg1\nmsg2\n",
			b:     "<PROMPT>\nmsg1\nmsg2\n",
			wantA: "msg1\nmsg2\n",
			wantB: "msg1\nmsg2\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotA, gotB := reconcilePromptFrameEdges(tc.a, tc.b)
			if gotA != tc.wantA || gotB != tc.wantB {
				t.Fatalf("reconcilePromptFrameEdges(%q, %q) = (%q, %q), want (%q, %q)",
					tc.a, tc.b, gotA, gotB, tc.wantA, tc.wantB)
			}
		})
	}
}

// TestReconcilePromptFrameEdgesKeepsTwoSidedBlankCounts proves the measuring
// stick can go red: when both sides carry the prompt frame, the pre-prompt
// blank counts stay comparable, so a one-blank divergence (the #1880 class:
// compare, spec-proc-guild-guard, comm-depth) survives reconciliation.
func TestReconcilePromptFrameEdgesKeepsTwoSidedBlankCounts(t *testing.T) {
	cOracle := "You whisper.\n\n<PROMPT>\n"
	goPort := "You whisper.\n\n\n<PROMPT>\n"
	gotA, gotB := reconcilePromptFrameEdges(cOracle, goPort)
	if gotA != cOracle || gotB != goPort {
		t.Fatalf("two-sided frames were altered: (%q, %q), want unchanged", gotA, gotB)
	}
	if gotA == gotB {
		t.Fatal("a real one-blank divergence compared equal")
	}
}

// TestNormalizeAndReconcileEndToEnd drives the full adopted pipeline on the
// two PF-063 classes: a repaint-noise pair compares equal, a real blank-line
// pair does not. Wire shapes are the Step 0 raw bytes: message + terminator +
// blanks + vitals prompt vs message alone, and one-vs-two blanks before the
// prompt.
func TestNormalizeAndReconcileEndToEnd(t *testing.T) {
	// Noise: C sends the trailing frame, Go's burst ends at the message.
	cBlock := normalizeRaw(t, "You ask Barnaby, 'why'\r\n\r\n\r\n22H 100M 85V> ")
	gBlock := normalizeRaw(t, "You ask Barnaby, 'why'\r\n")
	gotA, gotB := reconcilePromptFrameEdges(cBlock, gBlock)
	if gotA != gotB {
		t.Fatalf("repaint-noise pair still diverges: %q vs %q", gotA, gotB)
	}

	// Real: both sides prompt, Go carries one extra blank (compare block 0).
	cBlock = normalizeRaw(t, "Looks like you don't have those objects..\n\r\r\n24H 100M 85V> ")
	gBlock = normalizeRaw(t, "Looks like you don't have those objects..\r\n\r\n\r\n24H 100M 85V> ")
	gotA, gotB = reconcilePromptFrameEdges(cBlock, gBlock)
	if gotA == gotB {
		t.Fatal("a real blank-line divergence compared equal under the adopted comparator")
	}
}

// TestReconcilePromptFramesDocumentedBlindSpot states the adopted
// comparator's one deliberate hole in executable form: a block-edge prompt
// frame present on one engine only is transport timing and is stripped — so
// an engine that never prompts at a block edge is indistinguishable from
// one that does. No worse than the deletion rule this comparator replaced;
// prompt presence is proven by keep-prompts/no-settle scenarios (R5f).
func TestReconcilePromptFramesDocumentedBlindSpot(t *testing.T) {
	prompted := "msg\n\n<PROMPT>\n"
	neverPrompted := "msg\n"
	gotA, gotB := ReconcilePromptFrames(prompted, neverPrompted)
	if gotA != gotB {
		t.Fatalf("one-sided edge frame survived reconciliation: %q vs %q", gotA, gotB)
	}
}

// TestNormalizeKeepsPromptTokens proves the adopted default: a prompt-only
// line survives normalization as a <PROMPT> token with its vitals masked.
func TestNormalizeKeepsPromptTokens(t *testing.T) {
	got := normalizeRaw(t, "text\r\n22H 100M 85V> \r\nmore\r\n")
	if !strings.Contains(got, "<PROMPT>") {
		t.Fatalf("prompt line deleted by the adopted comparator: %q", got)
	}
	if strings.Contains(got, "22H") {
		t.Fatalf("vitals not masked: %q", got)
	}
}

// normalizeRaw is normalize with the ordinary (ANSI-stripped) mode, for
// tests that drive the adopted pipeline directly.
func normalizeRaw(t *testing.T, raw string) string {
	t.Helper()
	return normalize(raw, false)
}
