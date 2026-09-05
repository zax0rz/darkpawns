package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestKissRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["kiss"]
	if !ok {
		t.Fatal("kiss command gate is not registered")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosResting {
		t.Fatalf("kiss gate = (%d, %d), want (0, %d)", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["kiss"]
	if !ok {
		t.Fatal("kiss social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("kiss social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Isn't there someone you want to kiss?",
		"#",
		"You kiss $M.",
		"$n kisses $N.",
		"$n kisses you.",
		"Never around when required.",
		"All the lonely people :(",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("kiss social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("kiss social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
