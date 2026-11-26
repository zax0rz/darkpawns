package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

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
