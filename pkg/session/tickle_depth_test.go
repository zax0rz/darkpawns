package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestTickleRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["tickle"]
	if !ok {
		t.Fatal("tickle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("tickle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["tickle"]
	if !ok {
		t.Fatal("tickle social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("tickle social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Who do you want to tickle??",
		"#",
		"You tickle $N.",
		"$n tickles $N.",
		"$n tickles you - hee hee hee.",
		"Who is that??",
		"You tickle yourself, how funny!",
		"$n tickles $mself.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("tickle social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("tickle social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
