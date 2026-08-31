package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestGropeRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["grope"]
	if !ok {
		t.Fatal("grope command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("grope gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["grope"]
	if !ok {
		t.Fatal("grope social is not registered")
	}
	if social.HideFlag != 5 || social.MinLevel != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("grope social metadata = hide %d, min-level %d, min-victim %d; want 5, 0, 0", social.HideFlag, social.MinLevel, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Whom do you wish to grope??",
		"#",
		"Well, what sort of noise do you expect here?",
		"$n gropes $N.",
		"$n gropes you.",
		"Try someone who's here.",
		"You grope yourself -- YUCK.",
		"$n gropes $mself -- YUCK.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("grope social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("grope social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
