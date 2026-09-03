package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSpackleRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["spackle"]
	if !ok {
		t.Fatal("spackle command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("spackle gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["spackle"]
	if !ok {
		t.Fatal("spackle social is not registered")
	}
	if social.MinLevel != 1 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("spackle social metadata = level %d, hide %d, victim-position %d; want level 1, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You inspect the walls for damage.",
		"$n hauls out $s pail of spackle and looks for a crack to repair.",
		"You thoroughly coat $N with a bumpy sheen of spackle.",
		"$n puts a liberal coating of spackle on $N.",
		"$n spackles you all over.",
		"You throw spackle all over in frustration because $N is not here.",
		"You dump a pail of spackle over your head.",
		"$n thoroughly spackles $mself.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("spackle social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("spackle social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
