package oraclediff

import "strings"

// ReconcilePromptFrames is the two-sided half of the PF-063 refined
// comparator. The per-side canonicalization keeps <PROMPT> tokens and the
// blank lines before them; this pass then tolerates the one shape that no
// per-side rule can: a block-edge prompt frame that exists on one engine
// only. Step 0 raw bytes (2026-10-11/pf063-step0-raw) showed the 16
// prompt-repaint noise scenarios are exactly that — a leading bare
// <PROMPT> on one side, or a trailing blank+<PROMPT> frame on one side —
// while the 27 real divergences carry the frame on both sides and differ
// only in its blank count. So: when one side's leading (or trailing)
// framing run holds a prompt and the other side's does not, the run is
// transport timing and is stripped from the side that has it. When both
// sides hold the prompt, both runs stand and their blank counts stay
// comparable. Interior framing runs are never touched.
//
// It is a no-op unless DP_MEASURE_PROMPT_TOKENS=1: the shipping census
// comparison is byte-for-byte unchanged.
func ReconcilePromptFrames(oracle, goPort string) (string, string) {
	if !measurePromptTokens {
		return oracle, goPort
	}
	return reconcilePromptFrameEdges(oracle, goPort)
}

func reconcilePromptFrameEdges(a, b string) (string, string) {
	la := splitNormalizedLines(a)
	lb := splitNormalizedLines(b)

	// Trailing edge first: post-canonicalization a framing run that holds a
	// prompt ends with that single <PROMPT>, so "block ends with a prompt"
	// is the run's presence test. Stripping removes the run's blanks too —
	// the other side ends its burst at the message, so its own terminator
	// bytes are not being compared here.
	if endsWithPrompt(la) != endsWithPrompt(lb) {
		la = stripTrailingFraming(la)
		lb = stripTrailingFraming(lb)
	}
	// Leading edge: normalization trims leading blanks, so a leading framing
	// run is a single prompt-only line at most.
	if startsWithPrompt(la) != startsWithPrompt(lb) {
		if startsWithPrompt(la) {
			la = la[1:]
		}
		if startsWithPrompt(lb) {
			lb = lb[1:]
		}
	}
	return strings.Join(la, "\n") + "\n", strings.Join(lb, "\n") + "\n"
}

// splitNormalizedLines splits a normalize() product back into lines without
// its terminal newline.
func splitNormalizedLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func startsWithPrompt(lines []string) bool {
	return len(lines) > 0 && promptOnly.MatchString(lines[0])
}

func endsWithPrompt(lines []string) bool {
	return len(lines) > 0 && promptOnly.MatchString(lines[len(lines)-1])
}

func stripTrailingFraming(lines []string) []string {
	end := len(lines)
	for end > 0 && promptFramingLine(lines[end-1]) {
		end--
	}
	return lines[:end]
}
