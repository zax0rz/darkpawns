package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSneezeRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["sneeze"]
	if !ok {
		t.Fatal("sneeze command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("sneeze gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["sneeze"]
	if !ok {
		t.Fatal("sneeze social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("sneeze social metadata = level %d, hide %d, victim-position %d; want 0/0/0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Gesundheit!",
		"$n sneezes.",
		"You sneeze all over $S face.",
		"$n sneezes all over your face.",
		"$n sneezes all over $N's face.",
		"You missed.",
		"You sneeze all over yourself, ICKY!",
		"Eww! $n just sneezed all over $mself.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("sneeze social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("sneeze social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
