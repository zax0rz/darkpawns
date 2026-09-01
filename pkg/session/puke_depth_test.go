package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPukeRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["puke"]
	if !ok {
		t.Fatal("puke command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("puke gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["puke"]
	if !ok {
		t.Fatal("puke social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("puke social metadata = level %d, hide %d, victim-position %d; want level 0, hide 0, victim-position 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You puke.",
		"$n pukes.",
		"You puke on $M.",
		"$n pukes on $N.",
		"$n pukes on your clothes!",
		"Once again?",
		"You puke on yourself.",
		"$n pukes on $s clothes.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("puke social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("puke social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
