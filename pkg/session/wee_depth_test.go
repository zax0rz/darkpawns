package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWeeRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["wee"]
	if !ok {
		t.Fatal("wee command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("wee gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["wee"]
	if !ok {
		t.Fatal("wee social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("wee social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Weeeee!!!!!!",
		"Flapping $s hands up and down like a bird, $n runs around screaming: \"Weeee!!!\"",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("wee social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("wee social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
