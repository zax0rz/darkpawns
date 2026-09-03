package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSpitRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["spit"]
	if !ok {
		t.Fatal("spit command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("spit gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["spit"]
	if !ok {
		t.Fatal("spit social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("spit social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You spit over your left shoulder.",
		"$n spits over $s left shoulder.",
		"You spit on $M.",
		"$n spits in $N's face.",
		"$n spits in your face.",
		"Can you spit that far?",
		"You drool down your front.",
		"$n drools down $s front.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("spit social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("spit social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
