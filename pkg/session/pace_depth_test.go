package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPaceRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["pace"]
	if !ok {
		t.Fatal("pace command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("pace gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["pace"]
	if !ok {
		t.Fatal("pace social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != combat.PosResting || social.MinVictimPosition != 0 {
		t.Fatalf("pace social metadata = level %d, hide %d, min-victim %d; want 0, %d, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition, combat.PosResting)
	}
	wantMessages := []string{
		"You pace back and forth.",
		"$n paces back and forth.",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("pace social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("pace social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
