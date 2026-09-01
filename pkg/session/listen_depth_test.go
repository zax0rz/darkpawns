package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestListenRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["listen"]
	if !ok {
		t.Fatal("listen command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("listen gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["listen"]
	if !ok {
		t.Fatal("listen social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("listen social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{"You listen.", "$n listens.", "#"}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("listen social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("listen social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
