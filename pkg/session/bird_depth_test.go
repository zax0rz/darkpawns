package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBirdRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["bird"]
	if !ok {
		t.Fatal("bird command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("bird gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["bird"]
	if !ok {
		t.Fatal("bird social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("bird social metadata = hide %d, victim-position %d, override %d; want 0/0/0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	if len(social.Messages) != 8 {
		t.Fatalf("bird social has %d messages, want 8", len(social.Messages))
	}
}
