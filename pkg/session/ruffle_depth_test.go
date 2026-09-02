package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestRuffleRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["ruffle"]
	if !ok {
		t.Fatal("ruffle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
		t.Fatalf("ruffle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosStanding)
	}

	social, ok := game.Socials["ruffle"]
	if !ok {
		t.Fatal("ruffle social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("ruffle social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You've got to ruffle SOMEONE.",
		"#",
		"You ruffle $N's hair playfully.",
		"$n ruffles $N's hair playfully.",
		"$n ruffles your hair playfully.",
		"Might be a bit difficult.",
		"You ruffle your hair, wondering how far you can go before the rest think you're crazy.",
		"$n ruffles $s hair -- weirdo!",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("ruffle social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("ruffle social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
