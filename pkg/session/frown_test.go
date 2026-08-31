package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestFrownRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["frown"]
	if !ok {
		t.Fatal("frown command gate is not registered")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosResting {
		t.Fatalf("frown gate = (%d, %d), want (0, %d)", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["frown"]
	if !ok {
		t.Fatal("frown social is not registered")
	}
	if len(social.Messages) < 3 {
		t.Fatalf("frown social has %d messages, want at least 3", len(social.Messages))
	}
	if social.Messages[2] != "#" {
		t.Fatalf("frown char-found message = %q, want #", social.Messages[2])
	}
}
