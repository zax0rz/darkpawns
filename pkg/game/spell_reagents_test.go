package game

import "testing"

func TestSpawnObjectToPlayerInventoryIUsesObjToCharPlacement(t *testing.T) {
	w, player := newTestWorld(t)

	value, err := w.SpawnObjectToPlayerInventoryI(3001, player)
	if err != nil {
		t.Fatalf("SpawnObjectToPlayerInventoryI: %v", err)
	}
	obj, ok := value.(*ObjectInstance)
	if !ok {
		t.Fatalf("spawned value type = %T, want *ObjectInstance", value)
	}
	if got := obj.Location; got != LocInventoryPlayer(player.Name) {
		t.Errorf("spawned object location = %+v, want player inventory", got)
	}
	if got := player.Inventory.Snapshot(); len(got) != 1 || got[0] != obj {
		t.Errorf("player inventory = %v, want spawned object", got)
	}
	if got := w.GetItemsInRoom(player.GetRoomVNum()); len(got) != 0 {
		t.Errorf("spawned object also remains in room: %v", got)
	}
}

func TestSpawnObjectToRoomForSpellUsesObjToRoomPlacement(t *testing.T) {
	w, _ := newTestWorld(t)

	value, err := w.SpawnObjectToRoomForSpell(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnObjectToRoomForSpell: %v", err)
	}
	obj, ok := value.(*ObjectInstance)
	if !ok {
		t.Fatalf("spawned value type = %T, want *ObjectInstance", value)
	}
	if got := obj.Location; got != LocRoom(1001) {
		t.Errorf("spawned object location = %+v, want room 1001", got)
	}
	if got := w.GetItemsInRoom(1001); len(got) != 1 || got[0] != obj {
		t.Errorf("room contents = %v, want spawned object", got)
	}
}

func TestConsumeSpellReagentUsesObjectLocationPipeline(t *testing.T) {
	w, player := newTestWorld(t)
	obj, err := w.SpawnObject(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	obj.Prototype.Keywords = "shard obsidian"
	obj.Prototype.ShortDesc = "a shard of obsidian"
	if err := w.MoveObjectToPlayerInventory(obj, player); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory: %v", err)
	}

	if !player.ConsumeSpellReagent("shard of obsidian") {
		t.Fatal("ConsumeSpellReagent returned false for carried component")
	}
	if player.Inventory.GetItemCount() != 0 {
		t.Errorf("inventory still contains consumed reagent: %+v", player.Inventory.Snapshot())
	}
	if !obj.Location.IsNowhere() {
		t.Errorf("consumed reagent location = %+v, want LocNowhere", obj.Location)
	}
}

func TestConsumeSpellReagentVNumUsesExactPrototype(t *testing.T) {
	w, player := newTestWorld(t)
	obj, err := w.SpawnObject(3001, 1001)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	obj.Prototype.VNum = 1226
	obj.VNum = 1226
	if err := w.MoveObjectToPlayerInventory(obj, player); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory: %v", err)
	}

	if !player.ConsumeSpellReagentVNum(1226) {
		t.Fatal("ConsumeSpellReagentVNum returned false for exact carried component")
	}
	if player.Inventory.GetItemCount() != 0 {
		t.Errorf("inventory still contains consumed reagent: %+v", player.Inventory.Snapshot())
	}
}
