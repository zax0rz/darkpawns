package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestChuckleRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["chuckle"]
	if !ok {
		t.Fatal("chuckle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("chuckle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["chuckle"]
	if !ok {
		t.Fatal("chuckle social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("chuckle social metadata = hide %d, victim-position %d, override %d; want hide 0, victim-position 0, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{"You chuckle politely.", "$n chuckles politely.", "#"}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("chuckle social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("chuckle social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
