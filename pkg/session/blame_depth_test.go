package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBlameRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["blame"]
	if !ok {
		t.Fatal("blame command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("blame gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["blame"]
	if !ok {
		t.Fatal("blame social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("blame social metadata = hide %d, victim-position %d, override %d; want 0/5/0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	if len(social.Messages) != 8 {
		t.Fatalf("blame social has %d messages, want 8", len(social.Messages))
	}
}
