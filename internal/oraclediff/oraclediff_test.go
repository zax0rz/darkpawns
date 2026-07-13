package oraclediff

import (
	"strings"
	"testing"
)

func TestNormalizeTier1Rules(t *testing.T) {
	raw := "\x1b[31mRoom\x1b[0m\r\n" +
		"  Str: excellent     Dex: poor          Int: average   \r\n" +
		"  Wis: decent        Con: good          Cha: bad\r\n" +
		"22H 100M 83V >   \r\nPlayer HP: 22/22\r\nPlayers online: 7\r\n" +
		"As of 10-17-2008 there has been a pwipe.\r\n**** Dark Pawns 3.0 beta\r\nDescription  \r\n"
	want := "Room\n<ROLLED_STATS>\n<PROMPT>\nPlayer <VITALS>\nDescription\n"
	if got := Normalize(raw); got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func TestUnifiedDiff(t *testing.T) {
	diff := UnifiedDiff("c", "go", "same\nold\ntail\n", "same\nnew\ntail\n")
	for _, want := range []string{"--- c", "+++ go", "@@ -1,3 +1,3 @@", "-old", "+new"} {
		if !strings.Contains(diff, want) {
			t.Errorf("diff missing %q:\n%s", want, diff)
		}
	}
}

func TestParseScenarioEnter(t *testing.T) {
	scenario, err := ParseScenario("test", strings.NewReader("# comment\nname\n<ENTER>\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(scenario.Steps) != 2 || scenario.Steps[1] != "" {
		t.Fatalf("unexpected steps: %#v", scenario.Steps)
	}
}
