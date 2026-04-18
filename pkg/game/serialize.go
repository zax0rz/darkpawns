package game

import (
	"encoding/json"
	"fmt"

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

// DeserializeInventory creates inventory from JSON bytes.
func DeserializeInventory(data []byte, worldObjs map[int]*parser.Obj) (*Inventory, error) {
	var invData InventoryData
	if err := json.Unmarshal(data, &invData); err != nil {
		return nil, fmt.Errorf("unmarshal inventory: %w", err)
	}

	inv := NewInventory()
	inv.Capacity = invData.Capacity

	// Restore items from VNums
	for _, vnum := range invData.ItemVnums {
		if obj, ok := worldObjs[vnum]; ok {
			inv.Items = append(inv.Items, obj)
		} else {
			// Log warning but continue
			fmt.Printf("Warning: Object VNum %d not found in world\n", vnum)
		}
	}

	return inv, nil
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

// DeserializeEquipment creates equipment from JSON bytes.
func DeserializeEquipment(data []byte, worldObjs map[int]*parser.Obj) (*Equipment, error) {
	var eqData EquipmentData
	if err := json.Unmarshal(data, &eqData); err != nil {
		return nil, fmt.Errorf("unmarshal equipment: %w", err)
	}

	eq := NewEquipment()

	// Restore slots from VNums
	for slotName, vnum := range eqData.Slots {
		slot, ok := ParseEquipmentSlot(slotName)
		if !ok {
			fmt.Printf("Warning: Unknown equipment slot %s\n", slotName)
			continue
		}

		if obj, ok := worldObjs[vnum]; ok {
			eq.Slots[slot] = obj
		} else {
			fmt.Printf("Warning: Object VNum %d not found in world for slot %s\n", vnum, slotName)
		}
	}

	return eq, nil
}