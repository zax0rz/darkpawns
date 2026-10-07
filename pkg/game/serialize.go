package game

import (
	"encoding/json"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// InventoryData represents serialized inventory data.
type InventoryData struct {
	ItemVnums []int `json:"item_vnums"`
	Capacity  int   `json:"capacity"`
}

// EquipmentData represents serialized equipment data.
type EquipmentData struct {
	Slots map[string]int `json:"slots"` // slot name -> item vnum
}

// SerializeInventory converts inventory to JSON bytes.
func SerializeInventory(inv *Inventory, worldObjs map[int]*parser.Obj) ([]byte, error) {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	data := InventoryData{
		Capacity: inv.Capacity,
	}

	// Store only VNums to save space
	for _, item := range inv.Items {
		data.ItemVnums = append(data.ItemVnums, item.VNum)
	}

	return json.Marshal(data)
}

// SerializeEquipment converts equipment to JSON bytes.
func SerializeEquipment(eq *Equipment, worldObjs map[int]*parser.Obj) ([]byte, error) {
	eq.mu.RLock()
	defer eq.mu.RUnlock()

	data := EquipmentData{
		Slots: make(map[string]int),
	}

	// Store slot -> VNum mapping
	for slot, item := range eq.Slots {
		data.Slots[slot.String()] = item.VNum
	}

	return json.Marshal(data)
}
