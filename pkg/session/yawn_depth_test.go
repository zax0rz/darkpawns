package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestYawnRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["yawn"]
	if !ok {
		t.Fatal("yawn command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("yawn gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["yawn"]
	if !ok {
		t.Fatal("yawn social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("yawn social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You yawn, bored and tired.",
		"$n yawns.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("yawn social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("yawn social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
