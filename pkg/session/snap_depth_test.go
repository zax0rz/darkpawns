package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSnapRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["snap"]
	if !ok {
		t.Fatal("snap command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("snap gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["snap"]
	if !ok {
		t.Fatal("snap social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("snap social metadata = level %d, hide %d, victim-position %d; want 0/0/0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"PRONTO!  You snap your fingers.",
		"$n snaps $s fingers.",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("snap social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("snap social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
