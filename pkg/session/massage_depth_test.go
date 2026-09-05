package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestMassageRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["massage"]
	if !ok {
		t.Fatal("massage command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("massage gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["massage"]
	if !ok {
		t.Fatal("massage social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("massage social metadata = hide %d, min-level %d, min-victim %d; want all zero", social.HideFlag, social.MinLevel, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Massage what, thin air?",
		"#",
		"You gently massage $N's shoulders.",
		"$n massages $N's shoulders.",
		"$n gently massages your shoulders...ahhhhhhhhhh...",
		"You can only massage someone in the same room as you.",
		"You practice yoga as you try to massage yourself.",
		"$n gives a show on yoga-positions, trying to massage $mself.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("massage social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("massage social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
