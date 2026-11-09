package oraclediff

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRefusalTemplatesBoundedWholeLines(t *testing.T) {
	for _, r := range setupRefusalPatterns {
		t.Run(r.template, func(t *testing.T) {
			// Suffixes cannot be swallowed by a leading substituted name/object.
			for _, negative := range []string{r.template + " unrelated prose", "", "\n", strings.Repeat("x", 1001)} {
				if r.pattern.MatchString(negative) {
					t.Fatalf("%q matched %s", negative, r.template)
				}
			}
		})
	}
	for _, tc := range []struct{ template, good, bad string }{
		{"You don't see %s %s here.", "You don't see a chalice here.", "You don't see  chalice here."},
		{"You can't wear $p.", "You can't wear a golden chalice.", "You can't wear ."},
		{"%d is the highest possible level.", "40 is the highest possible level.", "forty is the highest possible level."},
		{"$N must follow you to enter your group.", "Sleeper must follow you to enter your group.", " must follow you to enter your group."},
		{"literal (x).*? 100%% $$", "literal (x).*? 100% $", "literal xAAAA 100% $"},
	} {
		pattern := compileRefusalTemplate(tc.template)
		if !pattern.MatchString(tc.good) || pattern.MatchString(tc.bad) {
			t.Fatalf("template %q good=%q bad=%q", tc.template, tc.good, tc.bad)
		}
		if pattern.MatchString(tc.good + "\n") {
			t.Fatalf("newline accepted: %q", tc.template)
		}
	}
}

// The JSON contains verbatim normalized C lines and the retained dump/line
// they came from. Synthetic template substitutions are not C witnesses.
func TestSetupRefusalRetainedCWitnesses(t *testing.T) {
	data, err := os.ReadFile("../../docs/fidelity/evidence/dp-1511/refusal-witnesses.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Template, Cite, Line, Dump string
		DumpLine                   int `json:"dump_line"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("no retained C witnesses")
	}
	witnessed := make(map[string]string, len(cases))
	for _, tc := range cases {
		witnessed[tc.Template] = tc.Line
	}
	for _, r := range setupRefusalAdditionalLines {
		if _, ok := witnessed[r.line]; !ok {
			t.Errorf("new exact refusal lacks retained C witness: %s (%s)", r.line, r.cite)
		}
	}
	for _, r := range setupRefusalPatterns {
		positive, ok := witnessed[r.template]
		if !ok {
			t.Errorf("pattern lacks retained C witness: %s (%s)", r.template, r.cite)
			continue
		}
		if !r.pattern.MatchString(positive) {
			t.Errorf("pattern misses C witness %q", positive)
		}
		if r.pattern.MatchString(positive + " unrelated prose") {
			t.Errorf("pattern matches refusal embedded in prose: %s", r.template)
		}
	}
	for _, tc := range cases {
		t.Run(tc.Template, func(t *testing.T) {
			line, cite := matchingSetupRefusal(tc.Line)
			if line != tc.Line || cite == "" {
				t.Fatalf("C witness %s:%d: %q returned %q (%s)", tc.Dump, tc.DumpLine, tc.Line, line, cite)
			}
			for _, r := range setupRefusalPatterns {
				if r.template == tc.Template && !r.pattern.MatchString(tc.Line) {
					t.Fatalf("pattern misses its witness %q", tc.Line)
				}
			}
		})
	}
}

func TestRefusalTemplateFieldBoundsAndPronouns(t *testing.T) {
	p := compileRefusalTemplate("You can't wear $p.")
	if !p.MatchString("You can't wear "+strings.Repeat("x", 8192)+".") || p.MatchString("You can't wear "+strings.Repeat("x", 8193)+".") {
		t.Fatal("C MAX_STRING_LENGTH field bound not enforced")
	}
	p = compileRefusalTemplate("$E can't carry that much weight.")
	if !p.MatchString("He can't carry that much weight.") || p.MatchString("boat: You can't carry that much weight.") {
		t.Fatal("act pronoun must be a C pronoun")
	}
}

func TestRunWarmupParameterizedRefusalFails(t *testing.T) {
	conn := &scriptedConn{outputs: []string{"You don't see a chalice here.\r\n"}}
	err := RunWarmup(conn, nil, []string{"get chalice"}, 0, WarmupPolicy{Engine: "c"})
	if err == nil || !strings.Contains(err.Error(), "setup-refused") {
		t.Fatalf("DP-1510 hollow warmup error = %v", err)
	}
}

func TestWarmupRetainsRefusedAndAllowedOutput(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		conn := &scriptedConn{outputs: []string{"You don't see a chalice here.\r\n"}}
		captured := false
		err := RunWarmup(conn, nil, []string{"get chalice"}, 0, WarmupPolicy{Engine: "c", AllowSetupRefusal: allowed, Capture: func(blocks []AudienceProbeBlock) error {
			captured = true
			if len(blocks) != 1 || !strings.Contains(blocks[0].Output, "You don't see a chalice here.") {
				t.Fatalf("lost refused block: %#v", blocks)
			}
			return nil
		}})
		if !captured || (err == nil) != allowed {
			t.Fatalf("allowed=%v captured=%v err=%v", allowed, captured, err)
		}
	}
}
