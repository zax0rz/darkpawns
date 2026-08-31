package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestDoActionFlirtSleepingTargetHitsPositionGate(t *testing.T) {
	w, actor, target, _, output := newChannelWorld(t)
	target.SetPosition(combat.PosSleeping)

	DoAction(w, actor, "flirt", target.Name)

	if got := channelOutput(output, actor.Name); got != "Local is not in a proper position for that.\r\n" {
		t.Fatalf("actor output = %q", got)
	}
	if got := channelOutput(output, target.Name); got != "" {
		t.Fatalf("sleeping target output = %q", got)
	}
}
