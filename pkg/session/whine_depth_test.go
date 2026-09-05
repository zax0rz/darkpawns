package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWhineRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["whine"]
	if !ok {
		t.Fatal("whine command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("whine gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["whine"]
	if !ok {
		t.Fatal("whine social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("whine social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You whine pitifully.",
		"$n whines pitifully about the whole situation.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("whine social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("whine social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
