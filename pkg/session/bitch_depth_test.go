package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBitchRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["bitch"]
	if !ok {
		t.Fatal("bitch command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("bitch gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["bitch"]
	if !ok {
		t.Fatal("bitch social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("bitch social metadata = hide %d, victim-position %d, override %d; want 0/5/0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	if len(social.Messages) != 8 {
		t.Fatalf("bitch social has %d messages, want 8", len(social.Messages))
	}
}
