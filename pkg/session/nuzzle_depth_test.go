package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestNuzzleRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["nuzzle"]
	if !ok {
		t.Fatal("nuzzle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("nuzzle gate = (%d, %d), want (0, %d)", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["nuzzle"]
	if !ok {
		t.Fatal("nuzzle social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != combat.PosResting || social.MinVictimPosition != 0 {
		t.Fatalf("nuzzle social metadata = level %d, hide %d, min-victim %d; want 1, %d, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition, combat.PosResting)
	}
	wantMessages := []string{
		"Nuzzle who??",
		"#",
		"You nuzzle $S neck softly.",
		"$n softly nuzzles $N's neck.",
		"$n softly nuzzles your neck.",
		"No.. they aren't here..",
		"I'm sorry, friend, but that's impossible.",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("nuzzle social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("nuzzle social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
