package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSpewRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["spew"]
	if !ok {
		t.Fatal("spew command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("spew gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["spew"]
	if !ok {
		t.Fatal("spew social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("spew social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"SPEW!!!!!",
		"EWW!!! $n spews all over.",
		"EWW!!! You spew all over $M.",
		"YECK!!! $n spews all over $N.",
		"YECK!!! $n spews all over you!",
		"Again?",
		"You spew all over yourself.",
		"$n spews all over $s own clothes.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("spew social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("spew social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
