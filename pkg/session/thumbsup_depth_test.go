package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestThumbsupRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["thumbsup"]
	if !ok {
		t.Fatal("thumbsup command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("thumbsup gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["thumbsup"]
	if !ok {
		t.Fatal("thumbsup social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("thumbsup social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You seem very happy today.",
		"$n gives everyone a big thumbs up.",
		"You give $N a big thumbs up.",
		"$n gives $N a big thumbs up.",
		"$n gives you a big thumbs up.",
		"You don't see that person.",
		"You feel extremely silly.",
		"$n gives $mself a big thumbs up.  How silly.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("thumbsup social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("thumbsup social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
