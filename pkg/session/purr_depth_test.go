package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPurrRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["purr"]
	if !ok {
		t.Fatal("purr command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("purr gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["purr"]
	if !ok {
		t.Fatal("purr social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("purr social metadata = level %d, hide %d, victim-position %d; want level 0, hide 0, victim-position 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"MMMMEEEEEEEEOOOOOOOOOWWWWWWWWWWWW.",
		"$n purrs contentedly.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("purr social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("purr social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
