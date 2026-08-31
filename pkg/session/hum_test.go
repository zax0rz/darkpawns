package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestHumRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["hum"]
	if !ok {
		t.Fatal("hum command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("hum gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["hum"]
	if !ok {
		t.Fatal("hum social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("hum social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You hum all innocent-like.",
		"$n hums innocently.",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("hum social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("hum social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
