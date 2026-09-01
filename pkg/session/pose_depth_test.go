package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPoseRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["pose"]
	if !ok {
		t.Fatal("pose command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
		t.Fatalf("pose gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosStanding)
	}

	social, ok := game.Socials["pose"]
	if !ok {
		t.Fatal("pose social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("pose social metadata = level %d, hide %d, victim-position %d; want level 1, hide 0, victim-position 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You stand as tall as the gods themselves.",
		"$n suffers from a severe case of schizophrenic megalomania.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("pose social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("pose social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
