package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestThunkRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["thunk"]
	if !ok {
		t.Fatal("thunk command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("thunk gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["thunk"]
	if !ok {
		t.Fatal("thunk social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("thunk social metadata = level %d, hide %d, min-victim %d; want 0, 5, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You hit your head and hear a hollow thunk.",
		"$n hits $s head with a hollow thunk.",
		"You thunk $N hollowly on the head.",
		"$n thunks $N hollowly on the head.",
		"$n thunks you hollowly on the head.",
		"Thunk who?",
		"You hit your head and hear a hollow thunk.",
		"$n hits $s head with a hollow thunk.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("thunk social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("thunk social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
