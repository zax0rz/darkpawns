package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestHowlRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["howl"]
	if !ok {
		t.Fatal("howl command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("howl gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["howl"]
	if !ok {
		t.Fatal("howl social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("howl social metadata = hide %d, victim-position %d, override %d; want hide 1, victim-position 0, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You howl loudly and mournfully.",
		"$n howls loudly and mournfully, ah-OOOOoooooo...",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("howl social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("howl social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
