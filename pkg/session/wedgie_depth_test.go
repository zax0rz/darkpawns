package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWedgieRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["wedgie"]
	if !ok {
		t.Fatal("wedgie command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("wedgie gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["wedgie"]
	if !ok {
		t.Fatal("wedgie social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("wedgie social metadata = C-hide %d, C-min-victim %d, override %d; want 0, 5, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You pull your underwear out of your crack!",
		"You watch as $n un-wedgies $mself.",
		"You grab $N's underwear and pull up HARD.",
		"You see $n give a painful wedgie to $N.",
		"$n almost rips your underwear off with an incredible wedgie.",
		"Wedgie who?",
		"You give yourself a wedgie.",
		"You watch $n pull $s underwear nearly over $s head.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("wedgie social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("wedgie social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
