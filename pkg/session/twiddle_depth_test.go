package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestTwiddleRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["twiddle"]
	if !ok {
		t.Fatal("twiddle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("twiddle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["twiddle"]
	if !ok {
		t.Fatal("twiddle social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("twiddle social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You patiently twiddle your thumbs.",
		"$n patiently twiddles $s thumbs.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("twiddle social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("twiddle social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
