package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWorshipRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["worship"]
	if !ok {
		t.Fatal("worship command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("worship gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["worship"]
	if !ok {
		t.Fatal("worship social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("worship social metadata = C-hide %d, C-min-victim %d, override %d; want 0, 5, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You find yourself head-down in the dirt, worshipping.",
		"$n starts worshipping nothing at all.",
		"You fall to your knees and worship $M deeply.",
		"$n falls to $s knees, worshipping $N with uncanny dedication.",
		"$n kneels before you in solemn worship.",
		"Uh.. who?  They're not here, pal.",
		"You seem sure to have found a true deity.....",
		"$n falls to $s knees and humbly worships $mself.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("worship social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("worship social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
