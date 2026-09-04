package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestThreatenRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["threaten"]
	if !ok {
		t.Fatal("threaten command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("threaten gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["threaten"]
	if !ok {
		t.Fatal("threaten social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("threaten social metadata = level %d, hide %d, min-victim %d; want 0, 5, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You threaten the room wildly.",
		"You watch as $n threatens everyone in the room.",
		"You threaten $N, that dirty rat!",
		"You see $n threaten $N, maybe you should step in and calm things down.",
		"$n threatens you, how mean!",
		"Threaten who?",
		"#",
		"",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("threaten social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("threaten social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
