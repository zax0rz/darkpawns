package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestStrutRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["strut"]
	if !ok {
		t.Fatal("strut command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
		t.Fatalf("strut gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosStanding)
	}

	social, ok := game.Socials["strut"]
	if !ok {
		t.Fatal("strut social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("strut social metadata = level %d, hide %d, min-victim %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Strut your stuff.",
		"$n struts proudly.",
		"#",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("strut social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("strut social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
