package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// C moon_gate moves a mounted player's mount through the portal with its
// rider (src/gate.c:282-290). PlayerTransfer carries the mount; this pins it.
func TestMoonGateCarriesMount(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Grove"}, {VNum: 4000, Name: "Beyond"}},
		Mobs:  []parser.Mob{{VNum: 2001, ShortDesc: "a horse", Level: 5}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	p := NewPlayer(1, "Rider", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	mount, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	p.MountName = mount.GetName()
	p.SetAffect(affMounted, true)
	mount.SetMountRider(p.Name)
	mount.SetFollowing(p.Name)
	mount.SetAffected(affMounted)

	// A red portal (not blue, cost 0) leads to room 4000 (src/gate.c:268-271).
	portal := w.NewObjectFromProto(&parser.Obj{VNum: RedPortalVNum, Keywords: "portal gate", ShortDesc: "a red portal"}, 1001)
	if !specMoonGate(w, p, portal, "enter", "portal") {
		t.Fatal("enter portal was not handled")
	}
	if p.GetRoom() != 4000 {
		t.Fatalf("rider room = %d, want 4000", p.GetRoom())
	}
	if mount.GetRoomVNum() != 4000 {
		t.Fatalf("mount room = %d, want 4000 (C moves the mount with its rider)", mount.GetRoomVNum())
	}
}
