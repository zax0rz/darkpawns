package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWhimperRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["whimper"]
	if !ok {
		t.Fatal("whimper command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("whimper gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["whimper"]
	if !ok {
		t.Fatal("whimper social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("whimper social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You whimper cowardly.",
		"$n whimpers in the corner, what a coward.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("whimper social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("whimper social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
