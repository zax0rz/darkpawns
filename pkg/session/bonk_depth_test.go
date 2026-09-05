package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBonkRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["bonk"]
	if !ok {
		t.Fatal("bonk command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("bonk gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["bonk"]
	if !ok {
		t.Fatal("bonk social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("bonk social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.MinVictimPosition, social.MinLevel)
	}
	wantMessages := []string{
		"Who do you want to bonk?",
		"$n waves $s hands around.",
		"You bonk $M on the head.",
		"$n bonks $N on the head.",
		"$n bonks you on the head..",
		"Who's that?",
		"You bonk yourself on the head.",
		"$n bonks $mself on the head.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("bonk social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("bonk social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
