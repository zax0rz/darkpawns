package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
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

func TestLuaMountMobileBodyPreservesCMountState(t *testing.T) {
	w, owner, horse := newRideDepthWorld(t)
	other := NewPlayer(2, "Other", owner.GetRoom())
	other.SetPosition(8)
	if err := w.AddPlayer(other); err != nil {
		t.Fatalf("AddPlayer other: %v", err)
	}
	output := captureMovementOutput(w)

	owner.SetAffect(affMounted, true)
	owner.MountName = horse.GetName()
	horse.SetAffected(affMounted)
	horse.SetAffected(affCharm)
	horse.SetMountRider(owner.Name)
	horse.SetFollowingBody(owner)
	adapter := NewWorldScriptableAdapter(w)

	adapter.Mount(scripting.CharRef{NPC: true, ID: horse.GetID()}, nil, "unmount")
	if horse.IsAffected(affMounted) || !horse.IsAffected(affCharm) {
		t.Fatalf("mobile mount affect state: mounted=%t charmed=%t", horse.IsAffected(affMounted), horse.IsAffected(affCharm))
	}
	if horse.GetMountRider() != owner.Name || horse.GetFollowing() != owner.Name || w.FollowingBody(horse) != owner {
		t.Fatalf("C unmount changed mobile master/rider link: rider=%q following=%q body=%p", horse.GetMountRider(), horse.GetFollowing(), w.FollowingBody(horse))
	}
	if !owner.IsAffected(affMounted) || owner.GetMountName() != horse.GetName() {
		t.Fatalf("unmounting the mobile changed its player's rider state: affected=%t mount=%q", owner.IsAffected(affMounted), owner.GetMountName())
	}
	if w.GetMount(owner) != nil || w.GetRider(horse) != nil {
		t.Fatal("C get_mount/get_rider exposed an affect-cleared mobile")
	}

	w.doRide(other, nil, "ride", "horse")
	if got := output[other.Name].String(); got != "Its master would not like that!\r\n" {
		t.Fatalf("second rider output after Lua unmount = %q", got)
	}
}

func TestLuaMountMobileRideAndDismountUseActualActors(t *testing.T) {
	w, rider, horse := newRideDepthWorld(t)
	observer := NewPlayer(2, "Observer", rider.GetRoom())
	observer.SetPosition(8)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	child := NewMob(&parser.Mob{
		VNum:        9002,
		Keywords:    "pony",
		ShortDesc:   "a pony",
		ActionFlags: []string{"mountable"},
	}, rider.GetRoom())
	w.mu.Lock()
	child.ID = w.nextMobID
	w.nextMobID++
	w.activeMobs[child.ID] = child
	w.mu.Unlock()
	output := captureMovementOutput(w)
	adapter := NewWorldScriptableAdapter(w)

	adapter.Mount(scripting.CharRef{NPC: true, ID: horse.GetID()}, nil, "dismount")
	if horse.IsAffected(affMounted) || (output[observer.Name] != nil && output[observer.Name].Len() != 0) {
		t.Fatal("unmounted mobile dismount changed state or emitted room bytes")
	}
	adapter.Mount(scripting.CharRef{NPC: true, ID: horse.GetID()}, charRefFor(child), "ride")
	if horse.IsAffected(affMounted) || child.IsAffected(affMounted) || child.GetFollowing() != "" {
		t.Fatal("C CAN_MOUNT mobile-rider gate created a mount or follower link")
	}

	horse.SetAffected(affMounted)
	horse.SetFollowingBody(child)
	child.SetAffected(affMounted)
	adapter.Mount(*charRefFor(child), nil, "dismount")
	if child.IsAffected(affMounted) || horse.IsAffected(affMounted) {
		t.Fatalf("mobile dismount left AFF_MOUNT: rider=%t mount=%t", child.IsAffected(affMounted), horse.IsAffected(affMounted))
	}
	if child.GetFollowing() != "" || horse.GetFollowing() != child.GetName() || w.FollowingBody(horse) != child {
		t.Fatalf("C unmount changed independent master links: child=%q horse=%q body=%p", child.GetFollowing(), horse.GetFollowing(), w.FollowingBody(horse))
	}
	if got := output[observer.Name].String(); got != "A pony dismounts from the back of a horse.\r\n" {
		t.Fatalf("room observer after mobile dismount = %q", got)
	}
}

func TestLuaMountPlayerRideDismountAndUnmountUseBridgeBody(t *testing.T) {
	w, rider, mount := newRideDepthWorld(t)
	observer := NewPlayer(2, "Observer", rider.GetRoom())
	observer.SetPosition(8)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	output := captureMovementOutput(w)
	adapter := NewWorldScriptableAdapter(w)
	riderRef := scripting.CharRef{ID: rider.ID}
	mountRef := charRefFor(mount)

	adapter.Mount(riderRef, mountRef, "ride")
	if !rider.IsAffected(affMounted) || !mount.IsAffected(affMounted) {
		t.Fatal("Lua player ride did not set C mount affects")
	}
	if got := output[rider.Name].String(); got != "You hop on your mount.\r\n" {
		t.Fatalf("Lua player ride output = %q", got)
	}

	output[rider.Name].Reset()
	output[observer.Name].Reset()
	adapter.Mount(riderRef, nil, "dismount")
	if rider.IsAffected(affMounted) || mount.IsAffected(affMounted) || rider.GetMountName() != "" {
		t.Fatalf("Lua player dismount state: rider=%t mount=%t name=%q", rider.IsAffected(affMounted), mount.IsAffected(affMounted), rider.GetMountName())
	}
	if mount.GetMountRider() != rider.Name || mount.GetFollowing() != rider.Name {
		t.Fatalf("Lua player dismount removed C master link: rider=%q following=%q", mount.GetMountRider(), mount.GetFollowing())
	}
	if got := output[rider.Name].String(); got != "You hop off your mount.\r\n" {
		t.Fatalf("Lua player dismount output = %q", got)
	}
	if got := output[observer.Name].String(); got != "Rider dismounts from the back of a horse.\r\n" {
		t.Fatalf("Lua player dismount room output = %q", got)
	}

	// C's lua_mount("unmount") calls unmount(rider, get_mount(rider)); it
	// clears AFF_MOUNT without removing the follower/master relationship.
	newer := NewMob(&parser.Mob{
		VNum:        9003,
		Keywords:    "pony",
		ShortDesc:   "a pony",
		ActionFlags: []string{"mountable"},
	}, rider.GetRoom())
	w.mu.Lock()
	newer.ID = w.nextMobID
	w.nextMobID++
	w.activeMobs[newer.ID] = newer
	w.mu.Unlock()
	rider.SetAffect(affMounted, true)
	mount.SetAffected(affMounted)
	newer.SetAffected(affMounted)
	newer.SetMountRider(rider.Name)
	newer.SetFollowingBody(rider) // newer head of C's prepended follower list
	output[rider.Name].Reset()
	adapter.Mount(riderRef, nil, "unmount")
	if rider.IsAffected(affMounted) || newer.IsAffected(affMounted) || !mount.IsAffected(affMounted) {
		t.Fatalf("Lua player unmount cleared wrong follower: rider=%t older=%t newer=%t", rider.IsAffected(affMounted), mount.IsAffected(affMounted), newer.IsAffected(affMounted))
	}
	if mount.GetMountRider() != rider.Name || mount.GetFollowing() != rider.Name || newer.GetMountRider() != rider.Name || newer.GetFollowing() != rider.Name {
		t.Fatal("Lua player unmount removed a retained follower/master link")
	}
}

func TestDoRideDirectGates(t *testing.T) {
	t.Run("fighting", func(t *testing.T) {
		w, rider, _ := newRideDepthWorld(t)
		output := captureMovementOutput(w)
		rider.SetFightingBody(NewPlayer(99999, "an enemy", 1001))

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
		{
			p := *mount.Proto()
			p.ActionFlags = nil
			setProtoWithFlags(mount, &p)
		}

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
