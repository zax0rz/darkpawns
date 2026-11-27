package oraclediff

import (
	"fmt"
	"strings"
	"testing"
)

func TestCompareProbeBlockVerdicts(t *testing.T) {
	scenario := Scenario{Name: "test-scenario"}

	// Agreeing pair (with ANSI escapes and CRLF that the census normalizer strips).
	agreeOracle := AudienceProbeBlock{
		Command:  "look",
		Audience: "actor",
		Output:   "\x1b[32mTemple Square\x1b[0m\r\n[ Exits: n s ]\r\n20H 100M 82V > ",
	}
	agreeGo := AudienceProbeBlock{
		Command:  "look",
		Audience: "actor",
		Output:   "Temple Square\n[ Exits: n s ]\n20H 100M 82V > ",
	}
	sameCmp := CompareProbeBlock(scenario, agreeOracle, agreeGo, true, nil)
	if sameCmp.Verdict != VerdictSame {
		t.Fatalf("agreeing pair verdict = %q, want %q (diff=%q)", sameCmp.Verdict, VerdictSame, sameCmp.Diff.Diff)
	}
	if sameCmp.Diff.Diff != "" {
		t.Fatalf("agreeing pair diff = %q, want empty", sameCmp.Diff.Diff)
	}

	// Differing pair without ledger entry -> diverged.
	diffGo := AudienceProbeBlock{
		Command:  "look",
		Audience: "actor",
		Output:   "Temple SquarX\n[ Exits: n s ]\n20H 100M 82V > ",
	}
	divCmp := CompareProbeBlock(scenario, agreeOracle, diffGo, true, nil)
	if divCmp.Verdict != VerdictDiverged {
		t.Fatalf("differing pair verdict = %q, want %q", divCmp.Verdict, VerdictDiverged)
	}
	if divCmp.Fingerprint == "" {
		t.Fatal("differing pair fingerprint is empty")
	}

	// Same differing pair WITH a matching ledger + pin -> expected.
	divTSV := "scenario\tmanifest\tcase_id\tstatus\tstability\n" +
		"test-scenario\ttest.tsv\ttest.case\texcluded\t\n" +
		"unstable-scenario\ttest.tsv\ttest.unstable\tblocked\trun-varying\n"
	pinsTSV := fmt.Sprintf("scenario\tlabel\tsha256\tcitations\ntest-scenario\tlook\t%s\ttest.tsv:test.case:excluded\n", divCmp.Fingerprint)
	ledger, err := ParseExpectedLedger(strings.NewReader(divTSV), strings.NewReader(pinsTSV))
	if err != nil {
		t.Fatalf("ParseExpectedLedger: %v", err)
	}
	expCmp := CompareProbeBlock(scenario, agreeOracle, diffGo, true, ledger)
	if expCmp.Verdict != VerdictExpected {
		t.Fatalf("pinned ledger pair verdict = %q, want %q", expCmp.Verdict, VerdictExpected)
	}

	// Differing SHAPE on the same baselined scenario -> diverged (pin rejects unpinned shape).
	otherDiffGo := AudienceProbeBlock{
		Command:  "look",
		Audience: "actor",
		Output:   "Temple SquarY\n[ Exits: n s ]\n20H 100M 82V > ",
	}
	unpinnedCmp := CompareProbeBlock(scenario, agreeOracle, otherDiffGo, true, ledger)
	if unpinnedCmp.Verdict != VerdictDiverged {
		t.Fatalf("unpinned shape on baselined scenario verdict = %q, want %q", unpinnedCmp.Verdict, VerdictDiverged)
	}

	// Run-varying baselined scenario -> expected regardless of shape.
	unstableScenario := Scenario{Name: "unstable-scenario"}
	unstableCmp := CompareProbeBlock(unstableScenario, agreeOracle, otherDiffGo, true, ledger)
	if unstableCmp.Verdict != VerdictExpected {
		t.Fatalf("run-varying ledger pair verdict = %q, want %q", unstableCmp.Verdict, VerdictExpected)
	}
}
