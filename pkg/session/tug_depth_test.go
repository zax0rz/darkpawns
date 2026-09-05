package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestTugRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["tug"]
	if !ok {
		t.Fatal("tug command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("tug gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["tug"]
	if !ok {
		t.Fatal("tug social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("tug social metadata = level %d, hide %d, min-victim %d; want 1, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You go around looking for someone to tug on.",
		"$n looks for someone's sleeve to tug on.",
		"You tug on $N's sleeve and whine.",
		"$n tugs on $N's sleeve and whines.",
		"$n tugs on your sleeve to get your attention.",
		"You tug on a non-existent sleeve.",
		"You tug on your own sleeve.  Whoops, you tore your shirt.",
		"$n tugs on $s own sleeve, creating a big hole in $s shirt.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("tug social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("tug social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
