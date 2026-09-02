package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestBowRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["bow"]
	if !ok {
		t.Fatal("bow command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
		t.Fatalf("bow gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosStanding)
	}

	social, ok := game.Socials["bow"]
	if !ok {
		t.Fatal("bow social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 5 || social.MinVictimPosition != 0 {
		t.Fatalf("bow social metadata = hide %d, victim-position %d, override %d; want hide 0, victim-position 5, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You bow deeply.",
		"$n bows deeply.",
		"You bow before $M.",
		"$n bows before $N.",
		"$n bows before you.",
		"Who's that?",
		"You kiss your toes.",
		"$n folds up like a jacknife and kisses $s own toes.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("bow social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("bow social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
