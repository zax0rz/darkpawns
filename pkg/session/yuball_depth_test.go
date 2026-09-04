package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestYuballRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["yuball"]
	if !ok {
		t.Fatal("yuball command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("yuball gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["yuball"]
	if !ok {
		t.Fatal("yuball social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("yuball social metadata = C-hide %d, C-min-victim %d, override %d; want 0, 5, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"You teach everyone how to do the \"Yub! Yub!\" song and dance.",
		"$n teaches you how to do the \"Yub! Yub!\" song and dance.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("yuball social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("yuball social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
