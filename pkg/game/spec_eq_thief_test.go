package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSpecEqThief_EntryGates(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 16)
	lastMsg()

	if specEqThief(w, player, mob, "give", "sword") {
		t.Fatal("eq thief should reject player command dispatch")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("command gate output = %q, want empty", got)
	}

	mob.SetPosition(combat.PosSitting)
	if specEqThief(w, nil, mob, "", "") {
		t.Fatal("eq thief should reject a non-standing mob")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("position gate output = %q, want empty", got)
	}

	mob.SetPosition(combat.PosStanding)
	player.SetLevel(LVL_IMMORT)
	dprng.ResetStream(1)
	if specEqThief(w, nil, mob, "", "") {
		t.Fatal("eq thief should reject an immortal target")
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("immortal target gate output = %q, want empty", got)
	}
}

func TestSpecEqThief_StealsFirstVisibleItemAndPreservesDraws(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 16)
	lastMsg()

	itemProto := &parser.Obj{
		VNum:      3201,
		Keywords:  "test item",
		ShortDesc: "a test item",
		WearFlags: [4]int{1},
	}
	w.mu.Lock()
	w.objs[itemProto.VNum] = itemProto
	w.mu.Unlock()
	item, err := w.SpawnObject(itemProto.VNum, -1)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	if err := w.MoveObjectToPlayerInventory(item, player); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory: %v", err)
	}
	player.SetLevel(1)
	player.SetPosition(combat.PosStunned) // C's GET_POS(victim) < POS_SLEEPING override.

	seed := eqThiefStealSeed(t)
	wantStream := dprng.New(seed)
	wantStream.Number(0, 4)   // outer target selection
	wantStream.Number(0, 60)  // kender item gate
	wantStream.Number(1, 101) // percent draw before position override
	wantStream.Number(50, 100)
	wantNext := wantStream.Number(0, 999)

	dprng.ResetStream(seed)
	if !specEqThief(w, nil, mob, "", "") {
		t.Fatal("eligible eq thief tick should be handled")
	}
	if got := player.GetInventory(); len(got) != 0 {
		t.Fatalf("player inventory after steal = %#v, want empty", got)
	}
	if len(mob.Inventory) != 1 || mob.Inventory[0] != item {
		t.Fatalf("mob inventory after steal = %#v, want stolen item", mob.Inventory)
	}
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("next RNG draw = %d, want %d after eq thief draw sequence", got, wantNext)
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("regular steal output = %q, want empty", got)
	}
}

func TestSpecEqThief_GetAndJunkBlackContainerItem(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 16)
	lastMsg()

	containerProto := &parser.Obj{
		VNum:      3202,
		Keywords:  "leather bag",
		ShortDesc: "a leather bag",
		TypeFlag:  ITEM_CONTAINER,
		Values:    [4]int{100, 0, -1, 0},
	}
	blackProto := &parser.Obj{
		VNum:      3203,
		Keywords:  "black stone",
		ShortDesc: "a black stone",
		WearFlags: [4]int{1},
	}
	w.mu.Lock()
	w.objs[containerProto.VNum] = containerProto
	w.objs[blackProto.VNum] = blackProto
	w.mu.Unlock()
	container, err := w.SpawnObject(containerProto.VNum, -1)
	if err != nil {
		t.Fatalf("SpawnObject container: %v", err)
	}
	black, err := w.SpawnObject(blackProto.VNum, -1)
	if err != nil {
		t.Fatalf("SpawnObject black item: %v", err)
	}
	if err := w.MoveObjectToMobInventory(container, mob); err != nil {
		t.Fatalf("MoveObjectToMobInventory: %v", err)
	}
	if err := w.MoveObjectToContainer(black, container); err != nil {
		t.Fatalf("MoveObjectToContainer: %v", err)
	}

	player.SetLevel(1)
	player.SetPosition(combat.PosStanding)
	seed := eqThiefOuterSuccessSeed(t)
	dprng.ResetStream(seed)
	if !specEqThief(w, nil, mob, "", "") {
		t.Fatal("eligible eq thief tick should be handled")
	}
	if len(container.GetContents()) != 0 {
		t.Fatalf("container contents after black retrieval = %#v, want empty", container.GetContents())
	}
	if len(mob.Inventory) != 1 || mob.Inventory[0] != container {
		t.Fatalf("mob inventory after black junk = %#v, want container only", mob.Inventory)
	}
	if black.Location != LocNowhere() {
		t.Fatalf("junked item location = %#v, want nowhere", black.Location)
	}
	if got := lastMsg(); !strings.Contains(got, "gets a black stone from a leather bag") ||
		!strings.Contains(got, "junks a black stone. It vanishes in a puff of smoke!") {
		t.Fatalf("black container audience = %q", got)
	}
}

func eqThiefStealSeed(t *testing.T) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 10000; seed++ {
		rng := dprng.New(seed)
		if rng.Number(0, 4) != 0 || rng.Number(0, 60) >= 16 {
			continue
		}
		rng.Number(1, 101)
		if -1 < rng.Number(50, 100) {
			return seed
		}
	}
	t.Fatal("could not find eq thief steal seed")
	return 0
}

func eqThiefOuterSuccessSeed(t *testing.T) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 10000; seed++ {
		if dprng.New(seed).Number(0, 4) == 0 {
			return seed
		}
	}
	t.Fatal("could not find eq thief outer success seed")
	return 0
}
