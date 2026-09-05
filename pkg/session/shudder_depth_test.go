package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShudderRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["shudder"]
	if !ok {
		t.Fatal("shudder command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("shudder gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["shudder"]
	if !ok {
		t.Fatal("shudder social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("shudder social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You shudder in fear.",
		"$n shudders in fear.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("shudder social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("shudder social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
