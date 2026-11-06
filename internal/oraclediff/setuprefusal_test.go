package oraclediff

import (
	"strings"
	"testing"
	"time"
)

// TestRunWarmupRefusedStepFails is the DP-1456 revert proof: a warmup step
// answered with "Huh?!?" must fail the run, because both engines refusing
// identically is how a hollow scenario passed (the spellmatrix cluster and
// transfer-depth). Reverting checkWarmupRefusals makes this test fail.
func TestRunWarmupRefusedStepFails(t *testing.T) {
	conn := &scriptedConn{outputs: []string{
		"You send your thoughts to ItemBuilder.\r\n", // goto/teleport noise
		"Huh?!?\r\n",
	}}
	err := RunWarmup(conn, nil, []string{"teleport Builder 8162", "set player Builder level 35"}, time.Millisecond, WarmupPolicy{Engine: "c"})
	if err == nil {
		t.Fatal("warmup answered with Huh?!? must fail the run")
	}
	if !strings.Contains(err.Error(), `setup-refused: warmup step 2 "set player Builder level 35" engine=c: Huh?!?`) {
		t.Fatalf("error = %v", err)
	}
}

// TestRunWarmupAnsweredStepPasses is the other half of the gate: ordinary
// output must not trip the refusal check, or every scenario would fail.
func TestRunWarmupAnsweredStepPasses(t *testing.T) {
	conn := &scriptedConn{outputs: []string{"Okay changed.\r\n"}}
	err := RunWarmup(conn, nil, []string{"set player Builder level 35"}, time.Millisecond, WarmupPolicy{Engine: "c"})
	if err != nil {
		t.Fatalf("ordinary warmup output must pass: %v", err)
	}
}

// TestRunWarmupGoSideRefusalFails proves a Go-only refusal also fails: setup
// that diverged between the engines is a real finding, not noise.
func TestRunWarmupGoSideRefusalFails(t *testing.T) {
	conn := &scriptedConn{outputs: []string{"There is no such player.\r\n"}}
	err := RunWarmup(conn, nil, []string{"set player Nobody level 5"}, time.Millisecond, WarmupPolicy{Engine: "go"})
	if err == nil || !strings.Contains(err.Error(), "engine=go: There is no such player.") {
		t.Fatalf("go-side refusal error = %v", err)
	}
}

// TestRunWarmupRefusalInPeerSenderStep checks the sender of a send:<peer>
// step, not the audience: the peer's own output is where its refusal lands.
func TestRunWarmupRefusalInPeerSenderStep(t *testing.T) {
	primary := &scriptedConn{outputs: []string{"", ""}}
	builder := &scriptedConn{outputs: []string{"Huh?!?\r\n"}}
	steps := []string{"send:builder oedit 4399"}
	err := RunWarmup(primary, map[string]Conn{"builder": builder}, steps, time.Millisecond, WarmupPolicy{Engine: "c"})
	if err == nil || !strings.Contains(err.Error(), `warmup step 1 "send:builder oedit 4399" engine=c: Huh?!?`) {
		t.Fatalf("peer-sender refusal error = %v", err)
	}
}

// TestRunWarmupAllowSetupRefusalOptOut skips the gate when the scenario
// carries the fixture (the parser test below pins the reason requirement).
func TestRunWarmupAllowSetupRefusalOptOut(t *testing.T) {
	conn := &scriptedConn{outputs: []string{"Huh?!?\r\n"}}
	err := RunWarmup(conn, nil, []string{"set player Mage level 30"}, time.Millisecond, WarmupPolicy{Engine: "c", AllowSetupRefusal: true})
	if err != nil {
		t.Fatalf("allow-setup-refusal must skip the gate: %v", err)
	}
}

// TestParseScenarioAllowSetupRefusalRequiresReason pins decision 4 of DP-1456:
// the directive without the reason comment is a parse error.
func TestParseScenarioAllowSetupRefusalRequiresReason(t *testing.T) {
	_, err := ParseScenario("optout-noreason", strings.NewReader(
		"[fixture]\nallow-setup-refusal\n[probe]\nlook\n"))
	if err == nil || !strings.Contains(err.Error(), "requires a \"# setup-refusal-reason:\" comment") {
		t.Fatalf("expected missing-reason parse error, got %v", err)
	}

	sc, err := ParseScenario("optout", strings.NewReader(
		"[fixture]\nallow-setup-refusal\n[probe]\nlook\n# setup-refusal-reason: warmup deliberately trips the frozen refusal on both engines\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !sc.AllowSetupRefusal {
		t.Fatal("allow-setup-refusal with a reason must parse")
	}
	if sc.SetupRefusalReason != "warmup deliberately trips the frozen refusal on both engines" {
		t.Fatalf("reason = %q", sc.SetupRefusalReason)
	}
}

func TestParseScenarioEmptySetupRefusalReasonRejected(t *testing.T) {
	_, err := ParseScenario("optout-empty", strings.NewReader(
		"[fixture]\nallow-setup-refusal\n[probe]\nlook\n# setup-refusal-reason:\n"))
	if err == nil || !strings.Contains(err.Error(), "empty setup-refusal-reason") {
		t.Fatalf("expected empty-reason parse error, got %v", err)
	}
}

// TestMatchingSetupRefusalWholeLineOnly pins the "whole normalized lines,
// never substrings of ordinary text" rule: prose containing a refusal phrase
// must not match, and ANSI framing must be stripped before comparing.
func TestMatchingSetupRefusalWholeLineOnly(t *testing.T) {
	if line, _ := matchingSetupRefusal("The old wizard mutters Huh?!? under his breath.\r\n"); line != "" {
		t.Fatalf("substring inside ordinary prose matched: %q", line)
	}
	if line, _ := matchingSetupRefusal("He said \"Cast what?!?\" loudly\r\n"); line != "" {
		t.Fatalf("quoted refusal inside prose matched: %q", line)
	}
	line, _ := matchingSetupRefusal("\x1b[0m\r\nHuh?!?\r\n> \x1b[0m")
	if line != "Huh?!?" {
		t.Fatalf("whole-line refusal with ANSI/prompt framing = %q", line)
	}
	// The message format of decision 3 lands in results.tsv as FAIL, never
	// INFRA: it must not match the census worker's infra signatures.
	for _, r := range setupRefusalLines {
		if strings.Contains(r.line, ": EOF") || strings.Contains(r.line, "connection reset") ||
			strings.Contains(r.line, "connection closed") || strings.Contains(r.line, "address already in use") {
			t.Fatalf("refusal line %q matches a census infra signature", r.line)
		}
	}
}
