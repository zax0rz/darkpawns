package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShiverRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["shiver"]
	if !ok {
		t.Fatal("shiver command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("shiver gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["shiver"]
	if !ok {
		t.Fatal("shiver social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("shiver social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Brrrrrrrrr.",
		"$n shivers uncomfortably.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("shiver social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("shiver social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
