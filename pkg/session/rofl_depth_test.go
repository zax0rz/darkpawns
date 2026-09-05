package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestRoflRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["rofl"]
	if !ok {
		t.Fatal("rofl command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("rofl gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["rofl"]
	if !ok {
		t.Fatal("rofl social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("rofl social metadata = level %d, hide %d, victim-position override %d; want level 0, hide 5, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You roll on the floor laughing.",
		"$n rolls on the floor laughing.",
		"$n rolls on the floor laughing at $N.",
		"$n rolls on the floor laughing at you.",
		"Laugh at who? They aren't here.",
		"You roll on the floor laughing at yourself.",
		"$n rolls on the floor laughing at $mself.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("rofl social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("rofl social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
