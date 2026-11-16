package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1548: C's do_tell (src/act.comm.c:909-930) delivers to a mobile when
// vict->desc != NULL (a switched immortal) and drops unswitched mobiles.
func TestNPCTellReachesSwitchedMobileDescriptor(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 8004, Name: "Temple", Zone: 80}},
		Mobs: []parser.Mob{
			{VNum: 12102, Keywords: "hansen merchant", ShortDesc: "Hansen the merchant", Level: 30},
			{VNum: 1, Keywords: "puff", ShortDesc: "Puff", Level: 10},
			{VNum: 2, Keywords: "guard", ShortDesc: "a guard", Level: 10},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	keeper, err := w.SpawnMob(12102, 8004)
	if err != nil {
		t.Fatalf("SpawnMob keeper: %v", err)
	}
	puff, err := w.SpawnMob(1, 8004)
	if err != nil {
		t.Fatalf("SpawnMob puff: %v", err)
	}
	if _, err := w.SpawnMob(2, 8004); err != nil {
		t.Fatalf("SpawnMob guard: %v", err)
	}

	owner := NewPlayer(1, "Shopswitch", 8004)
	var gotSwitched []string
	var gotKeeper []string
	w.MobileDescriptorOwner = func(m *MobInstance) *Player {
		if m == puff || m == keeper {
			return owner
		}
		return nil
	}
	w.MobileMessageSink = func(m *MobInstance, msg []byte) {
		switch m {
		case puff:
			gotSwitched = append(gotSwitched, string(msg))
		case keeper:
			gotKeeper = append(gotKeeper, string(msg))
		}
	}

	w.NPCTellCommand(keeper, "Puff What do you want me to valuate??")
	if len(gotSwitched) != 1 || gotSwitched[0] != "Hansen the merchant tells you, 'What do you want me to valuate??'\r\n" {
		t.Fatalf("switched mob tell output = %q", gotSwitched)
	}

	gotKeeper = nil
	w.NPCTellCommand(keeper, "guard hello")
	if len(gotKeeper) != 1 || gotKeeper[0] != "No-one by that name here.\r\n" {
		t.Fatalf("unswitched mob tell keeper output = %q", gotKeeper)
	}
}
