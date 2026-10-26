package main

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

const fixedTimeScenario = `[fixture]
quiet-mobs

[setup:oracle]
Testor
testpassword
<ENTER>
1

[setup:port]
Testor
testpassword
<ENTER>
1

[probe]
time
`

// The fixture directive pins the boot calendar; the value must survive
// parsing as the unix timestamp the engines will receive.
func TestFixedTimeFixtureParses(t *testing.T) {
	body := strings.Replace(fixedTimeScenario, "quiet-mobs", "quiet-mobs\nfixed-time 650812932", 1)
	scenario, err := oraclediff.ParseScenario("fixed-time-probe", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if scenario.FixedTime != 650812932 {
		t.Fatalf("FixedTime = %d, want 650812932", scenario.FixedTime)
	}
}

// mud_time_passed counts mud hours from beginning_of_time (src/db.c:417), so
// a pin before it yields a negative calendar; garbage is rejected outright.
func TestFixedTimeRejectsInvalidValues(t *testing.T) {
	for _, bad := range []string{"not-a-number", "650336714", "-1", "0"} {
		body := strings.Replace(fixedTimeScenario, "quiet-mobs", "quiet-mobs\nfixed-time "+bad, 1)
		_, err := oraclediff.ParseScenario("fixed-time-bad", strings.NewReader(body))
		if err == nil {
			t.Fatalf("fixed-time %q parsed without error", bad)
		}
		if !strings.Contains(err.Error(), "not a unix timestamp") {
			t.Fatalf("fixed-time %q error = %v, want the timestamp message", bad, err)
		}
	}
}

// Default off: without the directive the harness default reaches both
// engines; with it, the pinned value does.
func TestFixedTimeFor(t *testing.T) {
	scenario, err := oraclediff.ParseScenario("fixed-time-default", strings.NewReader(fixedTimeScenario))
	if err != nil {
		t.Fatal(err)
	}
	if got := fixedTimeFor(&scenario); got != fixedTime {
		t.Fatalf("fixedTimeFor without fixture = %q, want the default %q", got, fixedTime)
	}
	scenario.FixedTime = 650812932
	if got := fixedTimeFor(&scenario); got != "650812932" {
		t.Fatalf("fixedTimeFor with fixture = %q, want 650812932", got)
	}
}
