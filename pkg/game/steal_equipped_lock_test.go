package game

import (
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// TestStealEquippedTransferHoldsWorldLock proves the equipped steal's transfer
// runs inside w.mu.
//
// stealEquippedItem changes the victim's equipment slot, both inventories and
// the object's Location. Every other object transfer (give/put/drop/corpse)
// runs under w.mu through MoveObject, so an unlocked equipped steal could have
// a concurrent MoveObject detach through the by-then-stale Location (a silent
// no-op — the victim's equipment no longer held it) and attach the same
// *ObjectInstance to a second inventory, after which the steal added it again:
// one instance reachable from two lists, persisted as two items.
//
// Deterministic: the test holds w.mu itself and asserts the steal cannot
// complete until it is released. The pre-fix code never takes w.mu here, so it
// finishes while the lock is held.
func TestStealEquippedTransferHoldsWorldLock(t *testing.T) {
	w, thief := newFlagWorld(t)
	victim := NewPlayer(2, "Victimus", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	item := newFlagItem(w, 7160, (1<<0)|(1<<1)) // TAKE + FINGER
	if err := w.MoveObjectToPlayerInventory(item, victim); err != nil {
		t.Fatalf("carry: %v", err)
	}
	if err := w.MoveObject(item, LocEquippedPlayer(victim.Name, SlotFingerR)); err != nil {
		t.Fatalf("wear: %v", err)
	}
	victim.SetPosition(combat.PosSleeping)
	thief.Level = LVL_IMPL // percent stays below the roll: the steal succeeds
	thief.SetSkill(SkillSteal, 100)
	thief.SetPlrFlag(PlrOutlaw, true)
	thief.Stats.Str = 15
	thief.CopyBaseAttributes()

	w.mu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		stealEquippedItem(w, thief, victim, item, int(SlotFingerR), -1,
			GetPronouns(thief.Name, thief.GetSex()), GetPronouns(victim.Name, victim.GetSex()))
	}()

	select {
	case <-done:
		w.mu.Unlock()
		t.Fatal("equipped steal completed while w.mu was held: the transfer is not world-locked")
	case <-time.After(150 * time.Millisecond):
	}
	w.mu.Unlock()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("equipped steal did not complete after w.mu was released")
	}
	if !holdsObject(thief, item) {
		t.Fatal("equipped steal did not move the item to the thief")
	}
	if holdsObject(victim, item) {
		t.Fatal("equipped steal left the item with the victim")
	}
}

// TestStealEquippedConcurrentMoveKeepsOneCopy races the equipped steal against
// a MoveObject of the same instance. Both mutate the object's Location and the
// inventories, so an unlocked transfer could leave one *ObjectInstance
// reachable from two lists. The invariant must hold every round.
func TestStealEquippedConcurrentMoveKeepsOneCopy(t *testing.T) {
	w, thief := newFlagWorld(t)
	victim := NewPlayer(2, "Victimus", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	other := NewPlayer(3, "Other", 1001)
	if err := w.AddPlayer(other); err != nil {
		t.Fatalf("AddPlayer other: %v", err)
	}
	thief.Level = LVL_IMPL
	thief.SetSkill(SkillSteal, 100)
	thief.SetPlrFlag(PlrOutlaw, true)
	thief.Stats.Str = 15
	thief.CopyBaseAttributes()

	item := newFlagItem(w, 7161, (1<<0)|(1<<1))
	for round := 0; round < 150; round++ {
		// Reset: nothing carried, item worn by the sleeping victim.
		for _, p := range []*Player{thief, victim, other} {
			for _, leftover := range p.Inventory.Snapshot() {
				if err := w.MoveObject(leftover, LocRoom(1001)); err != nil {
					t.Fatalf("round %d: clear %s: %v", round, p.Name, err)
				}
			}
		}
		if err := w.MoveObject(item, LocEquippedPlayer(victim.Name, SlotFingerR)); err != nil {
			t.Fatalf("round %d: re-wear: %v", round, err)
		}
		victim.SetPosition(combat.PosSleeping)

		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			stealEquippedItem(w, thief, victim, item, int(SlotFingerR), -1,
				GetPronouns(thief.Name, thief.GetSex()), GetPronouns(victim.Name, victim.GetSex()))
		}()
		go func() {
			defer wg.Done()
			<-start
			_ = w.MoveObject(item, LocInventoryPlayer(other.Name))
		}()
		close(start)
		wg.Wait()

		holders := 0
		for _, p := range []*Player{thief, victim, other} {
			if holdsObject(p, item) {
				holders++
			}
		}
		if holders > 1 {
			t.Fatalf("round %d: one object reachable from %d inventories (duplication)", round, holders)
		}
		if _, worn := victim.Equipment.GetItemInSlot(SlotFingerR); worn && holders == 0 {
			t.Fatalf("round %d: object reported worn by the victim but held by nobody", round)
		}
	}
}
