package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSpankRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["spank"]
	if !ok {
		t.Fatal("spank command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("spank gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["spank"]
	if !ok {
		t.Fatal("spank social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 8 || social.MinVictimPosition != 0 {
		t.Fatalf("spank social metadata = level %d, hide %d, victim-position %d; want level 0, 8, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You spank WHO?  Eh?  How?  Naaah, you'd never.",
		"$n spanks the thin air with a flat hand.",
		"You spank $M vigorously, long and hard.  Your hand hurts.",
		"$n spanks $N over $s knee.  It hurts to even watch.",
		"$n spanks you long and hard.  You feel like a naughty child.",
		"Are you sure about this?  I mean, that person isn't even here!",
		"Hmm, not likely.",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("spank social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("spank social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
