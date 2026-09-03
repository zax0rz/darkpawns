package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestMuhahaRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["muhaha"]
	if !ok {
		t.Fatal("muhaha command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("muhaha gate = (%d, %d), want (0, %d)", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["muhaha"]
	if !ok {
		t.Fatal("muhaha social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("muhaha social metadata = level %d, hide %d, min-victim %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"MUHAHAHAHA!!!!!!!",
		"$n throws $s head back and laughs with draconian terror.",
		"#",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("muhaha social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("muhaha social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
