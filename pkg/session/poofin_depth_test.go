package session

import (
	"testing"
)

func TestCmdPoofinClearsAndPreservesCRawArgument(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Poofintest", LVL_IMPL, 1001)
	registerInWorld(t, actor)

	actor.player.PoofIn = "old message"
	if err := cmdPoofin(actor, nil); err != nil {
		t.Fatalf("cmdPoofin clear: %v", err)
	}
	if got, want := readMsgText(t, actor), "Okay.\r\n"; got != want {
		t.Fatalf("clear acknowledgement = %q, want %q", got, want)
	}
	if actor.player.PoofIn != "" {
		t.Fatalf("cleared poof-in message = %q, want empty", actor.player.PoofIn)
	}

	if err := executeCommandRaw(actor, "poofin", []string{"a", "swirl", "of", "arrival"}, true, "a  swirl of arrival"); err != nil {
		t.Fatalf("executeCommandRaw poofin: %v", err)
	}
	if got, want := readMsgText(t, actor), "Okay.\r\n"; got != want {
		t.Fatalf("set acknowledgement = %q, want %q", got, want)
	}
	if got, want := actor.player.PoofIn, "a  swirl of arrival"; got != want {
		t.Fatalf("stored poof-in message = %q, want %q", got, want)
	}
}
