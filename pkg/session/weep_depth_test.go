package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWeepRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["weep"]
	if !ok {
		t.Fatal("weep command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("weep gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["weep"]
	if !ok {
		t.Fatal("weep social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("weep social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Boo, hoo, hoo... boo, hoo, hoo...",
		"$n weeps with regal sadness.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("weep social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("weep social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
