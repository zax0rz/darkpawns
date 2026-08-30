package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

var elementsLoadCylinderRooms = []int{1360, 1364, 1380, 1384}

func newElementsLoadCylindersTestWorld(t *testing.T) (*World, *Player, *Player, map[string]string) {
	t.Helper()

	rooms := []parser.Room{
		{VNum: 1360, Name: "Earth Pillar"},
		{VNum: 1364, Name: "Air Pillar"},
		{VNum: 1380, Name: "Fire Pillar"},
		{VNum: 1384, Name: "Water Pillar"},
		{VNum: 9000, Name: "Other Room"},
	}
	objs := make([]parser.Obj, 0, 8)
	for vnum := 1300; vnum <= 1307; vnum++ {
		keyword := "talisman"
		short := "an elemental talisman"
		if vnum >= 1304 {
			keyword = "cylinder"
			short = "a cylinder of light"
		}
		obj := parser.Obj{
			VNum:      vnum,
			Keywords:  keyword,
			ShortDesc: short,
			LongDesc:  strings.ToUpper(short[:1]) + short[1:] + " is here.",
			ExtraFlags: [4]int{
				func() int {
					if vnum >= 1304 {
						return 1 << itemExtraGlow
					}
					return 0
				}(),
			},
		}
		if vnum < 1304 {
			obj.WearFlags[0] = 1
		}
		objs = append(objs, obj)
	}
	w, err := NewWorld(&parser.World{Rooms: rooms, Objs: objs})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "CylinderCarrier", 1360)
	peer := NewPlayer(2, "CylinderPeer", 1360)
	actor.SetAutoExit(false)
	peer.SetAutoExit(false)
	for _, player := range []*Player{actor, peer} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer(%s): %v", player.GetName(), err)
		}
	}
	return w, actor, peer, messages
}

func moveElementsObject(t *testing.T, w *World, vnum, room int) *ObjectInstance {
	t.Helper()
	obj, err := w.SpawnObject(vnum, room)
	if err != nil {
		t.Fatalf("SpawnObject(%d): %v", vnum, err)
	}
	if err := w.MoveObjectToRoom(obj, room); err != nil {
		t.Fatalf("MoveObjectToRoom(%d): %v", vnum, err)
	}
	return obj
}

func moveElementsObjectToInventory(t *testing.T, w *World, p *Player, vnum int) *ObjectInstance {
	t.Helper()
	obj, err := w.SpawnObject(vnum, -1)
	if err != nil {
		t.Fatalf("SpawnObject(%d): %v", vnum, err)
	}
	if err := w.MoveObjectToPlayerInventory(obj, p); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory(%d): %v", vnum, err)
	}
	return obj
}

func TestSpecElementsLoadCylinders_MapsAllRegisteredPillars(t *testing.T) {
	for i, room := range elementsLoadCylinderRooms {
		t.Run(string(rune('0'+i)), func(t *testing.T) {
			w, actor, peer, messages := newElementsLoadCylindersTestWorld(t)
			actor.SetRoom(room)
			peer.SetRoom(room)
			moveElementsObjectToInventory(t, w, actor, 1300+i)

			if !specElementsLoadCylinders(w, actor, nil, "drop", "talisman") {
				t.Fatal("drop was not consumed by the room special")
			}
			items := w.GetItemsInRoom(room)
			if len(items) != 2 || items[0].GetVNum() != 1304+i || items[1].GetVNum() != 1300+i {
				t.Fatalf("room items = %#v, want cylinder %d then talisman %d", items, 1304+i, 1300+i)
			}
			if got := messages[actor.GetName()]; !strings.Contains(got, "A "+[]string{"green", "yellow", "red", "blue"}[i]+" cylinder of light extends upwards from the pillar.\r\n") {
				t.Errorf("actor missing cylinder announcement: %q", got)
			}
			if got := messages[peer.GetName()]; !strings.Contains(got, "A "+[]string{"green", "yellow", "red", "blue"}[i]+" cylinder of light extends upwards from the pillar.\r\n") {
				t.Errorf("peer missing room-wide cylinder announcement: %q", got)
			}
		})
	}
}

func TestSpecElementsLoadCylinders_RejectsWrongTalismanAndExistingCylinder(t *testing.T) {
	w, actor, _, _ := newElementsLoadCylindersTestWorld(t)
	moveElementsObjectToInventory(t, w, actor, 1301)
	if !specElementsLoadCylinders(w, actor, nil, "drop", "talisman") {
		t.Fatal("wrong talisman drop should still be consumed")
	}
	for _, item := range w.GetItemsInRoom(1360) {
		if item.GetVNum() >= 1304 && item.GetVNum() <= 1307 {
			t.Fatalf("wrong talisman loaded cylinder %d", item.GetVNum())
		}
	}

	actor.SetRoom(1364)
	moveElementsObjectToInventory(t, w, actor, 1301)
	cylinder := moveElementsObject(t, w, 1304, 1364)
	if specElementsLoadCylinders(w, actor, nil, "drop", "talisman") {
		t.Fatal("existing any-cylinder gate should fall through")
	}
	if cylinder.Location != LocRoom(1364) {
		t.Fatalf("existing cylinder location = %#v, want room", cylinder.Location)
	}
	if len(actor.Inventory.Items) != 1 || actor.Inventory.Items[0].GetVNum() != 1301 {
		t.Fatalf("blocked drop changed inventory: %#v", actor.Inventory.Items)
	}
}

func TestSpecElementsLoadCylinders_GetRemovesOnlyCurrentRoomCylinder(t *testing.T) {
	w, actor, peer, messages := newElementsLoadCylindersTestWorld(t)
	moveElementsObject(t, w, 1300, 1360)
	cylinder := moveElementsObject(t, w, 1304, 1360)
	other := moveElementsObject(t, w, 1305, 1364)

	if !specElementsLoadCylinders(w, actor, nil, "get", "talisman") {
		t.Fatal("get should be consumed by the room special")
	}
	for _, item := range w.GetItemsInRoom(1360) {
		if item.GetVNum() == cylinder.GetVNum() {
			t.Fatal("current-room cylinder was not extracted")
		}
	}
	if other.Location != LocRoom(1364) {
		t.Fatalf("other-room cylinder location = %#v, want room", other.Location)
	}
	if !strings.Contains(messages[actor.GetName()], "The green cylinder of light slowly sinks back into the pillar.\r\n") {
		t.Errorf("actor missing cylinder removal announcement: %q", messages[actor.GetName()])
	}
	if !strings.Contains(messages[peer.GetName()], "The green cylinder of light slowly sinks back into the pillar.\r\n") {
		t.Errorf("peer missing room-wide removal announcement: %q", messages[peer.GetName()])
	}
}

func TestElementsRemoveCylinders_PreservesCFirstTalismanReturn(t *testing.T) {
	w, actor, _, _ := newElementsLoadCylindersTestWorld(t)
	moveElementsObject(t, w, 1300, actor.GetRoomVNum())
	cylinder := moveElementsObject(t, w, 1305, actor.GetRoomVNum())

	elementsRemoveCylinders(w, actor.GetRoomVNum())
	if cylinder.Location != LocRoom(actor.GetRoomVNum()) {
		t.Fatalf("later cylinder location = %#v, want unchanged room", cylinder.Location)
	}
}
