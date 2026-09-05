package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestRaiseRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["raise"]
	if !ok {
		t.Fatal("raise command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("raise gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["raise"]
	if !ok {
		t.Fatal("raise social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != combat.PosResting || social.MinVictimPosition != 0 {
		t.Fatalf("raise social metadata = level %d, hide %d, min-victim %d; want 0, %d, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition, combat.PosResting)
	}
	want := []string{
		"You raise your eyebrows in question.",
		"$n raises one of $s eyebrows in question.",
		"You raise your eyebrows in question at $N.",
		"You see $n raise $s eyebrows at $N in question.",
		"$n raises $s eyebrows at you in question.",
		"Raise your eyebrows at who?",
		"You try to see yourself raising your eyebrows.",
		"You watch $n contort $s face to watch $s eyebrows.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("raise social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("raise social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
