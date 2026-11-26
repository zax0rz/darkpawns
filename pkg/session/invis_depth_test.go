package session

import (
	"testing"
)

func TestInvisPromptIncludesCLevel(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Invprompt", LVL_IMMORT, 1001)
	s.player.SetInvisLevel(31)

	if got := s.promptText(); got != "i31 > " {
		t.Fatalf("invis prompt = %q, want %q", got, "i31 > ")
	}
}
