package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBlessRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["bless"]
	if !ok {
		t.Fatal("bless command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("bless gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["bless"]
	if !ok {
		t.Fatal("bless social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("bless social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.MinVictimPosition, social.MinLevel)
	}
	wantMessages := []string{
		"You mumble \"Bless you.\"",
		"$n mumbles \"Bless you.\"",
		"You nod at $M with mumbled \"Bless you.\"",
		"$n nods at $N and mumbles \"Bless you.\"",
		"$n nods at you and mumbles \"Bless you.\"",
		"Bless who?",
		"You bless yourself, since noone else will.",
		"$n blesses $mself, since noone else will.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("bless social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("bless social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
