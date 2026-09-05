package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSingRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["sing"]
	if !ok {
		t.Fatal("sing command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("sing gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["sing"]
	if !ok {
		t.Fatal("sing social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("sing social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You raise your clear (?) voice towards the sky.",
		"SEEK SHELTER AT ONCE!  $n has begun to sing.",
		"You begin to seranade $M. Lalala...",
		"$n raises $s voice and beings to seranade $N. Lalala.",
		"$n raises $s voice and beings to seranade you. Lalala.",
		"They can't hear you, wherever they are.",
		"You sing to yourself.  Ladidadida.",
		"$n sings to $mself.  Ladidadida.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("sing social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("sing social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
