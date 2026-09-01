package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPokeRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["poke"]
	if !ok {
		t.Fatal("poke command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("poke gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["poke"]
	if !ok {
		t.Fatal("poke social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("poke social metadata = level %d, hide %d, victim-position %d; want 0, 0, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Poke who??",
		"#",
		"You poke $M in the ribs.",
		"$n pokes $N in the ribs.",
		"$n pokes you in the ribs.",
		"You can't poke someone who's not here!",
		"You poke yourself in the ribs, feeling very silly.",
		"$n pokes $mself in the ribs, looking very sheepish.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("poke social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("poke social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
