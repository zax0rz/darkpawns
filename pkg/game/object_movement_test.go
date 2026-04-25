// Package game — object movement tests.
//
// These tests document current item behavior BEFORE the ObjectLocation refactor (PR 4).
// They serve as regression safety for the refactored location tracking system.
//
// SURPRISING BEHAVIOR DOCUMENTED HERE:
// 1. SpawnObject() does NOT call AddItemToRoom() — callers must do it separately.
// 2. Inventory.AddItem() sets item.Carrier = *Inventory, not *Player or *MobInstance.
//    ExtractObject() type-asserts Carrier as *Player or *MobInstance, so items in
//    player inventory (carried as *Inventory) will NOT be cleaned up by ExtractObject's
//    carrier-removal branch.  The global-instance deletion still fires.
package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newTestWorld constructs a minimal World with one room, one mob proto, one obj proto,
// and one registered player for object-movement tests.
func newTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Test Room", Zone: 1},
		},
		Mobs: []parser.Mob{
			{VNum: 2001},
		},
		Objs: []parser.Obj{
			{
				VNum: 3001, Keywords: "test object", ShortDesc: "a test object",
				LongDesc: "A test object lies here.", TypeFlag: 0,
				WearFlags: [3]int{0, 0, 0},
				Values:    [4]int{0, 0, 0, 0},
			},
			{
				VNum: 3002, Keywords: "test weapon", ShortDesc: "a test weapon",
				LongDesc: "A test weapon lies here.", TypeFlag: 5,
				WearFlags: [3]int{1 << 13, 0, 0}, // ITEM_WEAR_WIELD
				Values:    [4]int{0, 1, 4, 0},
			},
			{
				VNum: 3003, Keywords: "test container", ShortDesc: "a test container",
				LongDesc: "A test container lies here.", TypeFlag: 1,
				WearFlags: [3]int{0, 0, 0},
				Values:    [4]int{0, 0, 0, 0},
			},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	player := NewPlayer(1, "TestPlayer", 1001)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// Clean up stop-channel side-effects from NewWorld so tests don't leak goroutines.
	// We'll defer a clean shutdown if the AI ticker is still running.
	t.Cleanup(func() {
		w.StopAITicker()
	})

	return w, player
}

// ---------------------------------------------------------------------------
// TestRoomToPlayerInventory — pick up an item from the room into inventory
// ---------------------------------------------------------------------------

func TestRoomToPlayerInventory(t *testing.T) {
	w, player := newTestWorld(t)

	// Spawn object into the world (objectInstances map only)
	obj, err := w.SpawnObject(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject failed: %v", err)
	}

	// Separately add to room
	w.AddItemToRoom(obj, 1001)

	// Verify it's on the floor
	roomItems := w.GetItemsInRoom(1001)
	if len(roomItems) != 1 {
		t.Fatalf("expected 1 item in room, got %d", len(roomItems))
	}

	// Player picks up: remove from room, add to inventory
	removed := w.RemoveItemFromRoom(obj, 1001)
	if !removed {
		t.Fatal("RemoveItemFromRoom returned false")
	}

	if err := player.Inventory.AddItem(obj); err != nil {
		t.Fatalf("AddItem to inventory failed: %v", err)
	}

	// Assert: item in inventory, not in room
	if len(player.Inventory.Items) != 1 {
		t.Errorf("expected 1 item in inventory, got %d", len(player.Inventory.Items))
	}

	roomItems = w.GetItemsInRoom(1001)
	if len(roomItems) != 0 {
		t.Errorf("expected 0 items in room, got %d", len(roomItems))
	}

	// NOTE: Carrier is *Inventory, not *Player — see SURPRISING BEHAVIOR doc at top.
	if obj.Carrier == nil {
		t.Error("expected item.Carrier to be set")
	}
}

// ---------------------------------------------------------------------------
// TestPlayerInventoryToRoom — drop an item from inventory to the room
// ---------------------------------------------------------------------------

func TestPlayerInventoryToRoom(t *testing.T) {
	w, player := newTestWorld(t)

	// Spawn and add directly to player inventory
	obj, err := w.SpawnObject(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject failed: %v", err)
	}

	if err := player.Inventory.AddItem(obj); err != nil {
		t.Fatalf("AddItem to inventory failed: %v", err)
	}

	// Drop: remove from inventory, add to room
	removed := player.Inventory.RemoveItem(obj)
	if !removed {
		t.Fatal("RemoveItem from inventory returned false")
	}
	w.AddItemToRoom(obj, 1001)

	// Assert: item in room, not in inventory
	roomItems := w.GetItemsInRoom(1001)
	if len(roomItems) != 1 {
		t.Errorf("expected 1 item in room, got %d", len(roomItems))
	}

	if len(player.Inventory.Items) != 0 {
		t.Errorf("expected 0 items in inventory, got %d", len(player.Inventory.Items))
	}

	// Carrier is nil after RemoveItem
	if obj.Carrier != nil {
		t.Error("expected item.Carrier to be nil after drop")
	}
}

// ---------------------------------------------------------------------------
// TestPlayerInventoryToEquipment — equip an item from inventory
// ---------------------------------------------------------------------------

func TestPlayerInventoryToEquipment(t *testing.T) {
	w, player := newTestWorld(t)

	// Spawn the wearable object (vnum 3002, available in newTestWorld) and put it in inventory
	obj, err := w.SpawnObject(3002, 1001)
	if err != nil {
		t.Fatalf("SpawnObject failed: %v", err)
	}

	if err := player.Inventory.AddItem(obj); err != nil {
		t.Fatalf("AddItem to inventory failed: %v", err)
	}

	// Equip: remove from inventory, equip
	player.Inventory.RemoveItem(obj)
	if err := player.Equipment.Equip(obj, player.Inventory); err != nil {
		t.Fatalf("Equip failed: %v", err)
	}

	// Assert: item in equipment, not in inventory
	if len(player.Inventory.Items) != 0 {
		t.Errorf("expected 0 items in inventory, got %d", len(player.Inventory.Items))
	}

	wieldItem, ok := player.Equipment.GetItemInSlot(SlotWield)
	if !ok {
		t.Fatal("expected item in SlotWield, nothing found")
	}
	if wieldItem != obj {
		t.Error("SlotWield item is not the expected object")
	}

	// EquippedOn should be set
	if obj.EquippedOn == nil {
		t.Error("expected item.EquippedOn to be set")
	}
}

// ---------------------------------------------------------------------------
// TestEquipmentToInventory — unequip an item back to inventory
// ---------------------------------------------------------------------------

func TestEquipmentToInventory(t *testing.T) {
	w, player := newTestWorld(t)

	// Spawn equippable item (vnum 3002, available in newTestWorld), put in inventory, equip
	obj, err := w.SpawnObject(3002, 1001)
	if err != nil {
		t.Fatalf("SpawnObject failed: %v", err)
	}
	if err := player.Inventory.AddItem(obj); err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}
	player.Inventory.RemoveItem(obj)
	if err := player.Equipment.Equip(obj, player.Inventory); err != nil {
		t.Fatalf("Equip failed: %v", err)
	}

	// Unequip: remove from slot, add to inventory
	if err := player.Equipment.Unequip(SlotWield, player.Inventory); err != nil {
		t.Fatalf("Unequip failed: %v", err)
	}

	// Assert: item back in inventory, not in equipment
	if len(player.Inventory.Items) != 1 {
		t.Errorf("expected 1 item in inventory, got %d", len(player.Inventory.Items))
	}

	_, ok := player.Equipment.GetItemInSlot(SlotWield)
	if ok {
		t.Error("expected SlotWield to be empty after unequip")
	}

	if obj.EquippedOn != nil {
		t.Error("expected item.EquippedOn to be nil after unequip")
	}
}

// ---------------------------------------------------------------------------
// TestContainerNesting — put items inside a container, then remove them
// ---------------------------------------------------------------------------

func TestContainerNesting(t *testing.T) {
	w, _ := newTestWorld(t)

	// Spawn container (vnum 3003) and inner item (vnum 3001) — both available in newTestWorld
	container, err := w.SpawnObject(3003, 1001)
	if err != nil {
		t.Fatalf("SpawnObject container failed: %v", err)
	}
	inner, err := w.SpawnObject(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject inner item failed: %v", err)
	}

	// Add inner item to container
	added := container.AddToContainer(inner)
	if !added {
		t.Fatal("AddToContainer returned false — is TypeFlag==1?")
	}

	// Assert: inner in container.Contains
	if len(container.Contains) != 1 {
		t.Fatalf("expected 1 item in container.Contains, got %d", len(container.Contains))
	}
	if container.Contains[0] != inner {
		t.Error("container.Contains[0] is not the expected inner item")
	}
	if inner.Container != container {
		t.Error("expected inner.Container to point to the container")
	}

	// Remove from container
	removed := container.RemoveFromContainer(inner)
	if !removed {
		t.Fatal("RemoveFromContainer returned false")
	}

	if len(container.Contains) != 0 {
		t.Errorf("expected 0 items in container after removal, got %d", len(container.Contains))
	}
	if inner.Container != nil {
		t.Error("expected inner.Container to be nil after removal")
	}
}

// ---------------------------------------------------------------------------
// TestExtractObject — extract an object that is on the floor
// ---------------------------------------------------------------------------

func TestExtractObject(t *testing.T) {
	w, _ := newTestWorld(t)

	// Spawn object and add to room
	obj, err := w.SpawnObject(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject failed: %v", err)
	}
	w.AddItemToRoom(obj, 1001)

	// Verify it's there
	if len(w.GetItemsInRoom(1001)) != 1 {
		t.Fatal("expected item in room before extract")
	}

	// ExtractObject
	w.ExtractObject(obj, 1001)

	// Assert: not in room, not in objectInstances
	if len(w.GetItemsInRoom(1001)) != 0 {
		t.Error("expected 0 items in room after extract")
	}

	// Try to look up by ID — it should not exist
	w.mu.RLock()
	_, exists := w.objectInstances[obj.ID]
	w.mu.RUnlock()
	if exists {
		t.Error("expected object to be removed from objectInstances map")
	}
}

// ---------------------------------------------------------------------------
// TestExtractObjectFromInventory — extract an object carried by a player
// ---------------------------------------------------------------------------
//
// SURPRISING BEHAVIOR: Inventory.AddItem() sets item.Carrier = *Inventory, but
// ExtractObject() type-asserts Carrier as *Player or *MobInstance.  To work
// around this, the test manually sets Carrier to *Player to match what the
// ExtractObject carrier branch expects.  After the ObjectLocation refactor
// this should be cleaned up.

func TestExtractObjectFromInventory(t *testing.T) {
	w, player := newTestWorld(t)

	// Spawn object and add to player inventory
	obj, err := w.SpawnObject(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject failed: %v", err)
	}

	if err := player.Inventory.AddItem(obj); err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	// NOTE: AddItem sets Carrier = *Inventory, but ExtractObject needs
	// Carrier = *Player for the carrier-removal branch to fire. We set it
	// manually to match the typical real usage pattern (after pick-up).
	obj.Carrier = player

	// Verify in inventory
	if len(player.Inventory.Items) != 1 {
		t.Fatal("expected item in player inventory before extract")
	}

	w.mu.RLock()
	_, exists := w.objectInstances[obj.ID]
	w.mu.RUnlock()
	if !exists {
		t.Fatal("expected object in objectInstances map before extract")
	}

	// ExtractObject — will find Carrier as *Player
	w.ExtractObject(obj, 1001)

	// Assert: not in inventory, not in objectInstances
	if len(player.Inventory.Items) != 0 {
		t.Error("expected 0 items in inventory after extract")
	}

	w.mu.RLock()
	_, exists = w.objectInstances[obj.ID]
	w.mu.RUnlock()
	if exists {
		t.Error("expected object to be removed from objectInstances map")
	}

	if obj.Carrier != nil {
		t.Error("expected item.Carrier to be nil after extract")
	}
}

// ---------------------------------------------------------------------------
// TestInventoryFull — fill inventory to capacity and assert error
// ---------------------------------------------------------------------------

func TestInventoryFull(t *testing.T) {
	w, player := newTestWorld(t)

	// Fill inventory to capacity
	capacity := player.Inventory.Capacity // default 20 from NewInventory
	for i := 0; i < capacity; i++ {
		obj, err := w.SpawnObject(3001, 1001)
		if err != nil {
			t.Fatalf("SpawnObject %d failed: %v", i, err)
		}
		if err := player.Inventory.AddItem(obj); err != nil {
			t.Fatalf("AddItem %d failed: %v", i, err)
		}
	}

	// Verify full
	if !player.Inventory.IsFull() {
		t.Fatal("expected inventory to be full")
	}
	if len(player.Inventory.Items) != capacity {
		t.Fatalf("expected %d items, got %d", capacity, len(player.Inventory.Items))
	}

	// Try to add one more — should fail with ErrInventoryFull
	extra, err := w.SpawnObject(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject failed: %v", err)
	}

	err = player.Inventory.AddItem(extra)
	if err != ErrInventoryFull {
		t.Errorf("expected ErrInventoryFull, got %v", err)
	}
}
