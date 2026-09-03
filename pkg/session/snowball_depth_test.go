package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSnowballRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	gate, ok := commandGates["snowball"]
	if !ok {
		t.Fatal("snowball command has no C gate")
	}
	if gate.MinLevel != game.LVL_IMMORT || gate.MinPosition != combat.PosStanding {
		t.Fatalf("snowball gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, game.LVL_IMMORT, combat.PosStanding)
	}

	social, ok := game.Socials["snowball"]
	if !ok {
		t.Fatal("snowball social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("snowball social metadata = level %d hide %d victim-position %d, want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"Who do you want to throw a snowball at??",
		"#",
		"You throw a snowball in $N's face.",
		"$n conjures a snowball from thin air and throws it at $N.",
		"$n conjures a snowball from thin air and throws it at you.",
		"You stand with the snowball in your hand because your victim is not here.",
		"You conjure a snowball from thin air and throw it at yourself.",
		"$n conjures a snowball out of the thin air and throws it at $mself.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("snowball social messages = %d, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("snowball social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
