package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestApplaudRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["applaud"]
	if !ok {
		t.Fatal("applaud command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("applaud gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["applaud"]
	if !ok {
		t.Fatal("applaud social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("applaud social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.MinVictimPosition, social.MinLevel)
	}
	if len(social.Messages) != 8 {
		t.Fatalf("applaud social has %d messages, want 8", len(social.Messages))
	}
}
