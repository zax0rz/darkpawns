package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestAgreeRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["agree"]
	if !ok {
		t.Fatal("agree command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("agree gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["agree"]
	if !ok {
		t.Fatal("agree social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("agree social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	if len(social.Messages) != 8 {
		t.Fatalf("agree social has %d messages, want 8", len(social.Messages))
	}
}
