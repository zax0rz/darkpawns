package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSulkRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["sulk"]
	if !ok {
		t.Fatal("sulk command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("sulk gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["sulk"]
	if !ok {
		t.Fatal("sulk social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("sulk social metadata = level %d, hide %d, min-victim %d; want 1, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You sulk.",
		"$n sulks in the corner.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("sulk social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("sulk social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
