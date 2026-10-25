package game

// Room-light accounting tests: every C site that touches world[].light —
// equip_char/unequip_char (src/handler.c:737-740, 779-782), has_light in
// char_from_room/char_to_room (:520-521, 541-542), burnout (:1061-1064),
// make_corpse's unequips (src/fight.c:406-407), extract_char_final's
// obj_to_room(unequip_char(...)) (src/handler.c:1142) and Crash_extract_objs
// (src/objsave.c:947-955) — must adjust the Go room counter exactly once.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newLightTestWorld builds a ROOM_DARK room 1001 and a plain room 1002.
func newLightTestWorld(t *testing.T) (*World, *Player, *strings.Builder) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Dark Cellar", Zone: 1},
			{VNum: 1002, Name: "Corridor", Zone: 1},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	if !w.SetRoomFlagBit(1001, 0) { // ROOM_DARK
		t.Fatal("SetRoomFlagBit(1001, ROOM_DARK) failed")
	}

	var out strings.Builder
	sink := w.MessageSink
	w.MessageSink = func(name string, msg []byte) {
		if sink != nil {
			sink(name, msg)
		}
		out.Write(msg)
	}

	ch := NewPlayer(1, "Lucci", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return w, ch, &out
}

// newLantern builds a holdable ITEM_LIGHT with fuel in value[2]
// (GET_OBJ_VAL(obj, 2) is the lantern's lit-hours slot).
func newLantern(vnum, fuel int) *ObjectInstance {
	return NewObjectInstance(&parser.Obj{
		VNum:      vnum,
		ShortDesc: "a lantern",
		Keywords:  "lantern",
		TypeFlag:  itemLightTypeFlag,
		Values:    [4]int{0, 0, fuel, 0},
		WearFlags: [4]int{1<<0 | 1<<14}, // ITEM_WEAR_TAKE | ITEM_WEAR_HOLD
	}, -1)
}

func roomLight(t *testing.T, w *World, vnum int) int {
	t.Helper()
	room := w.GetRoomInWorld(vnum)
	if room == nil {
		t.Fatalf("room %d missing", vnum)
	}
	return room.Light
}

// (a) Holding a lit lantern in a dark room lights it; removing it darkens it
// again — equip_char/unequip_char count a lit ITEM_LIGHT in ANY wear position
// (src/handler.c:737-740, 779-782).
func TestHoldLitLanternLightsDarkRoom(t *testing.T) {
	w, ch, _ := newLightTestWorld(t)
	if !w.IsRoomDark(1001) {
		t.Fatal("precondition: ROOM_DARK room with no light must be dark")
	}

	lantern := newLantern(9001, 10)
	if err := ch.Inventory.AddItem(lantern); err != nil {
		t.Fatalf("AddItem lantern: %v", err)
	}
	if err := w.EquipItem(ch, lantern, eqWearHold); err != nil {
		t.Fatalf("EquipItem lantern: %v", err)
	}
	if w.IsRoomDark(1001) {
		t.Fatal("holding a lit lantern must light the room (handler.c:737-740)")
	}
	if got := roomLight(t, w, 1001); got != 1 {
		t.Fatalf("room light after hold = %d, want 1", got)
	}

	if err := w.UnequipItem(ch, eqWearHold); err != nil {
		t.Fatalf("UnequipItem lantern: %v", err)
	}
	if !w.IsRoomDark(1001) {
		t.Fatal("removing the lantern must darken the room (handler.c:779-782)")
	}
	if got := roomLight(t, w, 1001); got != 0 {
		t.Fatalf("room light after remove = %d, want 0", got)
	}
}

// (b) A held light crossing rooms moves its count with the character
// (char_from_room/char_to_room, src/handler.c:520-521, 541-542); both rooms
// end at 0 after a full cycle.
func TestHeldLightMovesWithPlayer(t *testing.T) {
	w, ch, _ := newLightTestWorld(t)

	lantern := newLantern(9002, 10)
	if err := w.EquipItem(ch, lantern, eqWearHold); err != nil {
		t.Fatalf("EquipItem lantern: %v", err)
	}
	if got := roomLight(t, w, 1001); got != 1 {
		t.Fatalf("room light after hold = %d, want 1", got)
	}

	if err := w.PlayerTransfer(ch, 1002); err != nil {
		t.Fatalf("PlayerTransfer to 1002: %v", err)
	}
	if got := roomLight(t, w, 1001); got != 0 {
		t.Fatalf("origin room light after leaving = %d, want 0 (handler.c:520-521)", got)
	}
	if got := roomLight(t, w, 1002); got != 1 {
		t.Fatalf("destination room light after arriving = %d, want 1 (handler.c:541-542)", got)
	}

	if err := w.PlayerTransfer(ch, 1001); err != nil {
		t.Fatalf("PlayerTransfer to 1001: %v", err)
	}
	if err := w.UnequipItem(ch, eqWearHold); err != nil {
		t.Fatalf("UnequipItem lantern: %v", err)
	}
	if err := w.PlayerTransfer(ch, 1002); err != nil {
		t.Fatalf("PlayerTransfer to 1002: %v", err)
	}
	for _, vnum := range []int{1001, 1002} {
		if got := roomLight(t, w, vnum); got != 0 {
			t.Fatalf("room %d light after full cycle = %d, want 0", vnum, got)
		}
	}
}

// (c) Burnout decrements exactly once: update_char_objects drops the counter
// when value[2] hits 0 (src/handler.c:1061-1064) and the unequip on the way
// out is a no-op because value[2] is already 0 (:779-782).
func TestBurnoutDropsRoomLightExactlyOnce(t *testing.T) {
	w, ch, _ := newLightTestWorld(t)

	lantern := newLantern(9003, 1)
	if err := w.EquipItem(ch, lantern, eqWearHold); err != nil {
		t.Fatalf("EquipItem lantern: %v", err)
	}
	if got := roomLight(t, w, 1001); got != 1 {
		t.Fatalf("room light after hold = %d, want 1", got)
	}

	w.updateCharObjects(ch)

	if got := roomLight(t, w, 1001); got != 0 {
		t.Fatalf("room light after burnout = %d, want 0 (handler.c:1061-1064)", got)
	}
	if w.IsEquipped(ch, eqWearHold) {
		t.Fatal("burnt-out light must leave the hold slot")
	}
	if lantern.GetValue(2) != 0 {
		t.Fatalf("lantern fuel after burnout = %d, want 0", lantern.GetValue(2))
	}
}

// (d) extract_char_final drops equipped items with obj_to_room(unequip_char())
// — one decrement per lit light, and the DROP itself adds nothing
// (src/handler.c:1142; obj_to_room :901-913 never touches light). A dropped
// lit lantern therefore leaves the room dark.
func TestExtractionDropsHeldLightWithoutRelight(t *testing.T) {
	w, ch, _ := newLightTestWorld(t)

	lantern := newLantern(9004, 10)
	if err := w.EquipItem(ch, lantern, eqWearHold); err != nil {
		t.Fatalf("EquipItem lantern: %v", err)
	}
	if got := roomLight(t, w, 1001); got != 1 {
		t.Fatalf("room light after hold = %d, want 1", got)
	}

	ExtractChar(ch)
	w.ExtractPendingChars()

	if got := roomLight(t, w, 1001); got != 0 {
		t.Fatalf("room light after extraction = %d, want 0", got)
	}
	if !w.IsRoomDark(1001) {
		t.Fatal("a dropped lit lantern must not keep the room lit (obj_to_room adds no light)")
	}
	if lantern.Location.Kind != ObjInRoom || lantern.Location.RoomVNum != 1001 {
		t.Fatalf("lantern location after extraction = %+v, want room 1001", lantern.Location)
	}
}

// (e) Renting out with a lit light: Crash_extract_objs frees the rent set
// while the character is still in the room (src/objsave.c:947-955, 996-1004),
// so each still-lit equipped light leaves the counter — and the later
// extraction pass must not decrement again for a renter.
func TestRentOutDropsHeldLightOnce(t *testing.T) {
	w, ch, _ := newLightTestWorld(t)

	lantern := newLantern(9005, 10)
	if err := w.EquipItem(ch, lantern, eqWearHold); err != nil {
		t.Fatalf("EquipItem lantern: %v", err)
	}

	w.RentOut(ch)
	if got := roomLight(t, w, 1001); got != 0 {
		t.Fatalf("room light after RentOut = %d, want 0", got)
	}

	ch.RentedOut = true
	ExtractChar(ch)
	w.ExtractPendingChars()
	if got := roomLight(t, w, 1001); got != 0 {
		t.Fatalf("room light after renter extraction = %d, want 0 (no double decrement)", got)
	}
}

// Login: char_to_room counts the character's lit light on entry
// (src/interpreter.c:2210; src/handler.c:541-542). Crash_load equipped the
// restored light while the character was still NOWHERE, so AddPlayer is the
// count's only increment on entry.
func TestAddPlayerCountsRestoredLight(t *testing.T) {
	w, _, _ := newLightTestWorld(t)

	enterer := NewPlayer(2, "Restorer", 1001)
	if enterer.Equipment == nil {
		enterer.Equipment = NewEquipment()
		enterer.Equipment.OwnerName = enterer.Name
	}
	lantern := newLantern(9006, 10)
	if err := enterer.Equipment.SetSlot(SlotHold, lantern); err != nil {
		t.Fatalf("SetSlot lantern: %v", err)
	}
	lantern.Location = LocEquippedPlayer(enterer.Name, SlotHold)

	if err := w.AddPlayer(enterer); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if got := roomLight(t, w, 1001); got != 1 {
		t.Fatalf("room light after entry with lit light = %d, want 1", got)
	}
	if w.IsRoomDark(1001) {
		t.Fatal("a character entering with a lit light must light the room")
	}
}
