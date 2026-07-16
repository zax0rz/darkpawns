package game

import "testing"

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
	if len(player.Inventory.Items) != 0 {
		t.Errorf("inventory still contains consumed reagent: %+v", player.Inventory.Items)
	}
	if !obj.Location.IsNowhere() {
		t.Errorf("consumed reagent location = %+v, want LocNowhere", obj.Location)
	}
}
