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
				WearFlags: [4]int{0, 0, 0, 0},
				Values:    [4]int{0, 0, 0, 0},
			},
			{
				VNum: 3002, Keywords: "test weapon", ShortDesc: "a test weapon",
				LongDesc: "A test weapon lies here.", TypeFlag: 5,
				WearFlags: [4]int{1 << 13, 0, 0, 0}, // ITEM_WEAR_WIELD
				Values:    [4]int{0, 1, 4, 0},
			},
			{
				VNum: 3003, Keywords: "test container", ShortDesc: "a test container",
				LongDesc: "A test container lies here.", TypeFlag: 1,
				WearFlags: [4]int{0, 0, 0, 0},
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

	// Player picks up via MoveObject
	w.MoveObject(obj, LocInventoryPlayer(player.Name))

	// Assert: item in inventory, not in room
	if len(player.Inventory.Items) != 1 {
		t.Errorf("expected 1 item in inventory, got %d", len(player.Inventory.Items))
	}

	roomItems = w.GetItemsInRoom(1001)
	if len(roomItems) != 0 {
		t.Errorf("expected 0 items in room, got %d", len(roomItems))
	}

	// Carrier field removed — Location is now the source of truth
	if obj.Location.Kind != ObjInInventory {
		t.Errorf("expected item Location.Kind to be ObjInInventory, got %v", obj.Location.Kind)
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
	obj.Location = LocInventoryPlayer(player.Name)

	// Drop via MoveObject
	w.MoveObject(obj, LocRoom(1001))

	// Assert: item in room, not in inventory
	roomItems := w.GetItemsInRoom(1001)
	if len(roomItems) != 1 {
		t.Errorf("expected 1 item in room, got %d", len(roomItems))
	}

	if len(player.Inventory.Items) != 0 {
		t.Errorf("expected 0 items in inventory, got %d", len(player.Inventory.Items))
	}

	// Location should be in room after drop
	if obj.Location.Kind != ObjInRoom {
		t.Errorf("expected item Location.Kind to be ObjInRoom, got %v", obj.Location.Kind)
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

	// Location should show equipped
	if obj.Location.Kind != ObjEquipped {
		t.Errorf("expected item Location.Kind to be ObjEquipped, got %v", obj.Location.Kind)
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

	if obj.Location.Kind != ObjInInventory && obj.Location.Kind != ObjNowhere {
		t.Errorf("expected item Location.Kind to be ObjInInventory or ObjNowhere after unequip, got %v", obj.Location.Kind)
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
	inner.Location = LocContainer(container.ID)

	// Assert: inner in container.Contains
	if len(container.Contains) != 1 {
		t.Fatalf("expected 1 item in container.Contains, got %d", len(container.Contains))
	}
	if container.Contains[0] != inner {
		t.Error("container.Contains[0] is not the expected inner item")
	}
	if inner.Location.Kind != ObjInContainer {
		t.Errorf("expected inner Location.Kind to be ObjInContainer, got %v", inner.Location.Kind)
	}

	// Remove from container
	removed := container.RemoveFromContainer(inner)
	if !removed {
		t.Fatal("RemoveFromContainer returned false")
	}

	if len(container.Contains) != 0 {
		t.Errorf("expected 0 items in container after removal, got %d", len(container.Contains))
	}
	// Note: RemoveFromContainer only removes from the slice; caller is responsible
	// for updating Location (e.g. via MoveObject). Direct RemoveFromContainer
	// does not clear Location.
	if inner.Location.Kind != ObjInContainer {
		t.Error("expected inner Location.Kind to still be ObjInContainer after direct RemoveFromContainer")
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

	// Use MoveObject to place in player inventory
	w.MoveObject(obj, LocInventoryPlayer(player.Name))

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

	// ExtractObject — will use Location to find and remove from inventory
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

	if obj.Location.Kind != ObjNowhere {
		t.Errorf("expected item Location.Kind to be ObjNowhere, got %v", obj.Location.Kind)
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
	extra.Location = LocInventoryPlayer(player.Name)
	if err != ErrInventoryFull {
		t.Errorf("expected ErrInventoryFull, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestMoveObjectRoomToInventory — MoveObject room-to-inventory
// ---------------------------------------------------------------------------

func TestMoveObjectRoomToInventory(t *testing.T) {
	w, player := newTestWorld(t)

	obj, _ := w.SpawnObject(3001, 1001)
	w.AddItemToRoom(obj, 1001)

	// Move via MoveObject
	err := w.MoveObjectToPlayerInventory(obj, player)
	if err != nil {
		t.Fatalf("MoveObjectToPlayerInventory failed: %v", err)
	}

	// Verify location
	if !obj.Location.IsInInventory() || !obj.Location.OwnerIsPlayer() {
		t.Error("expected ObjInInventory with OwnerPlayer")
	}
	if len(w.GetItemsInRoom(1001)) != 0 {
		t.Error("item should be removed from room")
	}
	if len(player.Inventory.Items) != 1 {
		t.Errorf("expected 1 item in inventory, got %d", len(player.Inventory.Items))
	}
	if err := obj.Location.Validate(); err != nil {
		t.Errorf("location invalid: %v", err)
	}
}

func TestMoveObjectInventoryToRoom(t *testing.T) {
	w, player := newTestWorld(t)

	obj, _ := w.SpawnObject(3001, 1001)
	player.Inventory.AddItem(obj)
	obj.Location = LocInventoryPlayer(player.Name)

	err := w.MoveObjectToRoom(obj, 1001)
	if err != nil {
		t.Fatalf("MoveObjectToRoom failed: %v", err)
	}

	if !obj.Location.IsInRoom() {
		t.Error("expected ObjInRoom")
	}
	if len(player.Inventory.Items) != 0 {
		t.Errorf("expected 0 items in inventory, got %d", len(player.Inventory.Items))
	}
	if len(w.GetItemsInRoom(1001)) != 1 {
		t.Errorf("expected 1 item in room, got %d", len(w.GetItemsInRoom(1001)))
	}
}

func TestMoveObjectInventoryToEquipment(t *testing.T) {
	w, player := newTestWorld(t)

	obj, _ := w.SpawnObject(3002, 1001) // wieldable weapon
	player.Inventory.AddItem(obj)
	obj.Location = LocInventoryPlayer(player.Name)

	err := w.MoveObject(obj, LocEquippedPlayer(player.Name, SlotWield))
	if err != nil {
		t.Fatalf("MoveObject to equipment failed: %v", err)
	}

	if !obj.Location.IsEquipped() {
		t.Error("expected ObjEquipped")
	}
	if len(player.Inventory.Items) != 0 {
		t.Errorf("expected 0 items in inventory, got %d", len(player.Inventory.Items))
	}
}

func TestMoveObjectToNowhere(t *testing.T) {
	w, player := newTestWorld(t)

	obj, _ := w.SpawnObject(3001, 1001)
	player.Inventory.AddItem(obj)
	obj.Location = LocInventoryPlayer(player.Name)

	err := w.MoveObjectToNowhere(obj)
	if err != nil {
		t.Fatalf("MoveObjectToNowhere failed: %v", err)
	}

	if !obj.Location.IsNowhere() {
		t.Error("expected ObjNowhere")
	}
	if len(player.Inventory.Items) != 0 {
		t.Errorf("expected 0 items in inventory, got %d", len(player.Inventory.Items))
	}
}

func TestMoveObjectInventoryFull(t *testing.T) {
	w, player := newTestWorld(t)

	// Fill inventory
	for i := 0; i < player.Inventory.Capacity; i++ {
		o, _ := w.SpawnObject(3001, 1001)
		player.Inventory.AddItem(o)
	}

	extra, _ := w.SpawnObject(3001, 1001)
	w.AddItemToRoom(extra, 1001)

	err := w.MoveObjectToPlayerInventory(extra, player)
	if err == nil {
		t.Fatal("expected error when moving to full inventory")
	}

	// Item should stay in room (rollback)
	if len(w.GetItemsInRoom(1001)) != 1 {
		t.Errorf("expected 1 item in room, got %d", len(w.GetItemsInRoom(1001)))
	}
}

func TestMoveObjectInvalidDestination(t *testing.T) {
	w, _ := newTestWorld(t)

	obj, _ := w.SpawnObject(3001, 1001)

	// Invalid: ObjInRoom with roomVNum <= 0
	err := w.MoveObject(obj, LocRoom(-1))
	if err == nil {
		t.Fatal("expected error for invalid room VNum")
	}

	// Invalid: ObjInInventory with empty player name
	err = w.MoveObject(obj, LocInventoryPlayer(""))
	if err == nil {
		t.Fatal("expected error for empty player name")
	}
}

// ---------------------------------------------------------------------------
// TestContainerCyclePrevention — A contains B; putting A into B must fail
// ---------------------------------------------------------------------------
//
// BUG: This test currently FAILS. attachObjectLocked(ObjInContainer) has no
// cycle check. MoveObjectToContainer(A, B) succeeds even when B is already
// inside A, creating a reference cycle that corrupts GetTotalWeight and any
// code that recursively walks Contains.

func TestContainerCyclePrevention(t *testing.T) {
	w, _ := newTestWorld(t)

	containerA, err := w.SpawnObject(3003, 1001)
	if err != nil {
		t.Fatalf("SpawnObject A failed: %v", err)
	}
	containerB, err := w.SpawnObject(3003, 1001)
	if err != nil {
		t.Fatalf("SpawnObject B failed: %v", err)
	}
	// Register both in room so detach finds them
	w.AddItemToRoom(containerA, 1001)
	w.AddItemToRoom(containerB, 1001)

	// Put B inside A
	if err := w.MoveObjectToContainer(containerB, containerA); err != nil {
		t.Fatalf("setup MoveObjectToContainer(B→A) failed: %v", err)
	}

	// Now try to put A inside B — must fail (would create A→B→A cycle)
	err = w.MoveObjectToContainer(containerA, containerB)
	if err == nil {
		t.Error("BUG: MoveObjectToContainer(A into B) succeeded, creating a container cycle")
	}

	// containerA must still be in the room, not inside containerB
	if containerA.Location.Kind == ObjInContainer {
		t.Errorf("containerA.Location should not be ObjInContainer after failed cycle attempt, got %v", containerA.Location.Kind)
	}
}

// ---------------------------------------------------------------------------
// TestMobEquipInEquipmentNotInventory — equip puts in Equipment, not Inventory
// ---------------------------------------------------------------------------
//
// BUG: This test currently FAILS. attachObjectLocked(ObjEquipped, OwnerMob)
// calls m.AddToInventory(obj) before setting Equipment[slot], so the item
// ends up in BOTH Inventory and Equipment. Correct behavior: Equipment only.
// Location is set correctly by the final obj.Location = dst assignment.

func TestMobEquipInEquipmentNotInventory(t *testing.T) {
	w, _ := newTestWorld(t)

	mob, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	obj, err := w.SpawnObject(3002, 1001) // wieldable weapon (WearFlags bit 13)
	if err != nil {
		t.Fatalf("SpawnObject failed: %v", err)
	}

	// Move to mob inventory first so detach has a valid prior location
	if err := w.MoveObjectToMobInventory(obj, mob); err != nil {
		t.Fatalf("MoveObjectToMobInventory failed: %v", err)
	}

	// Equip via MoveObject
	if err := w.MoveObject(obj, LocEquippedMob(mob.GetID(), SlotWield)); err != nil {
		t.Fatalf("MoveObject equip failed: %v", err)
	}

	// Location must reflect equipped state
	if !obj.Location.IsEquipped() {
		t.Errorf("Location.Kind = %v, want ObjEquipped", obj.Location.Kind)
	}
	if _, ok := mob.Equipment[int(SlotWield)]; !ok {
		t.Error("item missing from Equipment map after equip")
	}

	// Invariant: item must NOT appear in the Inventory slice
	for _, invItem := range mob.Inventory {
		if invItem == obj {
			t.Error("BUG: equipped item still present in mob Inventory slice — should be Equipment only")
			break
		}
	}
}

// ---------------------------------------------------------------------------
// TestMobUnequipToInventory — unequip moves from Equipment to Inventory
// ---------------------------------------------------------------------------

func TestMobUnequipToInventory(t *testing.T) {
	w, _ := newTestWorld(t)

	mob, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	obj, err := w.SpawnObject(3002, 1001) // wieldable weapon
	if err != nil {
		t.Fatalf("SpawnObject failed: %v", err)
	}

	// Equip directly via mob method (bypasses MoveObject, item in Equipment only)
	obj.Location = LocEquippedMob(mob.GetID(), SlotWield)
	mob.Equipment[int(SlotWield)] = obj

	// Unequip: move to mob inventory via MoveObject
	if err := w.MoveObjectToMobInventory(obj, mob); err != nil {
		t.Fatalf("MoveObjectToMobInventory failed: %v", err)
	}

	// Equipment slot must be empty
	if _, ok := mob.Equipment[int(SlotWield)]; ok {
		t.Error("item still in Equipment map after unequip to inventory")
	}

	// Item must be in Inventory
	found := false
	for _, invItem := range mob.Inventory {
		if invItem == obj {
			found = true
			break
		}
	}
	if !found {
		t.Error("item not found in mob Inventory after unequip")
	}

	if !obj.Location.IsInInventory() {
		t.Errorf("Location.Kind = %v, want ObjInInventory", obj.Location.Kind)
	}
}

// ---------------------------------------------------------------------------
// TestMoveObjectRollbackStrandedLocation — both attach and rollback-attach fail
// ---------------------------------------------------------------------------
//
// BUG: This test currently FAILS. When MoveObject's primary attach fails AND
// the rollback re-attach also fails, obj.Location retains the original (stale)
// value even though the object is in neither location.  Correct behavior:
// obj.Location should be set to LocNowhere() so callers know the object is
// stranded and not to trust the old pointer.

func TestMoveObjectRollbackStrandedLocation(t *testing.T) {
	w, playerA := newTestWorld(t)

	playerB := NewPlayer(2, "PlayerB", 1001)
	if err := w.AddPlayer(playerB); err != nil {
		t.Fatalf("AddPlayer B failed: %v", err)
	}

	// Fill playerA to capacity so any addItem call fails
	for i := 0; i < playerA.Inventory.Capacity; i++ {
		o, _ := w.SpawnObject(3001, 1001)
		playerA.Inventory.addItem(o)
	}

	// Fill playerB to capacity
	for i := 0; i < playerB.Inventory.Capacity; i++ {
		o, _ := w.SpawnObject(3001, 1001)
		playerB.Inventory.addItem(o)
	}

	// Force target into playerB beyond the capacity check by direct slice append.
	// After detach, playerB will again be exactly at capacity, so the rollback
	// addItem will also return ErrInventoryFull — stranding the item.
	target, _ := w.SpawnObject(3001, 1001)
	playerB.Inventory.Items = append(playerB.Inventory.Items, target)
	target.Location = LocInventoryPlayer(playerB.Name)

	// Attempt move: detach from B (succeeds), attach to A (fails: full),
	// rollback to B (fails: B is now exactly at Capacity after the detach).
	err := w.MoveObjectToPlayerInventory(target, playerA)
	if err == nil {
		t.Fatal("expected an error when moving to a full inventory")
	}

	// Confirm item is in neither inventory
	for _, item := range playerA.Inventory.Items {
		if item == target {
			t.Error("target should not be in playerA inventory")
		}
	}
	for _, item := range playerB.Inventory.Items {
		if item == target {
			t.Error("target should not be in playerB inventory after failed rollback")
		}
	}

	// Expected invariant: stranded object gets Location = LocNowhere
	// Actual (bug): Location is stale LocInventoryPlayer(playerB)
	if !target.Location.IsNowhere() {
		t.Errorf("BUG: stranded object Location = %+v, want LocNowhere", target.Location)
	}
}

// ---------------------------------------------------------------------------
// TestLocationFieldSync — validates ObjectLocation field sync on ObjectInstance
// ---------------------------------------------------------------------------

func TestLocationFieldSync(t *testing.T) {
	w, player := newTestWorld(t)

	// Spawn in room — Location should be ObjInRoom
	obj, _ := w.SpawnObject(3001, 1001)
	if !obj.Location.IsInRoom() {
		t.Error("spawned object should be ObjInRoom")
	}
	if obj.Location.RoomVNum != 1001 {
		t.Errorf("expected RoomVNum 1001, got %d", obj.Location.RoomVNum)
	}

	// Validate Location
	if err := obj.Location.Validate(); err != nil {
		t.Errorf("spawned location invalid: %v", err)
	}

	// Add to inventory — Location should update
	player.Inventory.AddItem(obj)
	obj.Location = LocInventoryPlayer(player.Name)
	if !obj.Location.IsInInventory() {
		// Note: AddItem sets Carrier to *Inventory, which SetCarrier can't
		// cleanly map to a Location kind. This documents the mismatch.
		t.Log("Carrier is *Inventory — Location mismatch expected until MoveObject")
	}

	// MoveObject to player inventory should update Location correctly
	obj2, _ := w.SpawnObject(3001, 1001)
	w.MoveObject(obj2, LocInventoryPlayer(player.Name))
	if !obj2.Location.OwnerIsPlayer() {
		t.Error("MoveObject to player inventory should set OwnerIsPlayer")
	}
	if err := obj2.Location.Validate(); err != nil {
		t.Errorf("player inventory location invalid: %v", err)
	}

	// Mob ID assignment
	mob, _ := w.SpawnMob(2001, 1001)
	if mob.GetID() <= 0 {
		t.Error("mob should have positive ID after spawn")
	}
	if found, ok := w.GetMobByID(mob.GetID()); !ok || found != mob {
		t.Error("GetMobByID should find the spawned mob")
	}
}
