package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestViolateRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["violate"]
	if !ok {
		t.Fatal("violate command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("violate gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["violate"]
	if !ok {
		t.Fatal("violate social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("violate social metadata = level %d, hide %d, min-victim %d; want 1, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"They call you... The Violater!",
		"$n looks like $e is just about to go on a violation rampage.",
		"You disgustingly violate $N.",
		"$N looks appalled as $n positively violates $M.",
		"$n violates you in unspeakable ways.",
		"You could get arrested for violating the air, you know.",
		"You feel violated!",
		"$n looks like $e's been violated.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("violate social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("violate social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
