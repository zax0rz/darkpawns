package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestMoshRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["mosh"]
	if !ok {
		t.Fatal("mosh command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("mosh gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["mosh"]
	if !ok {
		t.Fatal("mosh social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != combat.PosResting || social.MinVictimPosition != 0 {
		t.Fatalf("mosh social metadata = hide %d, victim position %d, override %d; want hide 0, victim position %d, override 0", social.MinLevel, social.HideFlag, social.MinVictimPosition, combat.PosResting)
	}
	wantMessages := []string{
		"You thrash your head around in circles, damn your neck hurts!",
		"You think $n looks like a silly chicken thrashing about.",
		"You slam your body into $N.",
		"Damn, that really must have hurt $N when $n hit $M with that mosh.",
		"$n slams into you with and incredible mosh.",
		"Mosh who?",
		"You mosh into a wall!",
		"$n looks injured moshing $mself into the wall.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("mosh social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("mosh social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
