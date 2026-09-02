package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestChideRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["chide"]
	if !ok {
		t.Fatal("chide command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("chide gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["chide"]
	if !ok {
		t.Fatal("chide social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("chide social metadata = hide %d, victim-position %d, override %d; want hide 0, victim-position 0, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You generally chide someone...",
		"#",
		"You gently chide $N.",
		"$n chides $N gently.",
		"$n chides you with a disapproving glance.",
		"Chide who?",
		"You chide yourself, laying a guilt trip on everyone.",
		"$n chides $mself, laying a guilt trip on you.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("chide social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("chide social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
