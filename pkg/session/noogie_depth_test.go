package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestNoogieRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["noogie"]
	if !ok {
		t.Fatal("noogie command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("noogie gate = (%d, %d), want (0, %d)", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["noogie"]
	if !ok {
		t.Fatal("noogie social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("noogie social metadata = level %d, hide %d, min-victim %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Generally you noogie SOMEONE.",
		"#",
		"You give $M a good noogie.",
		"$n grabs $N in a headlock and noogies $M.",
		"$n gives you a rough noogie.",
		"Noogie who?",
		"You nearly break your arm in the effort.",
		"$n nearly breaks $s arm trying to give $mself a noogie.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("noogie social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("noogie social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
