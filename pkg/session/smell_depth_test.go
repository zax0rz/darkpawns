package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSmellRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["smell"]
	if !ok {
		t.Fatal("smell command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("smell gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["smell"]
	if !ok {
		t.Fatal("smell social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("smell social metadata = level %d, hide %d, victim-position %d; want 1/0/0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You take a whiff.  Phew!",
		"$n sniffs around the room and wrinkles $s nose.",
		"You snuffle around the crotch of $N.",
		"$n plants $s nose in the crotch of $N and snuffles around like a dog.",
		"$n sticks $s nose right in your crotch.",
		"You smell... the invisible man!",
		"You smell your armpits.  Perhaps you should bathe.",
		"$n takes a big whiff of $s armpits and wonders if $e needs a bath.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("smell social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("smell social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
