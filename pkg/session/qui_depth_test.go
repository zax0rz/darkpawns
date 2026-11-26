package session

import (
	"strings"
	"testing"
)

func TestQuiMortalShortcutUsesCRefusal(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "QuiMortal", 1, 1001)

	if err := cmdQuiStub(s); err != nil {
		t.Fatalf("cmdQuiStub: %v", err)
	}
	if got := drainSendChannel(t, s); !strings.Contains(got, "You have to type quit--no less, to quit!") {
		t.Fatalf("qui output = %q, want C refusal", got)
	}
}
