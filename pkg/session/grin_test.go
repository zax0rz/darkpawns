package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestGrinRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["grin"]
	if !ok {
		t.Fatal("grin command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("grin gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["grin"]
	if !ok {
		t.Fatal("grin social is not registered")
	}
	if social.HideFlag != 0 || social.MinLevel != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("grin social metadata = hide %d, min-level %d, min-victim %d; want all zero", social.HideFlag, social.MinLevel, social.MinVictimPosition)
	}
	if len(social.Messages) != 8 {
		t.Fatalf("grin social has %d messages, want 8", len(social.Messages))
	}
}
