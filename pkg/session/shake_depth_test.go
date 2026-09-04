package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShakeRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["shake"]
	if !ok {
		t.Fatal("shake command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("shake gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["shake"]
	if !ok {
		t.Fatal("shake social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("shake social metadata = level %d, hide %d, min-victim %d; want 0, 5, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You shake your head.",
		"$n shakes $s head.",
		"You shake $S hand.",
		"$n shakes $N's hand.",
		"$n shakes your hand.",
		"Sorry good buddy, but that person doesn't seem to be here.",
		"You are shaken by yourself.",
		"$n shakes and quivers like a bowlful of jelly.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("shake social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("shake social message %d = %q, want %q", i, message, want[i])
		}
	}
}
