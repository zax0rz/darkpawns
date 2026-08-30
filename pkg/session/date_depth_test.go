package session

import (
	"strings"
	"testing"
)

func TestCmdDateIgnoresArguments(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Dategod", LVL_IMMORT, 1001)

	if err := cmdDate(s, []string{"boot"}); err != nil {
		t.Fatalf("cmdDate: %v", err)
	}
	got := readSessionText(t, s)
	if !strings.HasPrefix(got, "Current machine time: ") {
		t.Fatalf("cmdDate(boot) = %q, want C SCMD_DATE output", got)
	}
	if strings.HasPrefix(got, "Up since ") {
		t.Fatalf("cmdDate(boot) invented uptime output: %q", got)
	}
}
