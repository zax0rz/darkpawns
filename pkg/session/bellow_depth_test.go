package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBellowRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["bellow"]
	if !ok {
		t.Fatal("bellow command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("bellow gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["bellow"]
	if !ok {
		t.Fatal("bellow social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("bellow social metadata = hide %d, victim-position %d, override %d; want 1/0/0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	if len(social.Messages) != 8 {
		t.Fatalf("bellow social has %d messages, want 8", len(social.Messages))
	}
}
