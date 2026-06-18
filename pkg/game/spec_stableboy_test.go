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
			{VNum: horseVnum, Keywords: "horse", ShortDesc: "a horse"},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer w.StopAITicker()

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }

	ch := NewPlayer(1, "Tester", 1001)
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

	// buy without enough gold: rejected.
	ch.SetGold(100)
	specStableboy(w, ch, boy, "buy", "horse")
	if msg := lastMsg(); !strings.Contains(msg, "can't afford") {
		t.Errorf("buy without funds: got %q", msg)
	}
	if ch.GetGold() != 100 {
		t.Errorf("gold should be untouched after rejected buy: got %d", ch.GetGold())
	}

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
}
