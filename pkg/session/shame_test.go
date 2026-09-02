package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShameRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["shame"]
	if !ok {
		t.Fatal("shame command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("shame gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["shame"]
	if !ok {
		t.Fatal("shame social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("shame social metadata = hide %d, victim-position %d, override %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You spin around shaming everyone.",
		"$n spins around the room shaming everyone.",
		"You shame $N.",
		"$n shames $N.",
		"$n shames you.",
		"Who?!?",
		"You shame yourself.",
		"$n shames $mself.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("shame social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("shame social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
