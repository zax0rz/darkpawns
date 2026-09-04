package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestTiltRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["tilt"]
	if !ok {
		t.Fatal("tilt command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("tilt gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["tilt"]
	if !ok {
		t.Fatal("tilt social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("tilt social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You tilt your head in question.",
		"$n tilts $s head in question.",
		"You adjust $s head a little to the left.. perfect!",
		"$n adjusts $N's head a little to the left... perfect!",
		"$n adjusts your head a little to the left... perfect!",
		"Tilt who? where?",
		"You tilt your head in question.",
		"$n tilts $s head in question.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("tilt social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("tilt social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
