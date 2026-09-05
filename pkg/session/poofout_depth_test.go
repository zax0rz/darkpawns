package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestPoofoutRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["poofout"]
	if !ok {
		t.Fatal("poofout command has no C gate")
	}
	if gate.MinLevel != LVL_IMMORT || gate.MinPosition != combat.PosDead {
		t.Fatalf("poofout gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_IMMORT, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("poofout")
	if !ok {
		t.Fatal("poofout command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("poofout registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

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
