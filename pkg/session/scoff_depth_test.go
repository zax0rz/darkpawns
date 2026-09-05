package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestScoffRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["scoff"]
	if !ok {
		t.Fatal("scoff command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("scoff gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["scoff"]
	if !ok {
		t.Fatal("scoff social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("scoff social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You scoff at the idea.",
		"$n scoffs at the idea.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("scoff social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("scoff social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
