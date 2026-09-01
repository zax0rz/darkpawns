package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPoutRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["pout"]
	if !ok {
		t.Fatal("pout command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("pout gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["pout"]
	if !ok {
		t.Fatal("pout social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("pout social metadata = level %d, hide %d, victim-position %d; want level 0, hide 0, victim-position 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Ah, don't take it so hard.",
		"$n pouts.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("pout social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("pout social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
