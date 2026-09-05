package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestScratchRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["scratch"]
	if !ok {
		t.Fatal("scratch command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("scratch gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["scratch"]
	if !ok {
		t.Fatal("scratch social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("scratch social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You scratch your head in puzzlement.",
		"$n scratches $s head in puzzlement.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("scratch social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("scratch social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
