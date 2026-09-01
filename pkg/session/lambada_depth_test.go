package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestLambadaRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["lambada"]
	if !ok {
		t.Fatal("lambada command gate is not registered")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosStanding {
		t.Fatalf("lambada gate = (%d, %d), want (0, %d)", gate.MinLevel, gate.MinPosition, combat.PosStanding)
	}

	social, ok := game.Socials["lambada"]
	if !ok {
		t.Fatal("lambada social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("lambada social metadata = level %d, hide %d, victim-position %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You do the forbidden dance with yourself.",
		"$n does a dance.. which if not already forbidden, it should be.",
		"You lambada with $N, getting quite horny but making $M nauseous.",
		"$n does the forbidden dance with $N.",
		"$n leads you in the steps of the forbidden dance.",
		"Really now?",
		"Feeling a little horny?",
		"$n does a dance.. which if not already forbidden, it should be.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("lambada social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("lambada social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
