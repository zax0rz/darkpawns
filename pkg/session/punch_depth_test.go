package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPunchRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["punch"]
	if !ok {
		t.Fatal("punch command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("punch gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["punch"]
	if !ok {
		t.Fatal("punch social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("punch social metadata = level %d, hide %d, victim-position %d; want level 0, hide 0, victim-position 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Punch the air?  Sure, go ahead, fine by me...",
		"$n starts shadow-boxing.",
		"You punch $M right in the face!  Yuck, the BLOOD!",
		"$n punches weakly at $N, missing by miles.",
		"$n tries a punch at you but misses by a good quarter-mile...",
		"Punch who?",
		"You punch yourself in the face resulting in your own nose being bloodied.",
		"$n punches $mself in the face, looking kind of stupid.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("punch social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("punch social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
