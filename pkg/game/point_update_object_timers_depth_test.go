package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newObjectTimerDepthWorld(t *testing.T, rooms []parser.Room, mobs []parser.Mob, objs []parser.Obj) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: rooms, Mobs: mobs, Objs: objs})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

func objectTimerDepthPlayer(t *testing.T, w *World, name string, room int) *Player {
	t.Helper()
	p := NewPlayer(1, name, room)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return p
}

func TestPointUpdateDecaysObjectsInCarriedAndContainedLocations(t *testing.T) {
	room := parser.Room{VNum: 1001, Name: "Timer room", Zone: 1}
	objs := []parser.Obj{
		{VNum: 20, Keywords: "puddle", ShortDesc: "a puddle", TypeFlag: ITEM_OTHER},
		{VNum: 7001, Keywords: "corpse", ShortDesc: "a corpse", TypeFlag: ITEM_CONTAINER, Values: [4]int{0, 0, 0, 1}},
		{VNum: 7002, Keywords: "bag", ShortDesc: "a bag", TypeFlag: ITEM_CONTAINER},
		{VNum: 7003, Keywords: "ring", ShortDesc: "a ring", TypeFlag: ITEM_OTHER},
	}
	t.Run("carried puddle", func(t *testing.T) {
		w := newObjectTimerDepthWorld(t, []parser.Room{room}, nil, objs)
		p := objectTimerDepthPlayer(t, w, "Carrier", 1001)
		puddle := w.CreateObject(20, 1001)
		puddle.SetTimer(1)
		if err := w.MoveObjectToPlayerInventory(puddle, p); err != nil {
			t.Fatal(err)
		}

		w.PointUpdate()

		if _, live := w.objectInstances[puddle.ID]; live {
			t.Fatal("carried puddle survived point_update; C traverses every object_list location")
		}
	})

	t.Run("carried corpse spills to carrier room", func(t *testing.T) {
		w := newObjectTimerDepthWorld(t, []parser.Room{room}, nil, objs)
		p := objectTimerDepthPlayer(t, w, "Carrier", 1001)
		out := captureMovementOutput(w)
		corpse := w.CreateObject(7001, 1001)
		corpse.SetTimer(1)
		child := w.CreateObject(7003, 1001)
		if err := w.MoveObjectToContainer(child, corpse); err != nil {
			t.Fatal(err)
		}
		if err := w.MoveObjectToPlayerInventory(corpse, p); err != nil {
			t.Fatal(err)
		}

		w.PointUpdate()

		if _, live := w.objectInstances[corpse.ID]; live {
			t.Fatal("carried corpse survived point_update")
		}
		if child.Location != LocRoom(1001) {
			t.Fatalf("spilled child location = %#v, want room 1001", child.Location)
		}
		if got := out[p.Name].String(); got != "A corpse decays in your hands, and the contents fall to the ground.\r\n" {
			t.Fatalf("carrier output = %q", got)
		}
	})

	t.Run("contained corpse spills into parent", func(t *testing.T) {
		w := newObjectTimerDepthWorld(t, []parser.Room{room}, nil, objs)
		outer := w.CreateObject(7002, 1001)
		corpse := w.CreateObject(7001, 1001)
		corpse.SetTimer(1)
		child := w.CreateObject(7003, 1001)
		if err := w.MoveObjectToContainer(child, corpse); err != nil {
			t.Fatal(err)
		}
		if err := w.MoveObjectToContainer(corpse, outer); err != nil {
			t.Fatal(err)
		}

		w.PointUpdate()

		if _, live := w.objectInstances[corpse.ID]; live {
			t.Fatal("contained corpse survived point_update")
		}
		if child.Location != LocContainer(outer.ID) {
			t.Fatalf("spilled child location = %#v, want parent container %d", child.Location, outer.ID)
		}
		if got := outer.GetContents(); len(got) != 1 || got[0] != child {
			t.Fatalf("parent contents = %#v, want the spilled child", got)
		}
	})
}

func TestPointUpdateFieldObjectUsesCRealRoomIndex(t *testing.T) {
	objs := []parser.Obj{
		{VNum: 20, Keywords: "puddle", ShortDesc: "a puddle", TypeFlag: ITEM_OTHER},
		{VNum: 51, Keywords: "ice wall", ShortDesc: "a wall of ice", TypeFlag: ITEM_OTHER},
	}
	t.Run("rnum zero remains untouched", func(t *testing.T) {
		w := newObjectTimerDepthWorld(t, []parser.Room{{VNum: 1001, Name: "First", Zone: 1}}, nil, objs)
		objectTimerDepthPlayer(t, w, "Witness", 1001)
		wall := w.CreateObject(51, 1001)
		wall.SetTimer(1)

		w.PointUpdate()

		if _, live := w.objectInstances[wall.ID]; !live || wall.GetTimer() != 1 {
			t.Fatalf("rnum-zero field object live=%t timer=%d, want live and timer 1", w.objectInstances[wall.ID] == wall, wall.GetTimer())
		}
	})

	t.Run("nonzero rnum expires and spawns the puddle", func(t *testing.T) {
		w := newObjectTimerDepthWorld(t, []parser.Room{
			{VNum: 1001, Name: "First", Zone: 1},
			{VNum: 1002, Name: "Second", Zone: 1},
		}, nil, objs)
		p := objectTimerDepthPlayer(t, w, "Witness", 1002)
		out := captureMovementOutput(w)
		wall := w.CreateObject(51, 1002)
		wall.SetTimer(1)

		w.PointUpdate()

		if _, live := w.objectInstances[wall.ID]; live {
			t.Fatal("field object survived at C rnum 1")
		}
		if got := out[p.Name].String(); got != "A wall of ice rapidly melts into a large puddle.\r\n" {
			t.Fatalf("field wear-off output = %q", got)
		}
		items := w.GetItemsInRoom(1002)
		if len(items) != 1 || items[0].GetVNum() != 20 || items[0].GetTimer() != 2 {
			t.Fatalf("room objects after decay = %#v, want spawned puddle with timer 2", items)
		}
	})
}

func TestPointUpdateExpiresAssignedMoonGateByValueThree(t *testing.T) {
	w := newObjectTimerDepthWorld(t, []parser.Room{{VNum: 1001, Name: "Gate room", Zone: 1}}, nil,
		[]parser.Obj{{VNum: 4001, Keywords: "gate", ShortDesc: "a blue portal", TypeFlag: ITEM_OTHER, Values: [4]int{0, 0, 0, 0}}})
	p := objectTimerDepthPlayer(t, w, "Witness", 1001)
	out := captureMovementOutput(w)
	gate := w.CreateObject(4001, 1001)
	gate.SetValue(2, 1)

	w.PointUpdate()

	if _, live := w.objectInstances[gate.ID]; live {
		t.Fatal("assigned moon_gate object survived with val[2] at zero")
	}
	if got := out[p.Name].String(); got != "A blue portal fades out of existence.\r\n" {
		t.Fatalf("moon_gate output = %q", got)
	}
}

func TestPointUpdateDustAndCircleTimerOutputAcrossLocations(t *testing.T) {
	room := parser.Room{VNum: 1001, Name: "Timer room", Zone: 1}
	objs := []parser.Obj{
		{VNum: 18, Keywords: "dust", ShortDesc: "a pile of dust", TypeFlag: ITEM_OTHER},
		{VNum: 64, Keywords: "circle", ShortDesc: "a circle", TypeFlag: ITEM_OTHER},
	}
	for _, fixture := range []struct {
		name    string
		vnum    int
		message string
	}{
		{name: "dust", vnum: 18, message: "The pile of dust is blown away by a draft of wind.\r\n"},
		{name: "circle", vnum: 64, message: "The circle on the ground slowly fades away.\r\n"},
	} {
		for _, carried := range []bool{false, true} {
			name := "room"
			if carried {
				name = "carried"
			}
			t.Run(fixture.name+"/"+name, func(t *testing.T) {
				w := newObjectTimerDepthWorld(t, []parser.Room{room}, nil, objs)
				p := objectTimerDepthPlayer(t, w, "Witness", 1001)
				out := captureMovementOutput(w)
				obj := w.CreateObject(fixture.vnum, 1001)
				obj.SetTimer(1)
				if carried {
					if err := w.MoveObjectToPlayerInventory(obj, p); err != nil {
						t.Fatal(err)
					}
				}

				w.PointUpdate()

				if _, live := w.objectInstances[obj.ID]; live {
					t.Fatalf("%s object survived point_update", fixture.name)
				}
				want := fixture.message
				if carried {
					want = ""
				}
				got := ""
				if output := out[p.Name]; output != nil {
					got = output.String()
				}
				if got != want {
					t.Fatalf("%s output = %q, want %q", fixture.name, got, want)
				}
			})
		}
	}
}

func TestSpellGateStoresCountdownInObjectValueThree(t *testing.T) {
	w := newObjectTimerDepthWorld(t, []parser.Room{{VNum: 4001, Name: "Gate room", Zone: 1}}, nil,
		[]parser.Obj{{VNum: 4002, Keywords: "red portal", ShortDesc: "a red portal", TypeFlag: ITEM_OTHER, Values: [4]int{0, 0, 0, 0}}})
	p := objectTimerDepthPlayer(t, w, "Caster", 4001)

	if !w.SpellGate(p) {
		t.Fatal("SpellGate did not handle legal portal room")
	}
	items := w.GetItemsInRoom(4001)
	if len(items) != 1 || items[0].GetVNum() != 4002 {
		t.Fatalf("room objects = %#v, want red portal", items)
	}
	if got := items[0].GetValue(2); got != 2 {
		t.Fatalf("red gate val[2] = %d, want 2", got)
	}
}

func TestPointUpdateRunsGlobalObjectPassAfterAllMemoryDraws(t *testing.T) {
	room := parser.Room{VNum: 1001, Name: "RNG room", Zone: 1}
	parsedMobs := []parser.Mob{
		{VNum: 2100, Keywords: "first mob", ShortDesc: "a first mob", Level: 1, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 10}},
		{VNum: 2101, Keywords: "second mob", ShortDesc: "a second mob", Level: 1, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 10}},
	}
	objs := []parser.Obj{{VNum: 7001, Keywords: "corpse", ShortDesc: "a corpse", TypeFlag: ITEM_CONTAINER, Values: [4]int{0, 0, 0, 1}}}
	w := newObjectTimerDepthWorld(t, []parser.Room{room}, parsedMobs, objs)
	p := objectTimerDepthPlayer(t, w, "Witness", 1001)
	out := captureMovementOutput(w)
	for _, vnum := range []int{2100, 2101} {
		mob, err := w.SpawnMob(vnum, 1001)
		if err != nil {
			t.Fatal(err)
		}
		mob.Remember("Witness")
	}
	for range 2 {
		corpse := w.CreateObject(7001, 1001)
		corpse.SetTimer(1)
	}

	const seed = 3817
	dprng.ResetStream(seed)
	t.Cleanup(func() { dprng.ResetStream(1) })
	expected := dprng.New(seed)
	expected.Number(0, 98)
	expected.Number(0, 98)
	firstMessage := corpseDecayMessage(expected.Number(0, 6))
	secondMessage := corpseDecayMessage(expected.Number(0, 6))
	wantNext := expected.Next()

	w.PointUpdate()

	got := ""
	if output := out[p.Name]; output != nil {
		got = output.String()
	}
	if want := firstMessage + secondMessage; got != want {
		t.Fatalf("corpse messages = %q, want C global-list order %q", got, want)
	}
	if gotNext := dprng.Next(); gotNext != wantNext {
		t.Fatalf("next CMWC draw = %d, want %d after two memory draws then two corpse draws", gotNext, wantNext)
	}
}

func corpseDecayMessage(draw int) string {
	messages := []string{
		"A quivering horde of maggots consumes a corpse.\r\n",
		"Dissolving into the ground, a corpse disappears.\r\n",
		"Dissolving into the ground, a corpse disappears.\r\n",
		"A horde of flesh-eating ants consumes a corpse.\r\n",
		"A horde of flesh-eating ants consumes a corpse.\r\n",
		"The earth opens up and swallows a corpse.\r\n",
		"The earth opens up and swallows a corpse.\r\n",
	}
	return messages[draw]
}
