package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestRoarRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["roar"]
	if !ok {
		t.Fatal("roar command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("roar gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["roar"]
	if !ok {
		t.Fatal("roar social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("roar social metadata = level %d, hide %d, victim-position %d; want level 0, hide 0, victim-position 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You let out a viscious roar!",
		"$n raises $s head and lets out a viscious roar!",
		"Your let out a viscious roar at $M!",
		"$n lets out a viscious roar at $N!",
		"$n lets out a viscious roar at you!",
		"Roar at whom?",
		"You let out a viscious roar!",
		"$n raises $s head and lets out a viscious roar!",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("roar social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("roar social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
