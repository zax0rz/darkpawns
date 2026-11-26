package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

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
