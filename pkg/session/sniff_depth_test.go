package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSniffRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	gate, ok := commandGates["sniff"]
	if !ok {
		t.Fatal("sniff command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosResting {
		t.Fatalf("sniff gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["sniff"]
	if !ok {
		t.Fatal("sniff social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("sniff social metadata = level %d hide %d victim-position %d, want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{"You sniff sadly.  *SNIFF*", "$n sniffs sadly.", "#"}
	if len(social.Messages) != len(want) {
		t.Fatalf("sniff social messages = %d, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("sniff social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
