package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWinkRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["wink"]
	if !ok {
		t.Fatal("wink command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("wink gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["wink"]
	if !ok {
		t.Fatal("wink social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("wink social metadata = C-hide %d, C-min-victim %d, override %d; want 0, 5, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Have you got something in your eye?",
		"$n winks suggestively.",
		"You wink suggestively at $N.",
		"$n winks at $N.",
		"$n winks suggestively at you.",
		"No one with that name is present.",
		"You wink at yourself?? -- what are you up to?",
		"$n winks at $mself -- something strange is going on...",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("wink social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("wink social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
