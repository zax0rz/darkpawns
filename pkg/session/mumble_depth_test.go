package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestMumbleRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["mumble"]
	if !ok {
		t.Fatal("mumble command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("mumble gate = (%d, %d), want (0, %d)", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["mumble"]
	if !ok {
		t.Fatal("mumble social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("mumble social metadata = level %d, hide %d, min-victim %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"You mumble.",
		"$n mumbles something under $s breath.",
		"You mumble at $N.",
		"$n mumbles something under $s breath to $N.",
		"$n mumbles something under $s breath to you.",
		"You mumble something to no one in particular.",
		"You mumble to yourself.",
		"$n mumbles something under $s breath to $mself.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("mumble social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("mumble social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
