package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPonderRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["ponder"]
	if !ok {
		t.Fatal("ponder command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("ponder gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["ponder"]
	if !ok {
		t.Fatal("ponder social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("ponder social metadata = level %d, hide %d, victim-position %d; want level 1, hide 0, victim-position 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You ponder over matters as they appear to you at this moment.",
		"$n sinks deeply into $s own thoughts.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("ponder social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("ponder social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
