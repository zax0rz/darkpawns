package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWhapRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["whap"]
	if !ok {
		t.Fatal("whap command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("whap gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["whap"]
	if !ok {
		t.Fatal("whap social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("whap social metadata = C-hide %d, C-min-victim %d, override %d; want 0, 5, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You whap the ground.",
		"$n whaps the ground.",
		"You whap $N upside the head.",
		"$n whaps $N upside the head.",
		"$n whaps you upside the head.",
		"Whap who? They aren't here.",
		"You whap yourself upside the head.",
		"$n tries to knock some sense into $mself.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("whap social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("whap social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
