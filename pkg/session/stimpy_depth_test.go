package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestStimpyRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["stimpy"]
	if !ok {
		t.Fatal("stimpy command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("stimpy gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["stimpy"]
	if !ok {
		t.Fatal("stimpy social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("stimpy social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Oh! Happy Happy, Joy Joy!",
		"$n jumps up and shouts: \"Oh, Happy Happy, Joy Joy!!\"",
		"You turn to $M and shout: \"Oh, Happy Happy, Joy Joy!\"",
		"$n turns to $N and shouts: \"Oh, Happy Happy, Joy Joy!\"",
		"$n turns to you and shouts: \"Oh, Happy Happy, Joy Joy!\"",
		"Oh! Happy Happy, Joy Joy!",
		"Oh! Happy Happy, Joy Joy!",
		"$n sniffs $mself and says: \"Sttteeenky!!!\"",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("stimpy social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("stimpy social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
