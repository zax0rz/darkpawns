package game

import (
	"sync"
	"testing"
)

// TestEquipmentAccessIsLockedAgainstMutations hammers the equipment writers
// against every reader shape the game uses.
//
// Equip/Unequip/SetSlot mutated eq.slots under eq.mu, but several readers
// iterated the map raw — has_boat, findItemByName, the affect-modifier sum,
// the score display and the admin detail view (the last was VULN-028). A raw
// map iteration concurrent with a locked write is a fatal "concurrent map
// iteration and map write", and the map is touched on ordinary commands
// (`equipment`, `score`, stepping into water, wielding a weapon). Run under
// -race.
func TestEquipmentAccessIsLockedAgainstMutations(t *testing.T) {
	w, p := newFlagWorld(t)
	if p.Equipment == nil {
		p.Equipment = NewEquipment()
	}
	p.Equipment.OwnerName = p.Name
	item := newFlagItem(w, 7600, 1)

	stop := make(chan struct{})
	var readers sync.WaitGroup
	for i := 0; i < 4; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_ = p.Equipment.Snapshot()
				_ = p.Equipment.GetEquippedItems()
				_, _ = p.Equipment.GetItemInSlot(SlotWield)
				_ = p.Equipment.GetArmorClass()
				_ = p.Equipment.GetWornAffectAC()
				// The readers this change converted: each iterates the
				// equipment map, so each is a raw-iteration site pre-fix.
				_ = hasBoat(w, p)
				_ = hasKey(p, 7600)
				_, _ = findItemByName(p, "thing")
			}
		}()
	}

	for i := 0; i < 3000; i++ {
		_ = p.Equipment.SetSlot(SlotWield, item)
		_ = p.Equipment.Unequip(SlotWield, p.Inventory)
		// Unequip parks the item in the inventory; take it back out so the
		// cycle can repeat and the fixture stays a pure equip/unequip race.
		p.Inventory.RemoveItem(item)
	}
	close(stop)
	readers.Wait()

	if _, ok := p.Equipment.GetItemInSlot(SlotWield); ok {
		// Last action was Unequip; the slot must be empty.
		t.Fatal("wield slot still occupied after the final unequip")
	}
}
