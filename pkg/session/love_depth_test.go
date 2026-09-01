package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestLoveRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["love"]
	if !ok {
		t.Fatal("love command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("love gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["love"]
	if !ok {
		t.Fatal("love social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("love social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You love the whole world.",
		"$n loves everybody in the world.",
		"You tell your true feelings to $N.",
		"$n whispers softly to $N.",
		"$n whispers to you sweet words of love.",
		"Alas, your love is not present...",
		"Well, we already know you love yourself (lucky someone does!)",
		"$n loves $mself, can you believe it?",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("love social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("love social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
