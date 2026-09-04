package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestScreamRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["scream"]
	if !ok {
		t.Fatal("scream command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("scream gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["scream"]
	if !ok {
		t.Fatal("scream social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("scream social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"ARRRRRRRRRRGH!!!!!",
		"$n screams loudly!",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("scream social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("scream social message %d = %q, want %q", i, message, want[i])
		}
	}
}
