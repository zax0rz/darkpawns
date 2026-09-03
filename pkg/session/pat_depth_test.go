package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPatRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["pat"]
	if !ok {
		t.Fatal("pat command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("pat gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["pat"]
	if !ok {
		t.Fatal("pat social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("pat social metadata = level %d, hide %d, min-victim %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Pat who??",
		"#",
		"You pat $N on $S head.",
		"$n pats $N on $S head.",
		"$n pats you on your head.",
		"Who, where, what??",
		"You pat yourself on your head, very reassuring.",
		"$n pats $mself on the head.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("pat social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("pat social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
