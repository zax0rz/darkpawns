package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestThrottleRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["throttle"]
	if !ok {
		t.Fatal("throttle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("throttle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["throttle"]
	if !ok {
		t.Fatal("throttle social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("throttle social metadata = level %d, hide %d, min-victim %d; want 1, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You really feel like throttling someone!",
		"$n looks like $e could throttle someone!",
		"You wrap your hands around $S neck and throttle $M!",
		"$n wraps $s hands around $N's neck and throttles $M!",
		"$n grabs you by the neck and throttles you!",
		"Noone here by that name...",
		"You try to throttle yourself.",
		"$n tries to throttle $mself.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("throttle social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("throttle social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
