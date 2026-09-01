package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestMoanRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["moan"]
	if !ok {
		t.Fatal("moan command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("moan gate = (%d, %d), want (0, %d)", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["moan"]
	if !ok {
		t.Fatal("moan social is not registered")
	}
	if social.HideFlag != 0 || social.MinLevel != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("moan social metadata = hide %d, min-level %d, min-victim %d; want all zero", social.HideFlag, social.MinLevel, social.MinVictimPosition)
	}
	wantMessages := []string{"You start to moan.", "$n starts moaning.", "#"}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("moan social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("moan social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
