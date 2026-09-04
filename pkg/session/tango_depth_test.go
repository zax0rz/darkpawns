package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestTangoRegistrationUsesCEntryGateAndRecord(t *testing.T) {
	entry, ok := commandGates["tango"]
	if !ok {
		t.Fatal("tango command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosStanding {
		t.Fatalf("tango gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosStanding)
	}

	social, ok := game.Socials["tango"]
	if !ok {
		t.Fatal("tango social is not registered")
	}
	if social.MinLevel != 0 || social.HideFlag != 8 || social.MinVictimPosition != 0 {
		t.Fatalf("tango social metadata = level %d, hide %d, min-victim %d; want 0, 8, 0", social.MinLevel, social.HideFlag, social.MinVictimPosition)
	}
	want := []string{
		"With whom would you like to tango?",
		"$n puts a rose between $s teeth, but takes out it since noone joins $m.",
		"You put a rose between your teeth and tango with $M seductively.",
		"$n puts a rose between $s teeth and tangos with $N seductively.",
		"$n puts a rose between $s teeth and tangos with you seductively.",
		"That person isn't around.  Better sit this one out.",
		"Feels rather stupid, doesn't it?",
		"$n puts a rose between $s teeth and tries to tango with $mself.",
	}
	if len(social.Messages) != len(want) {
		t.Fatalf("tango social has %d messages, want %d", len(social.Messages), len(want))
	}
	for i, message := range want {
		if social.Messages[i] != message {
			t.Errorf("tango social message %d = %q, want %q", i, social.Messages[i], message)
		}
	}
}
