package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestSplitRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["split"]
	if !ok {
		t.Fatal("split command has no C gate")
	}
	if entry.MinLevel != 1 || entry.MinPosition != combat.PosSitting {
		t.Fatalf("split gate = level %d position %d, want level 1 position %d", entry.MinLevel, entry.MinPosition, combat.PosSitting)
	}
	registered, ok := cmdRegistry.Lookup("split")
	if !ok {
		t.Fatal("split command is not registered")
	}
	if registered.MinLevel != entry.MinLevel || registered.MinPosition != entry.MinPosition {
		t.Fatalf("split registry gate = level %d position %d, want level %d position %d", registered.MinLevel, registered.MinPosition, entry.MinLevel, entry.MinPosition)
	}
}

// TestSplitNoGroupUsesCMessage proves do_split's direct no-group refusal
// (src/act.other.c:863-866) without relying on the shared group/ungroup
// command vehicle. The command is still invoked through its registered C
// entry gate and the exported Go bridge.
func TestSplitNoGroupUsesCMessage(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Splitnogroup", 1, 1001)
	actor.player.SetPosition(combat.PosSitting)
	actor.player.SetGold(10)
	registerInWorld(t, actor)

	if err := ExecuteCommand(actor, "split", []string{"10"}); err != nil {
		t.Fatalf("ExecuteCommand returned error: %v", err)
	}
	if got, want := readMsgText(t, actor), "With whom do you wish to share your gold?\r\n"; got != want {
		t.Fatalf("no-group response = %q, want %q", got, want)
	}
}
