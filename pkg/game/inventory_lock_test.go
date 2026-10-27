package game

import (
	"sync"
	"testing"
)

// TestInventoryAccessIsLockedAgainstMoves hammers the object-movement writers
// against the inventory readers a player's own commands use.
//
// MoveObject's detach/attach/extract arms mutated Inventory through the no-lock
// internals holding only w.mu, while readers took inv.mu — so the reader's
// RLock excluded nothing. A give, drop, put or corpse move concurrent with
// `inventory`, `look`, a reagent consume or a boat check could therefore
// observe a torn item slice (dropped or duplicated items). Run under -race.
func TestInventoryAccessIsLockedAgainstMoves(t *testing.T) {
	w, a := newFlagWorld(t)
	b := NewPlayer(2, "Bee", 1001)
	if err := w.AddPlayer(b); err != nil {
		t.Fatalf("AddPlayer b: %v", err)
	}
	obj := newFlagItem(w, 7500, 1)
	if err := w.MoveObjectToPlayerInventory(obj, a); err != nil {
		t.Fatalf("seat: %v", err)
	}

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
				// Every reader shape a player command uses.
				_ = a.Inventory.Snapshot()
				_ = a.Inventory.GetItemCount()
				_, _ = a.Inventory.FindItem("thing")
				_ = a.Inventory.GetWeight()
				_ = a.HasBoat()
				w.DoInventory(a)
			}
		}()
	}

	for i := 0; i < 3000; i++ {
		dst := b
		if i%2 == 1 {
			dst = a
		}
		if err := w.MoveObject(obj, LocInventoryPlayer(dst.Name)); err != nil {
			close(stop)
			readers.Wait()
			t.Fatalf("iteration %d: move: %v", i, err)
		}
	}
	close(stop)
	readers.Wait()

	// The object must live in exactly one inventory at the end.
	holders := 0
	for _, p := range []*Player{a, b} {
		if invHolds(p, obj) {
			holders++
		}
	}
	if holders != 1 {
		t.Fatalf("object reachable from %d inventories, want exactly 1", holders)
	}
}

// invHolds reports whether p's inventory names exactly this instance.
func invHolds(p *Player, obj *ObjectInstance) bool {
	for _, it := range p.Inventory.Snapshot() {
		if it == obj {
			return true
		}
	}
	return false
}
