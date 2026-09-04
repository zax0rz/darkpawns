package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestReelRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["reel"]
	if !ok {
		t.Fatal("reel command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("reel gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["reel"]
	if !ok {
		t.Fatal("reel social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("reel social metadata = level %d, hide %d, min-victim %d; want 1, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You reel around the room drunkenly.",
		"$n reels around like a drunken sot.",
		"You reel up to $N and breathe alcohol into $S face.",
		"$n reels up to $N and tries to steady $mself, and they both fall down.",
		"$n reels up to you and grabs on to you for support, knocking you over.",
		"Who? *laugh* Not here, buddy.",
		"You reel around, feeling quite suave in your lack of sobriety.",
		"$n reels around and looks like it's time to visit the porcelain god.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("reel social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("reel social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
