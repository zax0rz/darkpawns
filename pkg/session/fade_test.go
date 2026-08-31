package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestFadeRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["fade"]
	if !ok {
		t.Fatal("fade command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("fade gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}
}

func TestFadeSleepingActorHitsCPositionGate(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Fadeactor", 1, 1001)
	actor.player.SetPosition(combat.PosSleeping)
	registerInWorld(t, actor)

	if err := ExecuteCommand(actor, "fade", nil); err != nil {
		t.Fatalf("ExecuteCommand returned error: %v", err)
	}
	if got := readMsgText(t, actor); got != "In your dreams, or what?\r\n" {
		t.Fatalf("message = %q", got)
	}
}
