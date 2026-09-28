package game

// obj_to_room_order_test.go — DP-1344's R5h unit proof for C obj_to_room's
// prepend order (src/handler.c:897-910: object->next_content =
// world[room].contents; world[room].contents = object;). Room lists are
// stored in C's contents order, so a placement through a fixed path must
// leave the newly placed object first in GetItemsInRoom.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newObjToRoomItem(w *World, vnum int) *ObjectInstance {
	obj := newTransferItem(vnum, "a thing", "thing", 1)
	obj.SetWeight(0)
	registerTransferObject(w, obj)
	return obj
}

// TestDropPlacementsPrependInCOrder proves two sequential room placements
// through the drop path (C perform_drop, act.item.c:504) leave the second
// object first, matching obj_to_room's prepend.
func TestDropPlacementsPrependInCOrder(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Vault"},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	ch := NewPlayer(1, "Dropper", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	first := newObjToRoomItem(w, 7101)
	if err := w.MoveObjectToRoom(first, 1001); err != nil {
		t.Fatalf("seed floor: %v", err)
	}
	second := newObjToRoomItem(w, 7102)
	if err := w.MoveObjectToPlayerInventory(second, ch); err != nil {
		t.Fatalf("seat second: %v", err)
	}

	w.DoDrop(ch, "thing")

	items := w.GetItemsInRoom(1001)
	if len(items) != 2 {
		t.Fatalf("room items = %d, want 2", len(items))
	}
	if items[0].VNum != 7102 || items[1].VNum != 7101 {
		t.Fatalf("room order after drop = [%d, %d], want the dropped 7102 first (C obj_to_room prepends)", items[0].VNum, items[1].VNum)
	}
}

// TestDonatePlacementPrependsInDonationRoom proves the donate path (C
// perform_drop's SCMD_DONATE arm, act.item.c:510) also prepends in the
// donation room.
func TestDonatePlacementPrependsInDonationRoom(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Vault"},
		{VNum: 3043, Name: "Donation"},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	ch := NewPlayer(1, "Donator", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	first := newObjToRoomItem(w, 7101)
	if err := w.MoveObjectToRoom(first, 3043); err != nil {
		t.Fatalf("seed donation room: %v", err)
	}
	second := newObjToRoomItem(w, 7102)
	if err := w.MoveObjectToPlayerInventory(second, ch); err != nil {
		t.Fatalf("seat second: %v", err)
	}

	// Call performDispose directly with an explicit donation room: DoDonate
	// rolls C's RDR dice (randRange 0-3), which would make the target room
	// stream-dependent.
	w.performDispose(ch, second, scmdDonate, "donate", 3043)

	items := w.GetItemsInRoom(3043)
	if len(items) != 2 {
		t.Fatalf("donation room items = %d, want 2", len(items))
	}
	if items[0].VNum != 7102 || items[1].VNum != 7101 {
		t.Fatalf("donation order = [%d, %d], want the donated 7102 first (C obj_to_room prepends)", items[0].VNum, items[1].VNum)
	}
}

// TestCorpsePlacementPrependsOnBusyFloor proves the corpse placement (C
// make_corpse, fight.c:423) prepends above an object already on the floor.
func TestCorpsePlacementPrependsOnBusyFloor(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Vault"},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	ch := NewPlayer(1, "Victim", 1001)
	ch.SetGold(5)
	ch.SetHealth(-20) // handlePlayerDeath's HP gate
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	floor := newObjToRoomItem(w, 7101)
	if err := w.MoveObjectToRoom(floor, 1001); err != nil {
		t.Fatalf("seed floor: %v", err)
	}

	w.handlePlayerDeath(ch, false, 0, "test")

	items := w.GetItemsInRoom(1001)
	if len(items) != 2 {
		t.Fatalf("room items after death = %d, want 2", len(items))
	}
	if items[0].VNum != -1 { // the synthetic corpse has no prototype vnum
		t.Fatalf("room order after death = [%d, %d], want the corpse first (C obj_to_room prepends)", items[0].VNum, items[1].VNum)
	}
	if items[1].VNum != 7101 {
		t.Fatalf("seeded floor object = %d, want 7101", items[1].VNum)
	}
}
