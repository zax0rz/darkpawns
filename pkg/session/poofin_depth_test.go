package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestPoofinRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["poofin"]
	if !ok {
		t.Fatal("poofin command has no C gate")
	}
	if gate.MinLevel != LVL_IMMORT || gate.MinPosition != combat.PosDead {
		t.Fatalf("poofin gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_IMMORT, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("poofin")
	if !ok {
		t.Fatal("poofin command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("poofin registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

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
