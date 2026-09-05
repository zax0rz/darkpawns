package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestGrimaceRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["grimace"]
	if !ok {
		t.Fatal("grimace command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("grimace gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["grimace"]
	if !ok {
		t.Fatal("grimace social is not registered")
	}
	if social.HideFlag != 0 || social.MinLevel != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("grimace social metadata = hide %d, min-level %d, min-victim %d; want all zero", social.HideFlag, social.MinLevel, social.MinVictimPosition)
	}
	wantMessages := []string{"You grimace.", "$n grimaces.", "#"}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("grimace social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("grimace social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
