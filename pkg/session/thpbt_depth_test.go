package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestThpbtRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["thpbt"]
	if !ok {
		t.Fatal("thpbt command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("thpbt gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["thpbt"]
	if !ok {
		t.Fatal("thpbt social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("thpbt social metadata = level %d, hide %d, min-victim %d; want 0, 5, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You stick out your tongue and go 'thhppbbbttt!'",
		"$n sticks out $s tongue and goes 'thhppbbbttt!'",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("thpbt social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("thpbt social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
