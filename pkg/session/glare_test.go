package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestGlareRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["glare"]
	if !ok {
		t.Fatal("glare command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("glare gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["glare"]
	if !ok {
		t.Fatal("glare social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != combat.PosResting {
		t.Fatalf("glare social metadata = hide %d, min-level %d; want hide 0, victim position %d", social.MinLevel, social.HideFlag, combat.PosResting)
	}
}
