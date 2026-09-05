package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSlapRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["slap"]
	if !ok {
		t.Fatal("slap command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("slap gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["slap"]
	if !ok {
		t.Fatal("slap social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("slap social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Normally you slap SOMEBODY.",
		"#",
		"You slap $N.",
		"$n slaps $N.",
		"You are slapped by $n.",
		"How about slapping someone in the same room as you??",
		"You slap yourself, silly you.",
		"$n slaps $mself, really strange...",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("slap social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("slap social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
