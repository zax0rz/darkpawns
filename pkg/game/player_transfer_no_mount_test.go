package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestPlayerTransferMountVariant pins the mount split that do_trans relies on:
// PlayerTransfer carries a ridden mount (the single-target branch, and C's
// gate/moon-gate callers), while PlayerTransferNoMount leaves it behind (the
// all-target branch). The contrast makes each assertion able to fail: a
// no-mount implementation that still moved the mount would trip the second
// case, and one that dropped carriers would trip the first.
func TestPlayerTransferMountVariant(t *testing.T) {
	newRider := func(t *testing.T) (*World, *Player, *MobInstance) {
		w, err := NewWorld(&parser.World{
			Rooms: []parser.Room{{VNum: 1001, Name: "Here"}, {VNum: 1002, Name: "There"}},
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
		return w, p, mount
	}

	t.Run("PlayerTransfer carries the mount", func(t *testing.T) {
		w, p, mount := newRider(t)
		if err := w.PlayerTransfer(p, 1002); err != nil {
			t.Fatal(err)
		}
		if got := mount.GetRoomVNum(); got != 1002 {
			t.Fatalf("mount room = %d, want 1002 (PlayerTransfer carries the mount)", got)
		}
	})

	t.Run("PlayerTransferNoMount leaves the mount", func(t *testing.T) {
		w, p, mount := newRider(t)
		if err := w.PlayerTransferNoMount(p, 1002); err != nil {
			t.Fatal(err)
		}
		if got := p.GetRoomVNum(); got != 1002 {
			t.Fatalf("rider room = %d, want 1002", got)
		}
		if got := mount.GetRoomVNum(); got != 1001 {
			t.Fatalf("mount room = %d, want 1001 (the no-mount variant leaves it behind)", got)
		}
	})
}
