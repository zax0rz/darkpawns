package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestLaughRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["laugh"]
	if !ok {
		t.Fatal("laugh command gate is not registered")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosResting {
		t.Fatalf("laugh gate = (%d, %d), want (0, %d)", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["laugh"]
	if !ok {
		t.Fatal("laugh social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("laugh social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You laugh.",
		"$n laughs.",
		"You laugh at $M.",
		"$n laughs at $N.",
		"$n laughs at you.",
		"Laugh at who?",
		"You laugh at yourself.",
		"$n laughs at $mself.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("laugh social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("laugh social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
