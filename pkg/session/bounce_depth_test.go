package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBounceRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["bounce"]
	if !ok {
		t.Fatal("bounce command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
		t.Fatalf("bounce gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosStanding)
	}

	social, ok := game.Socials["bounce"]
	if !ok {
		t.Fatal("bounce social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("bounce social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	if len(social.Messages) != 8 {
		t.Fatalf("bounce social has %d messages, want 8", len(social.Messages))
	}
}
