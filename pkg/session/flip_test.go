package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestFlipRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["flip"]
	if !ok {
		t.Fatal("flip command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
		t.Fatalf("flip gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosStanding)
	}
}

func TestFlipRestingActorHitsCPositionGate(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Flipgate", 1, 1001)
	actor.player.SetPosition(combat.PosResting)
	registerInWorld(t, actor)

	if err := ExecuteCommand(actor, "flip", nil); err != nil {
		t.Fatalf("ExecuteCommand returned error: %v", err)
	}
	if got := readMsgText(t, actor); got != "Nah... You feel too relaxed to do that..\r\n" {
		t.Fatalf("message = %q", got)
	}
}
