package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestLeerRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["leer"]
	if !ok {
		t.Fatal("leer command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("leer gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["leer"]
	if !ok {
		t.Fatal("leer social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("leer social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Keep those nasty thoughts to yourself!",
		"$n leers at no one in particular.",
		"You start leering at $M.",
		"$n starts leering at $N.",
		"$n leers at you.  How rude.",
		"Smirk all you like.",
		"Eh?  Why would you want to do that?",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("leer social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("leer social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
