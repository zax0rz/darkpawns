package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWaltzRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["waltz"]
	if !ok {
		t.Fatal("waltz command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("waltz gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["waltz"]
	if !ok {
		t.Fatal("waltz social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("waltz social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You waltz around the room.",
		"$n waltzs around the room.",
		"You waltz with $N.",
		"$n waltzs with $N.",
		"$n waltzs around the room with you.",
		"There's no one by that name around.",
		"You waltz around with an air partner, looking like Fred Astaire.",
		"$n waltzs around with an air partner, looking like Fred Astaire.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("waltz social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("waltz social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
