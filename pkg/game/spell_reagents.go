package game

// ConsumeSpellReagent extracts a named carried component through the canonical
// ObjectLocation movement path. The spells package discovers this narrow method
// without importing game, avoiding its old incompatible inventory interface.
func (p *Player) ConsumeSpellReagent(name string) bool {
	if p == nil || p.Inventory == nil {
		return false
	}
	item, found := p.Inventory.FindItem(name)
	if !found {
		return false
	}
	p.mu.RLock()
	w := p.worldRef
	p.mu.RUnlock()
	if w == nil {
		return false
	}
	// Consume-once: FindItem works from a snapshot, so two concurrent casts
	// both find the same reagent. Claiming it with the atomic remove is what
	// makes the check-and-take one step — otherwise one reagent buys two
	// spells, because ExtractObject is idempotent and both calls would report
	// success (CWE-362/CWE-367).
	if !p.Inventory.RemoveItem(item) {
		return false
	}
	w.ExtractObject(item, p.GetRoomVNum())
	return true
}

// ConsumeSpellReagentVNum extracts a carried component by its exact prototype
// vnum. C's has_reagents() uses the spell's material vnum, so spell paths that
// have more than a keyword-level identity can preserve that distinction.
func (p *Player) ConsumeSpellReagentVNum(vnum int) bool {
	if p == nil || p.Inventory == nil {
		return false
	}
	p.mu.RLock()
	w := p.worldRef
	p.mu.RUnlock()
	if w == nil {
		return false
	}
	for _, item := range p.Inventory.Snapshot() {
		if item != nil && item.VNum == vnum {
			// Consume-once: the snapshot above is stale for a concurrent cast,
			// so the atomic remove decides the winner (CWE-362/CWE-367).
			if !p.Inventory.RemoveItem(item) {
				return false
			}
			w.ExtractObject(item, p.GetRoomVNum())
			return true
		}
	}
	return false
}
