package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestTapRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["tap"]
	if !ok {
		t.Fatal("tap command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("tap gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["tap"]
	if !ok {
		t.Fatal("tap social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("tap social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You seem very impatient today.",
		"$n taps $s foot impatiently.",
		"You reach over and tap $N on the shoulder.",
		"$n reaches over and taps $N on the shoulder.",
		"$n reaches over and taps you on the shoulder.",
		"Really now?",
		"You tap yourself and go horizontal.",
		"$n taps $mself and goes horizontal.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("tap social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("tap social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
