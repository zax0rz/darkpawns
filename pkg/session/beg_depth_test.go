package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBegRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["beg"]
	if !ok {
		t.Fatal("beg command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("beg gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["beg"]
	if !ok {
		t.Fatal("beg social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("beg social metadata = hide %d, victim-position %d, override %d; want 0/0/0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	if len(social.Messages) != 8 {
		t.Fatalf("beg social has %d messages, want 8", len(social.Messages))
	}
}
