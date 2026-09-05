package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

const butlerTestRoom = 1001

func newButlerTestWorld(t *testing.T) (*World, *Player, *Player, *MobInstance, map[string]string) {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: butlerTestRoom, Name: "Butler Test Room", Zone: 1}},
		Objs: []parser.Obj{
			{
				VNum: 3000, Keywords: "case display", ShortDesc: "an armor display case",
				LongDesc: "An armor display case is here.", TypeFlag: ITEM_CONTAINER,
				Values: [4]int{100, contCloseable, -1, 0},
			},
			{
				VNum: 3001, Keywords: "cabinet weapons", ShortDesc: "a golden weapons cabinet",
				LongDesc: "A golden weapons cabinet is here.", TypeFlag: ITEM_CONTAINER,
				Values: [4]int{100, contCloseable, -1, 0},
			},
			{
				VNum: 3002, Keywords: "chest runed", ShortDesc: "a runed chest",
				LongDesc: "A runed chest is here.", TypeFlag: ITEM_CONTAINER,
				Values: [4]int{100, contCloseable, -1, 0},
			},
			{
				VNum: 3003, Keywords: "tunic armor", ShortDesc: "a frayed tunic",
				LongDesc: "A frayed tunic lies here.", TypeFlag: ITEM_ARMOR,
				WearFlags: [4]int{1}, Weight: 1,
			},
			{
				VNum: 3004, Keywords: "sword short", ShortDesc: "a short sword",
				LongDesc: "A short sword lies here.", TypeFlag: ITEM_WEAPON,
				WearFlags: [4]int{1}, Weight: 1,
			},
			{
				VNum: 3005, Keywords: "bread loaf", ShortDesc: "a loaf of bread",
				LongDesc: "A loaf of bread lies here.", TypeFlag: ITEM_FOOD,
				WearFlags: [4]int{1}, Weight: 1,
			},
			{
				VNum: 3006, Keywords: "board plank", ShortDesc: "a wooden board",
				LongDesc: "A wooden board lies here.", TypeFlag: ITEM_OTHER,
				WearFlags: [4]int{1}, Weight: 1,
			},
		},
		Mobs: []parser.Mob{{
			VNum:        8092,
			Keywords:    "servant butler",
			ShortDesc:   "the Servant",
			LongDesc:    "The Servant is here.",
			Level:       10,
			Str:         18,
			Dex:         18,
			Position:    combat.PosStanding,
			DefaultPos:  combat.PosStanding,
			ActionFlags: []string{"SPEC"},
			HP:          parser.DiceRoll{Num: 1, Sides: 8, Plus: 20},
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }

	actor := NewPlayer(1, "ButlerActor", butlerTestRoom)
	peer := NewPlayer(2, "ButlerPeer", butlerTestRoom)
	for _, player := range []*Player{actor, peer} {
		player.SetPosition(combat.PosStanding)
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.Name, err)
		}
	}
	mob, err := w.SpawnMob(8092, butlerTestRoom)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	mob.SetPosition(combat.PosStanding)
	clearButlerMessages(messages)
	return w, actor, peer, mob, messages
}

func spawnButlerTestObject(t *testing.T, w *World, vnum int) *ObjectInstance {
	t.Helper()
	obj, err := w.SpawnObject(vnum, -1)
	if err != nil {
		t.Fatalf("SpawnObject(%d): %v", vnum, err)
	}
	w.AddItemToRoom(obj, butlerTestRoom)
	return obj
}

func setupButlerContainers(t *testing.T, w *World) (*ObjectInstance, *ObjectInstance, *ObjectInstance) {
	t.Helper()
	return spawnButlerTestObject(t, w, 3000),
		spawnButlerTestObject(t, w, 3001),
		spawnButlerTestObject(t, w, 3002)
}

func clearButlerMessages(messages map[string]string) {
	for name := range messages {
		messages[name] = ""
	}
}

func TestSpecButler_EntryAndContainerGates(t *testing.T) {
	t.Run("command", func(t *testing.T) {
		w, actor, _, mob, messages := newButlerTestWorld(t)
		setupButlerContainers(t, w)
		if got := specButler(w, actor, mob, "look", ""); got {
			t.Fatal("command-bearing invocation was handled")
		}
		if got := messages[actor.Name]; got != "" {
			t.Fatalf("command gate emitted %q", got)
		}
	})

	t.Run("sleeping", func(t *testing.T) {
		w, actor, _, mob, messages := newButlerTestWorld(t)
		setupButlerContainers(t, w)
		mob.SetPosition(combat.PosSleeping)
		if got := specButler(w, actor, mob, "", ""); got {
			t.Fatal("sleeping butler invocation was handled")
		}
		if got := messages[actor.Name]; got != "" {
			t.Fatalf("sleep gate emitted %q", got)
		}
	})

	t.Run("fighting", func(t *testing.T) {
		w, actor, _, mob, messages := newButlerTestWorld(t)
		setupButlerContainers(t, w)
		mob.SetFighting(actor.Name)
		if got := specButler(w, actor, mob, "", ""); got {
			t.Fatal("fighting butler invocation was handled")
		}
		if got := messages[actor.Name]; got != "" {
			t.Fatalf("fighting gate emitted %q", got)
		}
	})

	t.Run("missing or invisible required container", func(t *testing.T) {
		for _, invisible := range []bool{false, true} {
			t.Run(map[bool]string{false: "missing", true: "invisible"}[invisible], func(t *testing.T) {
				w, actor, _, mob, messages := newButlerTestWorld(t)
				_, _, chest := setupButlerContainers(t, w)
				if invisible {
					chest.SetExtraFlag(0, extraFlagInvisible)
				} else if err := w.MoveObjectToNowhere(chest); err != nil {
					t.Fatalf("remove chest: %v", err)
				}
				if got := specButler(w, actor, mob, "", ""); got {
					t.Fatal("ineligible container set was handled")
				}
				if got := messages[actor.Name]; got != "" {
					t.Fatalf("container gate emitted %q", got)
				}
			})
		}
	})
}

func TestSpecButler_GetRoutingAudienceAndClose(t *testing.T) {
	w, actor, peer, mob, messages := newButlerTestWorld(t)
	cas, cabinet, chest := setupButlerContainers(t, w)

	// Add in the opposite order of C's prepended room list. The handler's
	// reverse snapshot traversal must therefore reproduce C's scan order.
	tunic := spawnButlerTestObject(t, w, 3003)
	sword := spawnButlerTestObject(t, w, 3004)
	bread1 := spawnButlerTestObject(t, w, 3005)
	bread2 := spawnButlerTestObject(t, w, 3005)

	if got := specButler(w, actor, mob, "", ""); !got {
		t.Fatal("eligible butler invocation was not handled")
	}
	want := "The Servant gets a loaf of bread.\r\n" +
		"The Servant puts a loaf of bread in a runed chest.\r\n" +
		"The Servant gets a loaf of bread.\r\n" +
		"The Servant puts a loaf of bread in a runed chest.\r\n" +
		"The Servant gets a short sword.\r\n" +
		"The Servant puts a short sword in a golden weapons cabinet.\r\n" +
		"The Servant gets a frayed tunic.\r\n" +
		"The Servant puts a frayed tunic in an armor display case.\r\n" +
		"The Servant closes an armor display case.\r\n" +
		"The Servant closes a golden weapons cabinet.\r\n" +
		"The Servant closes a runed chest.\r\n"
	if got := messages[actor.Name]; got != want {
		t.Fatalf("actor audience = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != want {
		t.Fatalf("peer audience = %q, want %q", got, want)
	}
	if got := len(w.GetItemsInRoom(butlerTestRoom)); got != 3 {
		t.Fatalf("room item count = %d, want 3 containers", got)
	}
	if got := len(mob.Inventory); got != 0 {
		t.Fatalf("mob inventory after successful puts = %d, want 0", got)
	}
	for name, container := range map[string]*ObjectInstance{
		"case": cas, "cabinet": cabinet, "chest": chest,
	} {
		if container.GetValue(contFlags)&contClosed == 0 {
			t.Errorf("%s remained open", name)
		}
		if len(container.Contains) != map[string]int{"case": 1, "cabinet": 1, "chest": 2}[name] {
			t.Errorf("%s contains %d items", name, len(container.Contains))
		}
	}
	for _, obj := range []*ObjectInstance{tunic, sword, bread1, bread2} {
		if obj.Location.Kind != ObjInContainer {
			t.Errorf("object %d location = %v, want container", obj.VNum, obj.Location.Kind)
		}
	}

	clearButlerMessages(messages)
	butlerDoorToggle(w, mob, cas, true)
	if got, want := messages[peer.Name], "The Servant opens an armor display case.\r\n"; got != want {
		t.Fatalf("open act = %q, want %q", got, want)
	}
	clearButlerMessages(messages)
	butlerDoorToggle(w, mob, cas, true)
	if got := messages[peer.Name]; got != "" {
		t.Fatalf("redundant open emitted %q", got)
	}
	butlerDoorToggle(w, mob, cas, false)
	if got, want := messages[peer.Name], "The Servant closes an armor display case.\r\n"; got != want {
		t.Fatalf("close act = %q, want %q", got, want)
	}
}

func TestSpecButler_CanGetPredicateAndFourItemCap(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*World, *MobInstance, *ObjectInstance)
	}{
		{
			name: "not takeable",
			setup: func(_ *World, _ *MobInstance, obj *ObjectInstance) {
				obj.Prototype.WearFlags[0] = 0
			},
		},
		{
			name: "invisible",
			setup: func(_ *World, _ *MobInstance, obj *ObjectInstance) {
				obj.SetExtraFlag(0, extraFlagInvisible)
			},
		},
		{
			name: "overweight",
			setup: func(_ *World, mob *MobInstance, obj *ObjectInstance) {
				mob.Str = 3
				obj.SetWeight(100)
			},
		},
		{
			name: "over item count",
			setup: func(w *World, mob *MobInstance, _ *ObjectInstance) {
				mob.Dex = 0
				mob.SetLevel(1)
				for i := 0; i < mobMaxCarryCount(mob); i++ {
					filler := spawnButlerTestObject(t, w, 3006)
					if err := w.MoveObjectToMobInventoryFront(filler, mob); err != nil {
						t.Fatalf("fill mob inventory: %v", err)
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, actor, _, mob, messages := newButlerTestWorld(t)
			setupButlerContainers(t, w)
			obj := spawnButlerTestObject(t, w, 3005)
			tc.setup(w, mob, obj)
			if got := specButler(w, actor, mob, "", ""); got {
				t.Fatal("CAN_GET_OBJ-rejected item was handled")
			}
			if got := messages[actor.Name]; got != "" {
				t.Fatalf("rejected item emitted %q", got)
			}
			if got := obj.Location.Kind; got != ObjInRoom {
				t.Fatalf("rejected item location = %v, want room", got)
			}
		})
	}

	t.Run("four item cap", func(t *testing.T) {
		w, actor, _, mob, _ := newButlerTestWorld(t)
		setupButlerContainers(t, w)
		objects := make([]*ObjectInstance, 0, 5)
		for i := 0; i < 5; i++ {
			objects = append(objects, spawnButlerTestObject(t, w, 3005))
		}
		if got := specButler(w, actor, mob, "", ""); !got {
			t.Fatal("eligible four-item invocation was not handled")
		}
		if got := len(mob.Inventory); got != 0 {
			t.Fatalf("mob inventory after four successful puts = %d, want 0", got)
		}
		if got := len(w.GetItemsInRoom(butlerTestRoom)); got != 4 {
			t.Fatalf("room items after four-item cap = %d, want 3 containers plus one skipped item", got)
		}
		if got := objects[0].Location.Kind; got != ObjInRoom {
			t.Fatalf("fifth scanned item location = %v, want room", got)
		}
	})
}

func TestSpecButler_CapacityFailurePreservesInventoryState(t *testing.T) {
	w, actor, peer, mob, messages := newButlerTestWorld(t)
	_, _, chest := setupButlerContainers(t, w)
	chest.SetValue(contCapacity, 0)
	obj := spawnButlerTestObject(t, w, 3005)

	if got := specButler(w, actor, mob, "", ""); !got {
		t.Fatal("capacity-failure invocation was not handled")
	}
	want := "The Servant gets a loaf of bread.\r\n" +
		"The Servant closes an armor display case.\r\n" +
		"The Servant closes a golden weapons cabinet.\r\n" +
		"The Servant closes a runed chest.\r\n"
	if got := messages[peer.Name]; got != want {
		t.Fatalf("capacity-failure audience = %q, want %q", got, want)
	}
	if got := obj.Location.Kind; got != ObjInInventory {
		t.Fatalf("capacity-failure object location = %v, want mob inventory", got)
	}
	if got := len(mob.Inventory); got != 1 || mob.Inventory[0] != obj {
		t.Fatalf("mob inventory = %#v, want the rejected object", mob.Inventory)
	}
	if got := len(chest.Contains); got != 0 {
		t.Fatalf("capacity-failure chest contains %d objects, want 0", got)
	}
}

func TestSpecButler_AutonomousRegisteredDispatch(t *testing.T) {
	w, _, peer, mob, messages := newButlerTestWorld(t)
	setupButlerContainers(t, w)
	obj := spawnButlerTestObject(t, w, 3005)
	if GetMobSpec(8092) == nil {
		t.Fatal("butler registration is missing")
	}

	w.MobileActivityForMob(mob)

	if got := obj.Location.Kind; got != ObjInContainer {
		t.Fatalf("autonomous object location = %v, want container", got)
	}
	if got := messages[peer.Name]; !strings.Contains(got, "The Servant gets a loaf of bread.") {
		t.Fatalf("autonomous dispatch audience = %q", got)
	}
}
