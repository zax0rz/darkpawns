package session

import (
	"testing"
)

func TestCmdPoofoutClearsAndPreservesCRawArgument(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Poofouttest", LVL_IMPL, 1001)
	registerInWorld(t, actor)

	actor.player.PoofOut = "old message"
	if err := cmdPoofout(actor, nil); err != nil {
		t.Fatalf("cmdPoofout clear: %v", err)
	}
	if got, want := readMsgText(t, actor), "Okay.\r\n"; got != want {
		t.Fatalf("clear acknowledgement = %q, want %q", got, want)
	}
	if actor.player.PoofOut != "" {
		t.Fatalf("cleared poof-out message = %q, want empty", actor.player.PoofOut)
	}

	if err := executeCommandRaw(actor, "poofout", []string{"a", "swirl", "of", "departure"}, true, "a  swirl of departure"); err != nil {
		t.Fatalf("executeCommandRaw poofout: %v", err)
	}
	if got, want := readMsgText(t, actor), "Okay.\r\n"; got != want {
		t.Fatalf("set acknowledgement = %q, want %q", got, want)
	}
	if got, want := actor.player.PoofOut, "a  swirl of departure"; got != want {
		t.Fatalf("stored poof-out message = %q, want %q", got, want)
	}
}
