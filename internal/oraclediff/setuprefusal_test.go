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

// TestParseScenarioProbeActorMixRejected pins the DP-1456 follow-up: mixing a
// plain [probe] with a [probe:X] section is a parse error, because the parser
// keeps one ProbeActor and the later section silently retargets the earlier
// steps — the trap that hollowed transfer-depth. Reverting the guard in
// ParseScenario makes both order cases fail this test.
func TestParseScenarioProbeActorMixRejected(t *testing.T) {
	for _, tc := range []struct {
		name, body string
	}{
		{"plain then actor", "[setup:oracle:peer]\npeer\n[probe]\nlook\n[probe:peer]\npeek\n"},
		{"actor then plain", "[setup:oracle:peer]\npeer\n[probe:peer]\npeek\n[probe]\nlook\n"},
	} {
		_, err := ParseScenario(tc.name, strings.NewReader(tc.body))
		if err == nil || !strings.Contains(err.Error(), "would silently retarget") {
			t.Fatalf("%s: expected retarget parse error, got %v", tc.name, err)
		}
	}
	// Single-actor and repeated same-actor sections stay legal.
	for _, tc := range []struct {
		name, body string
	}{
		{"actor only", "[setup:oracle:peer]\npeer\n[probe:peer]\npeek\n"},
		{"same actor twice", "[setup:oracle:peer]\npeer\n[probe:peer]\npeek\n[probe:peer]\nlook\n"},
	} {
		if _, err := ParseScenario(tc.name, strings.NewReader(tc.body)); err != nil {
			t.Fatalf("%s: unexpected error %v", tc.name, err)
		}
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

func TestCheckProbeSetupRefusals(t *testing.T) {
	// Reproduces the DP-1597 failure shape (level-30 actor running load obj 4399
	// inside [probe]) and also verifies full DP-1456/DP-1511 inventory matching
	// beyond Huh?!?.
	blocks := []AudienceProbeBlock{
		{Command: "load obj 4399", Audience: "actor", Output: "Huh?!?\r\n"},
		{Command: "load obj 4399", Audience: "builder", Output: ""},
		{Command: "send:builder load obj 4399", Audience: "builder", Output: "You create some northern devil ginseng powder.\r\n"},
		{Command: "send:builder load obj 4399", Audience: "primary", Output: "ItemBuilder has created some northern devil ginseng powder!\r\n"},
		{Command: "goto 999999", Audience: "actor", Output: "No room exists with that number.\r\n"},
		{Command: "goto 999999", Audience: "builder", Output: ""},
		{Command: "quaff powder", Audience: "actor", Output: "You quaff some northern devil ginseng powder.\r\n"},
		{Command: "quaff powder", Audience: "builder", Output: ""},
	}
	hits := CheckProbeSetupRefusals(blocks)
	if len(hits) != 2 {
		t.Fatalf("got %d hits, want 2: %+v", len(hits), hits)
	}
	if hits[0].Step != "load obj 4399" || hits[0].Line != "Huh?!?" {
		t.Fatalf("hit[0] = %+v, want load obj 4399 / Huh?!?", hits[0])
	}
	if hits[1].Step != "goto 999999" || hits[1].Line != "No room exists with that number." {
		t.Fatalf("hit[1] = %+v, want goto 999999 / No room exists with that number.", hits[1])
	}

	// Clean builder-loaded probe stays quiet.
	clean := []AudienceProbeBlock{
		{Command: "send:builder load obj 4399", Audience: "builder", Output: "You create some northern devil ginseng powder.\r\n"},
		{Command: "send:builder load obj 4399", Audience: "primary", Output: ""},
		{Command: "send:builder give powder ItemMage", Audience: "builder", Output: "Ok.\r\n"},
		{Command: "send:builder give powder ItemMage", Audience: "primary", Output: ""},
	}
	if got := CheckProbeSetupRefusals(clean); len(got) != 0 {
		t.Fatalf("clean builder-loaded probe flagged: %+v", got)
	}

	// Refused in-probe give (DP-1599: fixed scenarios depend on send:builder give).
	giveRefused := []AudienceProbeBlock{
		{Command: "send:builder give powder ItemMage", Audience: "builder", Output: "You don't seem to have a powder.\r\n"},
		{Command: "send:builder give powder ItemMage", Audience: "primary", Output: ""},
		{Command: "give bread Stealactor", Audience: "actor", Output: "Stealactor seems to have his hands full.\r\n"},
		{Command: "give bread Stealactor", Audience: "peer", Output: ""},
	}
	giveHits := CheckProbeSetupRefusals(giveRefused)
	if len(giveHits) != 2 {
		t.Fatalf("got %d give hits, want 2: %+v", len(giveHits), giveHits)
	}
	if giveHits[0].Step != "send:builder give powder ItemMage" || giveHits[0].Line != "You don't seem to have a powder." {
		t.Fatalf("giveHits[0] = %+v", giveHits[0])
	}
	if giveHits[1].Step != "give bread Stealactor" || giveHits[1].Line != "Stealactor seems to have his hands full." {
		t.Fatalf("giveHits[1] = %+v", giveHits[1])
	}
}

func TestParseScenarioProbeSetupRefusalReason(t *testing.T) {
	sc, err := ParseScenario("opted-out", strings.NewReader(
		"# probe-setup-refusal-reason: \"load obj 4399\": intentionally tests level-30 load refusal\n"+
			"# probe-setup-refusal-reason: \"send:low transfer all\": tests LVL_GRGOD gate\n"+
			"[probe]\nload obj 4399\nsend:low transfer all\n"))
	if err != nil {
		t.Fatalf("valid probe-setup-refusal-reason failed to parse: %v", err)
	}
	if got := sc.ProbeSetupRefusalReasons["load obj 4399"]; got != "intentionally tests level-30 load refusal" {
		t.Fatalf("ProbeSetupRefusalReasons[load obj 4399] = %q", got)
	}
	if got := sc.ProbeSetupRefusalReasons["send:low transfer all"]; got != "tests LVL_GRGOD gate" {
		t.Fatalf("ProbeSetupRefusalReasons[send:low transfer all] = %q", got)
	}

	// Named step passes; unnamed refusal in the same file still fails.
	blocks := []AudienceProbeBlock{
		{Command: "load obj 4399", Audience: "actor", Output: "Huh?!?\r\n"},
		{Command: "send:low transfer all", Audience: "low", Output: "You are not godly enough to do that!\r\n"},
	}
	if hits := CheckProbeSetupRefusalsForScenario(blocks, sc.ProbeSetupRefusalReasons); len(hits) != 0 {
		t.Fatalf("named steps should pass, got %+v", hits)
	}
	blocksWithStray := []AudienceProbeBlock{
		{Command: "load obj 4399", Audience: "actor", Output: "Huh?!?\r\n"},
		{Command: "send:low transfer all", Audience: "low", Output: "You are not godly enough to do that!\r\n"},
		{Command: "load obj 3000", Audience: "actor", Output: "Huh?!?\r\n"},
	}
	hits := CheckProbeSetupRefusalsForScenario(blocksWithStray, sc.ProbeSetupRefusalReasons)
	if len(hits) != 1 || hits[0].Step != "load obj 3000" {
		t.Fatalf("unnamed refusal in same file must fail, got %+v", hits)
	}

	// Whitespace inside the quoted step is trimmed on both the reason key and
	// the compared step, so a sloppily padded step name still opts out.
	ws, err := ParseScenario("padded-step", strings.NewReader(
		"# probe-setup-refusal-reason: \"  load obj 4399  \": padded key\n[probe]\nload obj 4399\n"))
	if err != nil {
		t.Fatalf("padded step key failed to parse: %v", err)
	}
	if hits := CheckProbeSetupRefusalsForScenario(blocks, ws.ProbeSetupRefusalReasons); len(hits) != 0 {
		t.Fatalf("padded named step should pass, got %+v", hits)
	}

	// Reasons may contain colons; only the first colon after the quoted step
	// is the separator (real reasons cite C lines like src/x.c:1498).
	colon, err := ParseScenario("colon-reason", strings.NewReader(
		"# probe-setup-refusal-reason: \"load obj 4399\": witnesses do_load (src/act.wizard.c:1274)\n[probe]\nload obj 4399\n"))
	if err != nil {
		t.Fatalf("colon-bearing reason failed to parse: %v", err)
	}
	if got := colon.ProbeSetupRefusalReasons["load obj 4399"]; got != "witnesses do_load (src/act.wizard.c:1274)" {
		t.Fatalf("colon-bearing reason = %q", got)
	}

	// Empty reason fails (both bare and after a quoted step).
	for _, tc := range []struct {
		name string
		src  string
		want string
	}{
		{
			name: "bare-empty",
			src:  "# probe-setup-refusal-reason:\n[probe]\nload obj 4399\n",
			want: "empty probe-setup-refusal-reason",
		},
		{
			name: "empty-reason-after-step",
			src:  "# probe-setup-refusal-reason: \"load obj 4399\":\n[probe]\nload obj 4399\n",
			want: "empty probe-setup-refusal-reason",
		},
		{
			name: "empty-step",
			src:  "# probe-setup-refusal-reason: \"\": some reason\n[probe]\nload obj 4399\n",
			want: "empty step in probe-setup-refusal-reason",
		},
		{
			name: "old-whole-file-form",
			src:  "# probe-setup-refusal-reason: intentionally tests level-30 load refusal\n[probe]\nload obj 4399\n",
			want: "must name step",
		},
		{
			name: "old-whole-file-form-with-colon",
			src:  "# probe-setup-refusal-reason: Note: tests level-30 load refusal\n[probe]\nload obj 4399\n",
			want: "must name step",
		},
		{
			name: "missing-colon-separator",
			src:  "# probe-setup-refusal-reason: \"load obj 4399\" missing colon\n[probe]\nload obj 4399\n",
			want: `expected ": <why>"`,
		},
		{
			name: "duplicate-step",
			src: "# probe-setup-refusal-reason: \"load obj 4399\": first\n" +
				"# probe-setup-refusal-reason: \"load obj 4399\": second\n[probe]\nload obj 4399\n",
			want: "duplicate probe-setup-refusal-reason",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseScenario(tc.name, strings.NewReader(tc.src))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ParseScenario(%s) err = %v, want substring %q", tc.name, err, tc.want)
			}
		})
	}
}
