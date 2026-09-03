package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPeerRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["peer"]
	if !ok {
		t.Fatal("peer command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("peer gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["peer"]
	if !ok {
		t.Fatal("peer social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("peer social metadata = level %d, hide %d, min-victim %d; want 1, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You peer around you, uncertain that what you see is actually true.",
		"$n peers around, looking as if $e has trouble seeing everything clearly.",
		"You peer at $M uncertainly.",
		"$n peers at $N uncertainly.",
		"$n peers at you uncertainly.",
		"You peer at someone from a distance.",
		"You peer around you, uncertain that what you see is actually true.",
		"$n peers around, looking as if $e has trouble seeing everything clearly.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("peer social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("peer social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
