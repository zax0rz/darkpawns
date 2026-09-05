package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBurpRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["burp"]
	if !ok {
		t.Fatal("burp command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("burp gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["burp"]
	if !ok {
		t.Fatal("burp social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("burp social metadata = hide %d, victim-position %d, override %d; want hide 0, victim-position 0, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{"You burp loudly.", "$n burps loudly.", "#"}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("burp social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("burp social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
