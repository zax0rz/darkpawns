package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestLickRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["lick"]
	if !ok {
		t.Fatal("lick command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("lick gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["lick"]
	if !ok {
		t.Fatal("lick social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("lick social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You lick your mouth and smile.",
		"$n licks $s mouth and smiles.",
		"You lick $M.",
		"$n licks $N.",
		"$n licks you.",
		"Lick away, nobody's here with that name.",
		"You lick yourself.",
		"$n licks $mself -- YUCK.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("lick social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("lick social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
