package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSqueezeRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["squeeze"]
	if !ok {
		t.Fatal("squeeze command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("squeeze gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["squeeze"]
	if !ok {
		t.Fatal("squeeze social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("squeeze social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Where, what, how, WHO???",
		"#",
		"You squeeze $M fondly.",
		"$n squeezes $N fondly.",
		"$n squeezes you fondly.",
		"Where, what, how, WHO???",
		"You squeeze yourself -- try to relax a little!",
		"$n squeezes $mself.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("squeeze social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("squeeze social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
