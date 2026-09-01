package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPlotRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["plot"]
	if !ok {
		t.Fatal("plot command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("plot gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["plot"]
	if !ok {
		t.Fatal("plot social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("plot social metadata = level %d, hide %d, victim-position %d; want 1, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"A plot begins to form in your mind.",
		"$n's eyes light up with an evil glow as $e forms a plot.",
		"You make your dastardly plot against $N.",
		"$n looks perfectly evil as $e forms a plot against $N.",
		"You knew it!  You just knew $n was plotting against you!",
		"You plot against thin air.",
		"You set your evil plan in motion -- against yourself!",
		"$n forms a deadly plot against $mself.  Scary.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("plot social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("plot social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
