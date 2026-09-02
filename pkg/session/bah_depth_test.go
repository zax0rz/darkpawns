package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBahRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["bah"]
	if !ok {
		t.Fatal("bah command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("bah gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["bah"]
	if !ok {
		t.Fatal("bah social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("bah social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	if len(social.Messages) != 3 {
		t.Fatalf("bah social has %d messages, want 3", len(social.Messages))
	}
}
