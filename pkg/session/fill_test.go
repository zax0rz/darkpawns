package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestFillRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["fill"]
	if !ok {
		t.Fatal("fill command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
		t.Fatalf("fill gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosStanding)
	}
}

func TestFillRestingActorHitsCPositionGate(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Fillactor", 1, 1001)
	actor.player.SetPosition(combat.PosResting)
	registerInWorld(t, actor)

	if err := ExecuteCommand(actor, "fill", nil); err != nil {
		t.Fatalf("ExecuteCommand returned error: %v", err)
	}
	if got := readMsgText(t, actor); got != "Nah... You feel too relaxed to do that..\r\n" {
		t.Fatalf("message = %q", got)
	}
}
