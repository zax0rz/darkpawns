package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShishkabobRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["shishkabob"]
	if !ok {
		t.Fatal("shishkabob command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("shishkabob gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["shishkabob"]
	if !ok {
		t.Fatal("shishkabob social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != combat.PosResting || social.MinVictimPosition != 0 {
		t.Fatalf("shishkabob social metadata = hide %d, victim-position %d, override %d; want 0, %d, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition, combat.PosResting)
	}
	wantMessages := []string{
		"You whip out your kabobing tools.",
		"$n whips out $s kabobing tools and prepares lunch.",
		"You shishkabob $M right through $S heart.",
		"$n shishkabobs $N right through $S heart.",
		"$n shishkabobs you right through your heart.",
		"Sorry good chef, but that person doesn't seem to be here.",
		"You stab yourself in the arm.  I hope you're not planning to eat that!",
		"$n kabobs $mself right in the arm... YUCK!!!",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("shishkabob social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("shishkabob social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
