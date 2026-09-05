package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSaluteRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["salute"]
	if !ok {
		t.Fatal("salute command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("salute gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["salute"]
	if !ok {
		t.Fatal("salute social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != combat.PosResting || social.MinVictimPosition != 0 {
		t.Fatalf("salute social metadata = level %d, hide %d, victim-position %d; want level 0, hide %d, victim-position 0", social.MinLevel, social.HideFlag, social.MinVictimPosition, combat.PosResting)
	}
	want := []string{
		"You salute with a quick snap of the wrist.",
		"$n snaps to attention and does a salute.",
		"You salute $N.",
		"$n salutes $N.",
		"$n salutes you.",
		"Salute who? They aren't here.",
		"You try to salute yourself.",
		"$n nearly breaks $s wrist trying to salute $mself.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("salute social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("salute social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
