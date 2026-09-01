package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestNibbleRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["nibble"]
	if !ok {
		t.Fatal("nibble command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("nibble gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["nibble"]
	if !ok {
		t.Fatal("nibble social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("nibble social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Nibble on who?",
		"#",
		"You nibble on $N's ear.",
		"$n nibbles on $N's ear.",
		"$n nibbles on your ear.",
		"Sorry, not here, better go back to dreaming about it.",
		"You nibble on your OWN ear???????????????????",
		"$n nibbles on $s OWN ear (I wonder how it is done!!).",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("nibble social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("nibble social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
