package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWaveRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["wave"]
	if !ok {
		t.Fatal("wave command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("wave gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["wave"]
	if !ok {
		t.Fatal("wave social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("wave social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You wave.",
		"$n waves happily.",
		"You wave goodbye to $N.",
		"$n waves goodbye to $N.",
		"$n waves goodbye to you.  Have a good journey.",
		"They didn't wait for you to wave goodbye.",
		"Are you going on adventures as well??",
		"$n waves goodbye to $mself.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("wave social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("wave social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
