package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestStartleRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["startle"]
	if !ok {
		t.Fatal("startle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("startle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["startle"]
	if !ok {
		t.Fatal("startle social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("startle social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You flinch, startled from your reverie.",
		"$n flinches, startled from $s reverie.",
		"You sneak up on $N and scare $M out of $S wits.",
		"$n sneaks up on $N and scares $M out of $S wits.",
		"$n sneaks up on you and scares you out of your wits.",
		"You're startled by the fact THEY'RE NOT HERE!",
		"You flinch, startled from your reverie.",
		"$n flinches, startled from $s reverie.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("startle social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("startle social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
