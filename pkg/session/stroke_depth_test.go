package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestStrokeRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["stroke"]
	if !ok {
		t.Fatal("stroke command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("stroke gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["stroke"]
	if !ok {
		t.Fatal("stroke social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("stroke social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Whose thigh would you like to stroke?",
		"#",
		"You gently stroke $S inner thigh.",
		"$n gently strokes $N's inner thigh... hmm...",
		"$n gently strokes your inner thigh with feathery touches.",
		"That person is not within reach.",
		"You are about to do something you would rather not be caught doing.",
		"$n starts to do something disgusting and then stops.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("stroke social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("stroke social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
