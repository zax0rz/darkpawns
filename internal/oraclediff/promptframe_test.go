package oraclediff

import (
	"strings"
	"testing"
)

// withMeasuredMode runs body with the PF-063 measurement gate forced on, the
// way DP_MEASURE_PROMPT_TOKENS=1 sets it in a harness process. The package's
// tests do not run in parallel, so save/restore is safe.
func withMeasuredMode(t *testing.T, body func()) {
	t.Helper()
	saved := measurePromptTokens
	measurePromptTokens = true
	defer func() { measurePromptTokens = saved }()
	body()
}

// normalizeMeasured is normalize() under the measurement gate.
func normalizeMeasured(t *testing.T, raw string) string {
	t.Helper()
	var got string
	withMeasuredMode(t, func() { got = normalize(raw, false) })
	return got
}

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

// TestMeasuredNormalizeAndReconcileEndToEnd drives the full refined pipeline
// on the two measured classes: a repaint-noise pair compares equal, a real
// blank-line pair does not. Wire shapes are the Step 0 raw bytes: message +
// terminator + blanks + vitals prompt vs message alone, and one-vs-two
// blanks before the prompt.
func TestMeasuredNormalizeAndReconcileEndToEnd(t *testing.T) {
	withMeasuredMode(t, func() {
		// Noise: C sends the trailing frame, Go's burst ends at the message.
		cBlock := normalizeMeasured(t, "You ask Barnaby, 'why'\r\n\r\n\r\n22H 100M 85V> ")
		gBlock := normalizeMeasured(t, "You ask Barnaby, 'why'\r\n")
		gotA, gotB := reconcilePromptFrameEdges(cBlock, gBlock)
		if gotA != gotB {
			t.Fatalf("repaint-noise pair still diverges: %q vs %q", gotA, gotB)
		}

		// Real: both sides prompt, Go carries one extra blank (compare block 0).
		cBlock = normalizeMeasured(t, "Looks like you don't have those objects..\n\r\r\n24H 100M 85V> ")
		gBlock = normalizeMeasured(t, "Looks like you don't have those objects..\r\n\r\n\r\n24H 100M 85V> ")
		gotA, gotB = reconcilePromptFrameEdges(cBlock, gBlock)
		if gotA == gotB {
			t.Fatal("a real blank-line divergence compared equal under the refined comparator")
		}
	})
}

// TestReconcilePromptFramesNoOpInProduction proves the shipping comparison is
// unchanged: without the measurement env, reconcile returns its inputs.
// Skipped only when the whole test binary runs in measurement mode, which
// production never does.
func TestReconcilePromptFramesNoOpInProduction(t *testing.T) {
	if measurePromptTokens {
		t.Skip("test binary itself runs in measurement mode")
	}
	a, b := "msg\n\n<PROMPT>\n", "msg\n"
	gotA, gotB := ReconcilePromptFrames(a, b)
	if gotA != a || gotB != b {
		t.Fatalf("production reconcile altered blocks: (%q, %q)", gotA, gotB)
	}
}

// TestMeasuredNormalizeKeepsPromptTokens proves the gate wiring in normalize:
// under measurement mode a prompt-only line survives as a <PROMPT> token
// instead of being deleted, and its vitals are masked. The production
// counterpart (prompt lines dropped) is asserted by the existing
// TestNormalizeTier1Rules, which fails if the gate ever leaks.
func TestMeasuredNormalizeKeepsPromptTokens(t *testing.T) {
	withMeasuredMode(t, func() {
		got := normalizeMeasured(t, "text\r\n22H 100M 85V> \r\nmore\r\n")
		if !strings.Contains(got, "<PROMPT>") {
			t.Fatalf("prompt line deleted under measurement mode: %q", got)
		}
		if strings.Contains(got, "22H") {
			t.Fatalf("vitals not masked: %q", got)
		}
	})
}
