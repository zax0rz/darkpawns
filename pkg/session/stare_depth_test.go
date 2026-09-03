package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestStareRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["stare"]
	if !ok {
		t.Fatal("stare command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("stare gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["stare"]
	if !ok {
		t.Fatal("stare social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != combat.PosResting || social.MinVictimPosition != 0 {
		t.Fatalf("stare social metadata = level %d, hide %d, victim-position %d; want 0, %d, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition, combat.PosResting)
	}
	wantMessages := []string{
		"You stare at the sky.",
		"$n stares at the sky.",
		"You stare dreamily at $N, completely lost in $S eyes..",
		"$n stares dreamily at $N.",
		"$n stares dreamily at you, completely lost in your eyes.",
		"You stare and stare but can't see that person anywhere...",
		"You stare dreamily at yourself - enough narcissism for now.",
		"$n stares dreamily at $mself - NARCISSIST!",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("stare social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("stare social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
