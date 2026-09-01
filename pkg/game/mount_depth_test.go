package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newRideDepthWorld(t *testing.T) (*World, *Player, *MobInstance) {
	t.Helper()

	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Open Field", Zone: 1}},
		Mobs:  []parser.Mob{},
		Objs:  []parser.Obj{},
	})
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	rider := NewPlayer(1, "Rider", 1001)
	rider.SetPosition(8)
	if err := w.AddPlayer(rider); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	mount := NewMob(&parser.Mob{
		VNum:        9001,
		Keywords:    "horse pony",
		ShortDesc:   "a horse",
		ActionFlags: []string{"mountable"},
	}, 1001)
	w.mu.Lock()
	mount.ID = w.nextMobID
	w.nextMobID++
	w.activeMobs[mount.ID] = mount
	w.mu.Unlock()
	return w, rider, mount
}

func TestDoRideDirectGates(t *testing.T) {
	t.Run("fighting", func(t *testing.T) {
		w, rider, _ := newRideDepthWorld(t)
		output := captureMovementOutput(w)
		rider.SetFighting("an enemy")

		w.doRide(rider, nil, "mount", "horse")

		if got := output[rider.Name].String(); got != "You're too busy fighting!\n\r" {
			t.Fatalf("fighting output = %q", got)
		}
	})

	t.Run("indoors", func(t *testing.T) {
		w, rider, _ := newRideDepthWorld(t)
		output := captureMovementOutput(w)
		w.GetRoomInWorld(rider.GetRoomVNum()).Flags = []string{"8"}

		w.doRide(rider, nil, "mount", "horse")

		if got := output[rider.Name].String(); got != "Go outside if you want to ride!\r\n" {
			t.Fatalf("indoors output = %q", got)
		}
	})

	t.Run("actor charm", func(t *testing.T) {
		w, rider, _ := newRideDepthWorld(t)
		output := captureMovementOutput(w)
		rider.SetAffect(affCharm, true)

		w.doRide(rider, nil, "mount", "horse")

		if got := output[rider.Name].String(); got != "Get your master's permission first!\r\n" {
			t.Fatalf("actor charm output = %q", got)
		}
	})

	t.Run("mount charm", func(t *testing.T) {
		w, rider, mount := newRideDepthWorld(t)
		output := captureMovementOutput(w)
		mount.SetAffected(affCharm)
		mount.SetFollowing("SomebodyElse")

		w.doRide(rider, nil, "mount", "horse")

		if got := output[rider.Name].String(); got != "Its master would not like that!\r\n" {
			t.Fatalf("mount charm output = %q", got)
		}
	})

	t.Run("not mountable", func(t *testing.T) {
		w, rider, mount := newRideDepthWorld(t)
		output := captureMovementOutput(w)
		mount.Prototype.ActionFlags = nil

		w.doRide(rider, nil, "mount", "horse")

		if got := output[rider.Name].String(); got != "You can't ride a horse!\r\n" {
			t.Fatalf("not-mountable output = %q", got)
		}
	})
}

func TestDoRideSuccessStateAndAudience(t *testing.T) {
	w, rider, mount := newRideDepthWorld(t)
	observer := NewPlayer(2, "Observer", 1001)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer failed: %v", err)
	}
	output := captureMovementOutput(w)

	w.doRide(rider, nil, "mount", "horse")

	if !rider.IsAffected(affMounted) || rider.MountName != mount.GetName() {
		t.Fatalf("rider mount state = affected %v, name %q", rider.IsAffected(affMounted), rider.MountName)
	}
	if !mount.IsAffected(affMounted) || !mount.IsAffected(affCharm) || mount.GetMountRider() != rider.Name || mount.GetFollowing() != rider.Name {
		t.Fatalf("mount state = mounted %v, charmed %v, rider %q, following %q", mount.IsAffected(affMounted), mount.IsAffected(affCharm), mount.GetMountRider(), mount.GetFollowing())
	}
	if got := output[rider.Name].String(); got != "You hop on your mount.\r\n" {
		t.Fatalf("rider output = %q", got)
	}
	if got := output[observer.Name].String(); got != "Rider hops onto the back of a horse.\r\n" {
		t.Fatalf("observer output = %q", got)
	}
}

func TestDoRideMountedGates(t *testing.T) {
	t.Run("actor already mounted", func(t *testing.T) {
		w, rider, _ := newRideDepthWorld(t)
		output := captureMovementOutput(w)
		rider.SetAffect(affMounted, true)

		w.doRide(rider, nil, "mount", "horse")

		if got := output[rider.Name].String(); got != "You can't ride two beasts at once!\r\n" {
			t.Fatalf("actor-mounted output = %q", got)
		}
	})

	t.Run("mount already ridden", func(t *testing.T) {
		w, rider, mount := newRideDepthWorld(t)
		output := captureMovementOutput(w)
		mount.SetAffected(affMounted)
		mount.SetMountRider("SomebodyElse")

		w.doRide(rider, nil, "mount", "horse")

		if got := output[rider.Name].String(); got != "The beast is already being ridden!\r\n" {
			t.Fatalf("mount-ridden output = %q", got)
		}
	})
}

func TestDoRideUsesCOneArgument(t *testing.T) {
	w, rider, mount := newRideDepthWorld(t)
	output := captureMovementOutput(w)

	w.doRide(rider, nil, "mount", "horse trailing words")

	if !rider.IsAffected(affMounted) || mount.GetMountRider() != rider.Name {
		t.Fatal("one-argument target did not mount the horse")
	}
	if output[rider.Name].String() != "You hop on your mount.\r\n" {
		t.Fatalf("one-argument output = %q", output[rider.Name].String())
	}
}
