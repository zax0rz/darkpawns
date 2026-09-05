package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBlinkRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["blink"]
	if !ok {
		t.Fatal("blink command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("blink gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["blink"]
	if !ok {
		t.Fatal("blink social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("blink social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.MinVictimPosition, social.MinLevel)
	}
	wantMessages := []string{
		"You blink your eyelashes innocently.",
		"$n blinks $s eyelashes innocently.",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("blink social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("blink social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
