package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newHandlerOutputSurfaceWorld(t *testing.T, objects []parser.Obj) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Output Room"}},
		Objs:  objects,
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

func addHandlerOutputSurfacePlayer(t *testing.T, w *World, id int, name string) *Player {
	t.Helper()
	p := NewPlayer(id, name, 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer(%s): %v", name, err)
	}
	return p
}

func TestPlayerWearRefusalOutputMatchesC(t *testing.T) {
	w := newHandlerOutputSurfaceWorld(t, []parser.Obj{
		{
			VNum: 99001, Keywords: "shield", ShortDesc: "a shield", TypeFlag: ITEM_ARMOR,
			WearFlags: [4]int{1 | 1<<9}, ExtraFlags: [4]int{1 << 9},
		},
		{
			VNum: 99002, Keywords: "robe", ShortDesc: "a robe", TypeFlag: ITEM_ARMOR,
			WearFlags: [4]int{1 | 1<<3}, ExtraFlags: [4]int{1 << 12},
		},
	})
	actor := addHandlerOutputSurfacePlayer(t, w, 1, "Wearer")
	observer := addHandlerOutputSurfacePlayer(t, w, 2, "Observer")
	actor.SetAlignment(1000)
	actor.SetClass(ClassMageUser)

	outputs := make(map[string]string)
	w.MessageSink = func(name string, message []byte) { outputs[name] += string(message) }

	shield, err := w.SpawnObject(99001, -1)
	if err != nil {
		t.Fatalf("SpawnObject(shield): %v", err)
	}
	if err := w.PlaceObjectInPlayerInventory(shield, actor); err != nil {
		t.Fatalf("PlaceObjectInPlayerInventory(shield): %v", err)
	}
	w.DoWear(actor, "shield")
	if got := outputs[actor.Name]; !strings.Contains(got, "You are zapped by a shield and instantly let go of it.\r\n") {
		t.Fatalf("anti-alignment actor output = %q", got)
	}
	if got := outputs[observer.Name]; !strings.Contains(got, "Wearer is zapped by a shield and instantly lets go of it.\r\n") {
		t.Fatalf("anti-alignment room output = %q", got)
	}
	if shield.Location != LocInventoryPlayer(actor.Name) || w.IsEquipped(actor, eqWearShield) {
		t.Fatalf("anti-alignment refusal changed item placement: location=%+v equipped=%t", shield.Location, w.IsEquipped(actor, eqWearShield))
	}

	outputs[actor.Name], outputs[observer.Name] = "", ""
	robe, err := w.SpawnObject(99002, -1)
	if err != nil {
		t.Fatalf("SpawnObject(robe): %v", err)
	}
	if err := w.PlaceObjectInPlayerInventory(robe, actor); err != nil {
		t.Fatalf("PlaceObjectInPlayerInventory(robe): %v", err)
	}
	w.DoWear(actor, "robe")
	if got := outputs[actor.Name]; got != "You cannot use a robe.\r\n" {
		t.Fatalf("invalid-class actor output = %q", got)
	}
	if got := outputs[observer.Name]; got != "" {
		t.Fatalf("invalid-class room output = %q, want no room bytes", got)
	}
	if robe.Location != LocInventoryPlayer(actor.Name) || w.IsEquipped(actor, eqWearBody) {
		t.Fatalf("invalid-class refusal changed item placement: location=%+v equipped=%t", robe.Location, w.IsEquipped(actor, eqWearBody))
	}
}

func TestPointUpdateHeldLightOutputMatchesC(t *testing.T) {
	w := newHandlerOutputSurfaceWorld(t, []parser.Obj{{
		VNum: 99003, Keywords: "torch", ShortDesc: "a torch", TypeFlag: ITEM_LIGHT,
		WearFlags: [4]int{1 | 1<<14}, Values: [4]int{0, 0, 2, 0}, Weight: 1,
	}})
	actor := addHandlerOutputSurfacePlayer(t, w, 1, "Lightuser")
	observer := addHandlerOutputSurfacePlayer(t, w, 2, "Lightpeer")
	for _, p := range []*Player{actor, observer} {
		p.SetCondition(CondFull, -1)
		p.SetCondition(CondDrunk, -1)
		p.SetCondition(CondThirst, -1)
	}

	torch, err := w.SpawnObject(99003, -1)
	if err != nil {
		t.Fatalf("SpawnObject(torch): %v", err)
	}
	if err := w.MoveObject(torch, LocEquippedPlayer(actor.Name, SlotHold)); err != nil {
		t.Fatalf("equip held torch: %v", err)
	}
	if got := w.GetRoomInWorld(1001).Light; got != 1 {
		t.Fatalf("room light after equip = %d, want 1", got)
	}

	var events []string
	w.MessageSink = func(name string, message []byte) {
		events = append(events, name+":"+string(message))
	}
	w.PointUpdate()
	w.PointUpdate()

	want := []string{
		"Lightuser:Your light begins to flicker and fade.\r\n",
		"Lightpeer:Lightuser's light begins to flicker and fade.\r\n",
		"Lightuser:Your light sputters out and dies.\r\n",
		"Lightpeer:Lightuser's light sputters out and dies.\r\n",
		"Lightuser:It crumbles into dust.\r\n",
	}
	if len(events) != len(want) {
		t.Fatalf("held-light events = %#v, want %#v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("held-light event %d = %q, want %q (all events: %#v)", i, events[i], want[i], events)
		}
	}
	if got := w.GetRoomInWorld(1001).Light; got != 0 {
		t.Fatalf("room light after expiry = %d, want 0", got)
	}
	if torch.Location != LocNowhere() {
		t.Fatalf("expired torch location = %+v, want nowhere", torch.Location)
	}
}
