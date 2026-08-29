package game

import (
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestSpecStableboy verifies the stableboy spec proc's buy/stable/collect
// cycle, matching src/spec_procs2.c SPECIAL(stableboy): buying a horse costs
// 300 gold and adds a charmed follower; stabling it removes the follower and
// starts a per-day rent meter; collecting it later charges
// MountCostDay * days (minimum 1 day) and restores the follower.
func TestSpecStableboy(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Stables", Zone: 1}},
		Mobs: []parser.Mob{
			{VNum: 8022, Keywords: "stableboy", ShortDesc: "the stableboy"},
			{VNum: horseVnum, Keywords: "horse", ShortDesc: "a horse", ActionFlags: []string{"mountable"}},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer w.StopAITicker()

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }

	ch := NewPlayer(1, "Tester", 1001)
	ch.Stats.Cha = 10
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	proto, _ := w.GetMobPrototype(8022)
	boy := NewMobInstance(proto, 1001)

	lastMsg := func() string { s := out.String(); out.Reset(); return s }

	// list: flavor only, no state change.
	specStableboy(w, ch, boy, "list", "")
	if msg := lastMsg(); !strings.Contains(msg, "300 gold coins") {
		t.Errorf("list: got %q", msg)
	}

	// C requires the complete argument to be exactly "horse" before it
	// reaches the follower-cap and affordability gates.
	specStableboy(w, ch, boy, "buy", "horseman")
	if msg := lastMsg(); !strings.Contains(msg, "Buy what") {
		t.Errorf("buy with non-exact horse argument: got %q", msg)
	}

	// buy without enough gold: rejected.
	ch.SetGold(100)
	specStableboy(w, ch, boy, "buy", "horse")
	if msg := lastMsg(); !strings.Contains(msg, "can't afford") {
		t.Errorf("buy without funds: got %q", msg)
	}
	if ch.GetGold() != 100 {
		t.Errorf("gold should be untouched after rejected buy: got %d", ch.GetGold())
	}

	// C checks the follower cap before gold. With CHA 2, one existing follower
	// is already enough to block the purchase at GET_CHA(ch)/2.
	ch.Stats.Cha = 2
	follower := NewPlayer(2, "Follower", 1001)
	if err := w.AddPlayer(follower); err != nil {
		t.Fatalf("AddPlayer follower: %v", err)
	}
	AddFollowerQuiet(follower, ch)
	ch.SetGold(300)
	specStableboy(w, ch, boy, "buy", "horse")
	if msg := lastMsg(); msg != "You can't have any more followers!\r\n" {
		t.Errorf("buy at follower cap: got %q", msg)
	}
	follower.SetFollowing("")
	ch.Stats.Cha = 10

	// buy with enough gold: 300 deducted, charmed horse follows.
	ch.SetGold(300)
	specStableboy(w, ch, boy, "buy", "horse")
	if ch.GetGold() != 0 {
		t.Fatalf("gold after buying horse: got %d, want 0", ch.GetGold())
	}
	var horse *MobInstance
	for _, m := range w.GetMobsInRoom(ch.GetRoom()) {
		if m.GetFollowing() == ch.Name {
			horse = m
		}
	}
	if horse == nil {
		t.Fatalf("expected a charmed horse following the player after buy")
	}
	if !horse.IsAffected(affCharm) {
		t.Errorf("purchased horse should be charmed")
	}

	// stable: rent meter starts, horse leaves the world.
	specStableboy(w, ch, boy, "stable", "")
	if ch.MountVNum != horseVnum || ch.MountCostDay != 5 || ch.MountRentTime == 0 {
		t.Fatalf("after stable: MountVNum=%d MountCostDay=%d MountRentTime=%d", ch.MountVNum, ch.MountCostDay, ch.MountRentTime)
	}
	stillThere := false
	for _, m := range w.GetMobsInRoom(ch.GetRoom()) {
		if m.GetFollowing() == ch.Name {
			stillThere = true
		}
	}
	if stillThere {
		t.Errorf("stabled horse should be removed from the room")
	}

	// stable again with no mount: rejected.
	specStableboy(w, ch, boy, "stable", "")
	if msg := lastMsg(); !strings.Contains(msg, "don't have a mount") {
		t.Errorf("stable with no mount: got %q", msg)
	}

	// collect: back-date the rent clock to simulate elapsed days, then verify
	// cost = MountCostDay * days and the gold/follower/mount-state changes.
	ch.MountRentTime = time.Now().Unix() - 3*86400 // 3 days elapsed
	ch.SetGold(0)
	specStableboy(w, ch, boy, "collect", "")
	if msg := lastMsg(); !strings.Contains(msg, "can't afford the 15 gold") {
		t.Errorf("collect without funds: got %q", msg)
	}
	if ch.MountVNum != horseVnum || ch.MountCostDay != 5 || ch.MountRentTime == 0 {
		t.Errorf("failed collect should preserve mount state: VNum=%d CostDay=%d RentTime=%d", ch.MountVNum, ch.MountCostDay, ch.MountRentTime)
	}

	ch.SetGold(100)
	specStableboy(w, ch, boy, "collect", "")
	wantCost := 5 * 3
	if ch.GetGold() != 100-wantCost {
		t.Fatalf("gold after collect: got %d, want %d (cost=%d)", ch.GetGold(), 100-wantCost, wantCost)
	}
	if ch.MountVNum != 0 || ch.MountCostDay != 0 || ch.MountRentTime != 0 {
		t.Errorf("mount state should be cleared after collect: VNum=%d CostDay=%d RentTime=%d", ch.MountVNum, ch.MountCostDay, ch.MountRentTime)
	}
	returned := false
	for _, m := range w.GetMobsInRoom(ch.GetRoom()) {
		if m.GetFollowing() == ch.Name && m.IsAffected(affCharm) {
			returned = true
		}
	}
	if !returned {
		t.Errorf("expected the collected horse to return as a charmed follower")
	}

	// collect again with no stabled mount: rejected.
	specStableboy(w, ch, boy, "collect", "")
	if msg := lastMsg(); !strings.Contains(msg, "need to have stabled a mount") {
		t.Errorf("collect with no mount: got %q", msg)
	}

	// A missing stored prototype reaches C's read_mobile failure branch and
	// leaves the rent state and gold untouched.
	ch.MountVNum = 99999
	ch.MountCostDay = 5
	ch.MountRentTime = time.Now().Unix()
	ch.SetGold(100)
	specStableboy(w, ch, boy, "collect", "")
	if msg := lastMsg(); !strings.Contains(msg, "unable to gather your mount") {
		t.Errorf("collect with missing prototype: got %q", msg)
	}
	if ch.MountVNum != 99999 || ch.GetGold() != 100 {
		t.Errorf("failed collect mutated state: VNum=%d gold=%d", ch.MountVNum, ch.GetGold())
	}

	// Likewise, a missing horse prototype reaches the buy failure branch only
	// after the exact argument, follower-cap, and gold gates.
	delete(w.mobs, horseVnum)
	ch.MountVNum = 0
	ch.MountCostDay = 0
	ch.MountRentTime = 0
	ch.SetGold(300)
	specStableboy(w, ch, boy, "buy", "horse")
	if msg := lastMsg(); !strings.Contains(msg, "all out of mounts") {
		t.Errorf("buy with missing prototype: got %q", msg)
	}
	if ch.GetGold() != 300 {
		t.Errorf("failed buy mutated gold: got %d", ch.GetGold())
	}
}

func newStableboyBranchWorld(t *testing.T) (*World, *Player, *MobInstance, *strings.Builder) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Stables", Zone: 1}},
		Mobs: []parser.Mob{
			{VNum: 8022, Keywords: "stableboy", ShortDesc: "the stableboy"},
			{VNum: horseVnum, Keywords: "horse", ShortDesc: "a horse", ActionFlags: []string{"mountable"}},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	out := &strings.Builder{}
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }

	ch := NewPlayer(1, "Tester", 1001)
	ch.Stats.Cha = 10
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	proto, _ := w.GetMobPrototype(8022)
	return w, ch, NewMobInstance(proto, 1001), out
}

func TestSpecStableboyAutonomousEntry(t *testing.T) {
	w, _, boy, _ := newStableboyBranchWorld(t)
	if specStableboy(w, nil, boy, "", "") {
		t.Fatal("stableboy should fall through on the autonomous no-command call")
	}
}

func TestSpecStableboyUsesAnyUnmountedMountableFollower(t *testing.T) {
	w, ch, boy, _ := newStableboyBranchWorld(t)
	horse, err := w.SpawnMob(horseVnum, ch.GetRoom())
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	horse.SetFollowing(ch.Name)

	if !specStableboy(w, ch, boy, "stable", "") {
		t.Fatal("stableboy should consume stable for a mountable follower")
	}
	if ch.MountVNum != horseVnum {
		t.Fatalf("mountable follower VNum = %d, want %d", ch.MountVNum, horseVnum)
	}
}

func TestSpecStableboyMountedStableClearsMountAndShowsFollowerStop(t *testing.T) {
	w, ch, boy, out := newStableboyBranchWorld(t)
	horse, err := w.SpawnMob(horseVnum, ch.GetRoom())
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	out.Reset()
	horse.SetAffected(affCharm)
	horse.SetAffected(affMounted)
	horse.SetFollowing(ch.Name)
	horse.SetMountRider(ch.Name)
	ch.MountName = horse.GetName()
	ch.SetAffect(affMounted, true)

	if !specStableboy(w, ch, boy, "stable", "") {
		t.Fatal("stableboy should consume mounted stable")
	}
	if ch.IsMounted() || horse.IsMountedMob() || horse.IsAffected(affMounted) {
		t.Fatalf("mounted state not cleared: player=%v mob=%v affect=%v", ch.IsMounted(), horse.IsMountedMob(), horse.IsAffected(affMounted))
	}
	if !strings.Contains(out.String(), "A horse stops following you.\r\n") {
		t.Errorf("mounted stable output = %q", out.String())
	}
}
