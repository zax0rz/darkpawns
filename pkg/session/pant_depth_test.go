package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPantRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["pant"]
	if !ok {
		t.Fatal("pant command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("pant gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["pant"]
	if !ok {
		t.Fatal("pant social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != combat.PosResting || social.MinVictimPosition != 0 {
		t.Fatalf("pant social metadata = level %d, hide %d, min-victim %d; want 0, %d, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition, combat.PosResting)
	}
	wantMessages := []string{
		"You pant.",
		"$n pants.",
		"You pant in $N's direction.",
		"$n pants at $N.",
		"$n pants at you.",
		"Sorry, but that person doesn't seem to be here.",
		"You quitely pant to yourself.",
		"$n seems to be having a breathing problem.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("pant social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("pant social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
