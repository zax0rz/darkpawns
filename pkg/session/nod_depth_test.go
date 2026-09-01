package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestNodRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["nod"]
	if !ok {
		t.Fatal("nod command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("nod gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["nod"]
	if !ok {
		t.Fatal("nod social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("nod social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You nod.",
		"$n nods.",
		"You nod at $M.",
		"$n nods at $N.",
		"$n nods at you.",
		"Who?",
		"#",
		"",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("nod social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("nod social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
