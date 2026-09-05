package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestTackleRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["tackle"]
	if !ok {
		t.Fatal("tackle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("tackle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["tackle"]
	if !ok {
		t.Fatal("tackle social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("tackle social metadata = level %d, hide %d, min-victim %d; want 0, 5, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You tackle the air.  It stands not a chance.",
		"$n starts running around $mself in a desperate attempt to tackle the air.",
		"You ruthlessly tackle $M to the ground.",
		"$n ruthlessly tackles $N, pinning $M to the ground.",
		"$n suddenly lunges at you and tackles you to the ground!",
		"That person isn't here (lucky for them, it would seem...)",
		"Tackle yourself?  Yeah, right....",
		"$n makes a dextrous move and kicks $s left leg away with $s right.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("tackle social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("tackle social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
