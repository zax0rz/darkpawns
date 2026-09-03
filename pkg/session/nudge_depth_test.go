package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestNudgeRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["nudge"]
	if !ok {
		t.Fatal("nudge command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosResting {
		t.Fatalf("nudge gate = (%d, %d), want (0, %d)", entry.MinLevel, entry.MinPosition, combat.PosResting)
	}

	social, ok := game.Socials["nudge"]
	if !ok {
		t.Fatal("nudge social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 0 || social.MinVictimPosition != 0 {
		t.Fatalf("nudge social metadata = level %d, hide %d, min-victim %d; want all zero", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	wantMessages := []string{
		"Nudge?  Nudge???  The HELL you say!!!!",
		"#",
		"You nudge $M with your elbow.",
		"$n nudges $N suggestively with $s elbow.",
		"$n nudges you suggestively.  You two have an understanding.",
		"Eh?  That person isn't here, you know.",
		"Well, just nudge yourself, but how do you get your elbow in that position?",
		"$n nudges $mself with $s elbows, making $m look like a large chicken.",
	}
	if len(social.Messages) != len(wantMessages) {
		t.Fatalf("nudge social has %d messages, want %d", len(social.Messages), len(wantMessages))
	}
	for i, want := range wantMessages {
		if social.Messages[i] != want {
			t.Errorf("nudge social message %d = %q, want %q", i, social.Messages[i], want)
		}
	}
}
