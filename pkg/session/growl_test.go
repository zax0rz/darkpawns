package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestGrowlRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["growl"]
	if !ok {
		t.Fatal("growl command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("growl gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["growl"]
	if !ok {
		t.Fatal("growl social is not registered")
	}
	if social.HideFlag != 0 || social.MinLevel != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("growl social metadata = hide %d, min-level %d, min-victim %d; want all zero", social.HideFlag, social.MinLevel, social.MinVictimPosition)
	}
	wantMessages := []string{"Grrrrrrrrrr...", "$n growls.", "#"}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("growl social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("growl social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
