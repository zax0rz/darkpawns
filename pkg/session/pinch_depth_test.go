package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPinchRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["pinch"]
	if !ok {
		t.Fatal("pinch command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("pinch gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["pinch"]
	if !ok {
		t.Fatal("pinch social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("pinch social metadata = level %d, hide %d, victim-position %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Pinch who?",
		"#",
		"You pinch $M right on the butt.",
		"$n pinches $N on the butt.",
		"$n pinches your butt.",
		"Pinch who?",
		"You pinch yourself to see if you're dreaming.",
		"$n pinches $mself to makes sure $e's awake.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("pinch social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("pinch social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
