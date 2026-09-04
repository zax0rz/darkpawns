package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestStretchRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["stretch"]
	if !ok {
		t.Fatal("stretch command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("stretch gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["stretch"]
	if !ok {
		t.Fatal("stretch social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("stretch social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You stretch your arms out.",
		"$n stretchs $s arms out and lets out a yawn.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("stretch social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("stretch social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
