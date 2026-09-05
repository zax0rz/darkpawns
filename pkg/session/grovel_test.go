package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestGrovelRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["grovel"]
	if !ok {
		t.Fatal("grovel command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("grovel gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["grovel"]
	if !ok {
		t.Fatal("grovel social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("grovel social metadata = hide %d, min-position %d, override %d; want 1, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You grovel in the dirt.",
		"$n grovels in the dirt.",
		"You grovel before $M.",
		"$n grovels in the dirt before $N.",
		"$n grovels in the dirt before you.",
		"Who?",
		"That seems a little silly to me..",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("grovel social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("grovel social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
