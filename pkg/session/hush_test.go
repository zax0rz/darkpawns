package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestHushRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["hush"]
	if !ok {
		t.Fatal("hush command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("hush gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["hush"]
	if !ok {
		t.Fatal("hush social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("hush social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You shut up before it's too late.",
		"$n shuts up before it's too late.",
		"You hush $M with a firm backhand to the mouth.",
		"$n hushes $N with a firm backhand to the mouth.",
		"$n hushes you with a firm backhand to the mouth.",
		"They are extremely quiet.. somewhere else.",
		"You shut up before it's too late.",
		"$n shuts up before it's too late.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("hush social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("hush social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
